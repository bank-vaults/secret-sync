package storesync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"sync"
	"text/template"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

// processor is used to optimally fetch secrets from a source or internal fetched map.
type processor struct {
	mu      sync.RWMutex
	source  v1alpha1.StoreReader
	fetched map[v1alpha1.SecretRef][]byte
}

func newProcessor(source v1alpha1.StoreReader) *processor {
	return &processor{
		mu:      sync.RWMutex{},
		source:  source,
		fetched: map[v1alpha1.SecretRef][]byte{},
	}
}

// GetSyncPlan fetches the data from source and applies templating based on the provided request.
// Returned map defines all secret keys (and their values) that need to be sent to target store to complete the request.
func (p *processor) GetSyncPlan(ctx context.Context, req v1alpha1.SyncRequest) (map[v1alpha1.SecretRef][]byte, error) {
	switch {
	// FromRef can only sync a single secret
	case req.FromRef != nil:
		fetchedValue, err := p.FetchFromRef(ctx, *req.FromRef)
		if err != nil {
			return nil, err
		}

		syncKey := *req.FromRef
		if req.Target.Key != nil {
			syncKey = v1alpha1.SecretRef{
				Key:     *req.Target.Key,
				Version: req.FromRef.Version,
			}
		}

		syncValue := fetchedValue
		if req.Template != nil {
			syncValue, err = getTemplatedValue(req.Template, fetchedValue)
			if err != nil {
				return nil, err
			}
		}

		return map[v1alpha1.SecretRef][]byte{
			syncKey: syncValue,
		}, nil

	// FromQuery can sync both a single secret or multiple secrets
	case req.FromQuery != nil:
		fetchedSecrets, err := p.FetchFromQuery(ctx, *req.FromQuery)
		if err != nil {
			return nil, err
		}

		// Handle FromQuery => Key
		if req.Target.Key != nil {
			syncKey := v1alpha1.SecretRef{
				Key:     *req.Target.Key,
				Version: nil,
			}

			// TODO: Fix template data accessors
			templateData := make(map[string]string)
			for key, value := range fetchedSecrets {
				templateData[key.GetProperty()] = string(value)
			}
			if req.Template == nil {
				return nil, fmt.Errorf("requires 'template' for 'fromQuery' and 'target.key'")
			}
			syncValue, err := getTemplatedValue(req.Template, templateData)
			if err != nil {
				return nil, err
			}

			return map[v1alpha1.SecretRef][]byte{
				syncKey: syncValue,
			}, nil
		}

		// Handle FromQuery => KeyPrefix
		if req.Target.KeyPrefix != nil {
			syncMap := make(map[v1alpha1.SecretRef][]byte)
			for key, value := range fetchedSecrets {
				syncKey := v1alpha1.SecretRef{
					Key:     *req.Target.KeyPrefix + "/" + key.GetProperty(),
					Version: key.Version,
				}

				syncValue := value
				if req.Template != nil {
					syncValue, err = getTemplatedValue(req.Template, value)
					if err != nil {
						return nil, err
					}
				}

				syncMap[syncKey] = syncValue
			}
			return syncMap, nil
		}
		return nil, fmt.Errorf("no sources specified")

	// FromSources can only sync a single secret
	case len(req.FromSources) > 0:
		fetchedSecrets, err := p.FetchFromSources(ctx, req.FromSources)
		if err != nil {
			return nil, err
		}

		if req.Target.Key == nil {
			return nil, fmt.Errorf("requires 'target.key' for 'fromSources'")
		}
		syncKey := v1alpha1.SecretRef{
			Key:     *req.Target.Key,
			Version: nil,
		}

		// TODO: Fix template data accessors
		templateData := make(map[string]string)
		for key, value := range fetchedSecrets {
			templateData[key.GetProperty()] = string(value)
		}
		if req.Template == nil {
			return nil, fmt.Errorf("requires 'template' for 'fromSources'")
		}
		syncValue, err := getTemplatedValue(req.Template, templateData)
		if err != nil {
			return nil, err
		}

		return map[v1alpha1.SecretRef][]byte{
			syncKey: syncValue,
		}, nil
	}

	return nil, fmt.Errorf("no sources specified")
}

