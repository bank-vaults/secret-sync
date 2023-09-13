package storesync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"strings"
	"sync"
	"text/template"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

type kvData struct {
	value []byte
}

type kvStore struct {
	mu   sync.RWMutex
	data map[v1alpha1.SecretRef]kvData
}

type templateData struct {
	Data interface{}
}

func newKvStore() *kvStore {
	return &kvStore{
		mu:   sync.RWMutex{},
		data: map[v1alpha1.SecretRef]kvData{},
	}
}

// FetchFromRef fetches v1alpha1.SecretRef data from reference or internal store.
func (s *kvStore) FetchFromRef(ctx context.Context, source v1alpha1.StoreReader, ref v1alpha1.SecretRef) (map[v1alpha1.SecretRef]kvData, error) {
	// Check if exists
	if data, exists := s.GetKey(ref); exists {
		return map[v1alpha1.SecretRef]kvData{ref: data}, nil
	}

	// Get secret
	value, err := source.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Update internal store
	s.SetKey(ref, value)
	data, _ := s.GetKey(ref)
	return map[v1alpha1.SecretRef]kvData{
		ref: data,
	}, nil
}

// FetchFromQuery fetches v1alpha1.SecretRef data from query or internal store.
func (s *kvStore) FetchFromQuery(ctx context.Context, source v1alpha1.StoreReader, query v1alpha1.SecretQuery) (map[v1alpha1.SecretRef]kvData, error) {
	// List secrets
	listKeys, err := source.ListSecretKeys(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed while doing query %v: %w", query, err)
	}

	// Fetch key values from source in parallel
	data := make(map[v1alpha1.SecretRef]kvData)
	dataMu := sync.RWMutex{}
	procGroup, procCtx := errgroup.WithContext(ctx)
	for _, key := range listKeys {
		func(key v1alpha1.SecretRef) {
			procGroup.Go(func() error {
				// Fetch
				result, err := s.FetchFromRef(procCtx, source, key)
				if err != nil {
					return err
				}

				// Update
				dataMu.Lock()
				for key, value := range result {
					data[key] = value
				}
				dataMu.Unlock()
				return nil
			})
		}(key)
	}

	// Return
	if err = procGroup.Wait(); err != nil {
		return nil, err
	}
	return data, nil
}

// FetchFromSelectors fetches v1alpha1.SecretRef data from selectors or internal store.
func (s *kvStore) FetchFromSelectors(ctx context.Context, source v1alpha1.StoreReader, selectors []v1alpha1.SecretsSelector) (map[v1alpha1.SecretRef]kvData, error) {
	// Fetch key values from source in parallel
	data := make(map[v1alpha1.SecretRef]kvData)
	dataMu := sync.RWMutex{}
	procGroup, procCtx := errgroup.WithContext(ctx)
	for _, selector := range selectors {
		func(selector v1alpha1.SecretsSelector) {
			procGroup.Go(func() error {
				// Fetch
				var err error
				var result map[v1alpha1.SecretRef]kvData
				if selector.FromRef != nil {
					result, err = s.FetchFromRef(procCtx, source, *selector.FromRef)
				} else if selector.FromQuery != nil {
					result, err = s.FetchFromQuery(procCtx, source, *selector.FromQuery)
				} else {
					return fmt.Errorf("both ref and query are empty")
				}

				// Check error
				if err != nil {
					return err
				}

				// Update
				dataMu.Lock()
				for key, value := range result {
					data[key] = value
				}
				dataMu.Unlock()
				return nil
			})
		}(selector)
	}

	// Return
	if err := procGroup.Wait(); err != nil {
		return nil, err
	}
	return data, nil
}

// Fetch fetches data from API or local store.
func (s *kvStore) Fetch(ctx context.Context, source v1alpha1.StoreReader, item v1alpha1.SyncItem) (map[v1alpha1.SecretRef]kvData, error) {
	if item.FromRef != nil {
		return s.FetchFromRef(ctx, source, *item.FromRef)
	} else if item.FromQuery != nil {
		return s.FetchFromQuery(ctx, source, *item.FromQuery)
	} else if len(item.FromSources) > 0 {
		return s.FetchFromSelectors(ctx, source, item.FromSources)
	}
	return nil, fmt.Errorf("no sources specified")
}

func (s *kvStore) GetKey(key v1alpha1.SecretRef) (kvData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if res, ok := s.data[key]; ok {
		return res, true
	}
	return kvData{}, false
}

func (s *kvStore) SetKey(key v1alpha1.SecretRef, value []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = kvData{
		value: value,
	}
}