// FetchFromRef fetches v1alpha1.SecretRef data from reference or from internal fetch store.
func (p *processor) FetchFromRef(ctx context.Context, fromRef v1alpha1.SecretRef) ([]byte, error) {
	// Get from fetch store
	data, exists := p.getFetchedKey(fromRef)

	// Fetch and save if not found
	if !exists {
		var err error
		data, err = p.source.GetSecret(ctx, fromRef)
		if err != nil {
			return nil, err
		}
		p.addFetchedKey(fromRef, data)
	}

	// Return
	return data, nil
}

// FetchFromQuery fetches v1alpha1.SecretRef data from query or from internal fetch store.
func (p *processor) FetchFromQuery(ctx context.Context, fromQuery v1alpha1.SecretQuery) (map[v1alpha1.SecretRef][]byte, error) {
	// List secrets from source
	listKeys, err := p.source.ListSecretKeys(ctx, fromQuery)
	if err != nil {
		return nil, fmt.Errorf("failed while doing query %v: %w", fromQuery, err)
	}

	// Fetch queried keys in parallel
	fetchMu := sync.Mutex{}
	fetched := make(map[v1alpha1.SecretRef][]byte)
	fetchGroup, fetchCtx := errgroup.WithContext(ctx)
	for _, key := range listKeys {
		func(key v1alpha1.SecretRef) {
			fetchGroup.Go(func() error {
				// Fetch
				value, err := p.FetchFromRef(fetchCtx, key)
				if err != nil {
					return err
				}

				// Update
				fetchMu.Lock()
				fetched[key] = value
				fetchMu.Unlock()
				return nil
			})
		}(key)
	}

	// Return
	if err = fetchGroup.Wait(); err != nil {
		return nil, err
	}
	return fetched, nil
}

// FetchFromSources fetches v1alpha1.SecretRef data from selectors or from internal fetch store..
func (p *processor) FetchFromSources(ctx context.Context, fromSources []v1alpha1.SecretSource) (map[v1alpha1.SecretRef][]byte, error) {
	// Fetch source keys from source or fetch store in parallel
	fetchMu := sync.Mutex{}
	fetched := make(map[v1alpha1.SecretRef][]byte)
	fetchGroup, fetchCtx := errgroup.WithContext(ctx)
	for _, src := range fromSources {
		func(src v1alpha1.SecretSource) {
			fetchGroup.Go(func() error {
				// Fetch
				var err error
				result := make(map[v1alpha1.SecretRef][]byte)
				if src.FromRef != nil {
					result[*src.FromRef], err = p.FetchFromRef(fetchCtx, *src.FromRef)
				} else if src.FromQuery != nil {
					result, err = p.FetchFromQuery(fetchCtx, *src.FromQuery)
				} else {
					return fmt.Errorf("both ref and query are empty")
				}
				// Check error
				if err != nil {
					return err
				}

				// Update
				fetchMu.Lock()
				for key, value := range result {
					fetched[key] = value
				}
				fetchMu.Unlock()
				return nil
			})
		}(src)
	}

	// Return
	if err := fetchGroup.Wait(); err != nil {
		return nil, err
	}
	return fetched, nil
}

// getFetchedKey returns a key value from local fetched source.
func (p *processor) getFetchedKey(key v1alpha1.SecretRef) ([]byte, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	res, ok := p.fetched[key]
	return res, ok
}

// addFetchedKey adds a key value to local fetched store.
func (p *processor) addFetchedKey(key v1alpha1.SecretRef, value []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fetched[key] = value
}

func getTemplatedValue(syncTemplate *v1alpha1.SyncTemplate, templateData interface{}) ([]byte, error) {
	// Handle Template.RawData
	if syncTemplate.RawData != nil {
		tpl, err := template.New("template").Parse(*syncTemplate.RawData)
		if err != nil {
			return nil, err
		}
		output := new(bytes.Buffer)
		if err = tpl.Execute(output, struct{ Data interface{} }{Data: templateData}); err != nil {
			return nil, err
		}
		return output.Bytes(), nil
	}

	// Handle Template.Data
	if len(syncTemplate.Data) > 0 {
		outputMap := make(map[string]string)
		for key, keyTpl := range syncTemplate.Data {
			tpl, err := template.New("template").Parse(keyTpl)
			if err != nil {
				return nil, err
			}
			output := new(bytes.Buffer)
			if err = tpl.Execute(output, struct{ Data interface{} }{Data: templateData}); err != nil {
				return nil, err
			}
			outputMap[key] = output.String()
		}

		return json.Marshal(outputMap)
	}

	return nil, fmt.Errorf("cannot apply empty template")
}