func (s *kvStore) GetSyncData(ctx context.Context, source v1alpha1.StoreReader, item v1alpha1.SyncItem) (map[v1alpha1.SecretRef]kvData, error) {
	// Fetch store data
	sourceData, err := s.Fetch(ctx, source, item)
	if err != nil {
		return nil, err
	}

	// Process keys
	keysToSync := map[v1alpha1.SecretRef]kvData{}
	switch target := item.Target; {
	// Handle KeyPrefix which indicates that multiple keys need to be synced
	case target.KeyPrefix != nil:
		// Supports FromQuery
		if item.FromQuery == nil {
			return nil, fmt.Errorf("requires 'fromQuery' for 'target.keyPrefix'")
		}
		for key, keyData := range sourceData {
			tplValue, err := applyOptionalTemplate(item.Template, keyData, templateData{string(keyData.value)})
			if err != nil {
				return nil, err
			}
			keysToSync[prefixedKey(key, *target.KeyPrefix)] = *tplValue
			fmt.Printf("SYNCING %v => %s\n", prefixedKey(key, *target.KeyPrefix), string(tplValue.value))
		}
		return keysToSync, nil

	// Handle Key which indicates that only one key needs to be synced
	case target.Key != nil:
		// Supports FromRef
		if item.FromRef != nil {
			for key, keyData := range sourceData {
				tplValue, err := applyOptionalTemplate(item.Template, keyData, templateData{string(keyData.value)})
				if err != nil {
					return nil, err
				}
				keysToSync[newKey(*target.Key, key.Version)] = *tplValue
				fmt.Printf("SYNCING %v => %s\n", newKey(*target.Key, key.Version), string(tplValue.value))
			}
			return keysToSync, nil
		}

		// Supports FromQuery
		if item.FromQuery != nil {
			if item.Template == nil {
				return nil, fmt.Errorf("requires 'template' for 'fromQuery' and 'target.key'")
			}
			tplData := make(map[string]string)
			for key, keyData := range sourceData {
				tplData[key.GetProperty()] = string(keyData.value)
			}
			tplValue, err := applyOptionalTemplate(item.Template, kvData{}, templateData{tplData})
			if err != nil {
				return nil, err
			}
			keysToSync[newKey(*target.Key, nil)] = *tplValue
			fmt.Printf("SYNCING %v => %s\n", newKey(*target.Key, nil), string(tplValue.value))

			return keysToSync, nil
		}

		// Supports FromSources
		if len(item.FromSources) > 0 {
			if item.Template != nil {
				return nil, fmt.Errorf("requires 'template' for 'fromSources' and 'target.key'")
			}
			return keysToSync, nil
		}

	// Handle empty
	default:
		// Supports FromRef
		if item.FromRef == nil {
			return nil, fmt.Errorf("requires at least 'fromRef' for empty 'target'")
		}
		for _, keyData := range sourceData {
			tplValue, err := applyOptionalTemplate(item.Template, keyData, templateData{string(keyData.value)})
			if err != nil {
				return nil, err
			}
			keysToSync[*item.FromRef] = *tplValue
			fmt.Printf("SYNCING %v => %s\n", *item.FromRef, string(tplValue.value))
		}
		return keysToSync, nil
	}

	// Synchronize
	return nil, fmt.Errorf("invalid request")
}

func applyOptionalTemplate(tpl *v1alpha1.SyncTemplate, keyData kvData, tplData templateData) (*kvData, error) {
	if tpl == nil {
		return &keyData, nil
	}

	// Handle Template.RawData
	if tpl.RawData != nil {
		// Apply templates
		tmpl, err := template.New("template").Parse(*tpl.RawData)
		if err != nil {
			return nil, err
		}
		buf := new(bytes.Buffer)
		err = tmpl.Execute(buf, tplData)
		if err != nil {
			return nil, err
		}
		return &kvData{
			value: buf.Bytes(),
		}, nil
	}

	// Handle Template.Data
	if len(tpl.Data) == 0 {
		return &keyData, nil
	}

	valueMap := make(map[string]string)
	for key, keyTpl := range tpl.Data {
		tmpl, err := template.New("template").Parse(keyTpl)
		if err != nil {
			return nil, err
		}
		buf := new(bytes.Buffer)
		err = tmpl.Execute(buf, tplData)
		if err != nil {
			return nil, err
		}
		valueMap[key] = buf.String()
	}

	rawValue, err := json.Marshal(valueMap)
	if err != nil {
		return nil, err
	}

	return &kvData{
		value: rawValue,
	}, nil
}

func applyTemplate(tpl string, data interface{}) ([]byte, error) {
	// Apply templates
	tmpl, err := template.New("template").Parse(tpl)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func prefixedKey(key v1alpha1.SecretRef, prefix string) v1alpha1.SecretRef {
	return v1alpha1.SecretRef{
		Key:     strings.Join([]string{prefix, key.GetProperty()}, "/"),
		Version: key.Version,
	}
}

func newKey(key string, version *string) v1alpha1.SecretRef {
	return v1alpha1.SecretRef{
		Key:     key,
		Version: version,
	}
}
