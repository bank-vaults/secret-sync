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

type keyData struct {
	value []byte
}

type kvStore struct {
	mu   sync.RWMutex
	data map[v1alpha1.SecretRef]keyData
}

func newKvStore() *kvStore {
	return &kvStore{
		mu:   sync.RWMutex{},
		data: map[v1alpha1.SecretRef]keyData{},
	}
}

// FetchFromRef fetches v1alpha1.SecretRef data from reference or internal store.
func (s *kvStore) FetchFromRef(ctx context.Context, source v1alpha1.StoreReader, ref v1alpha1.SecretRef) (map[v1alpha1.SecretRef]keyData, error) {
	// Check if exists
	if data, exists := s.GetKey(ref); exists {
		return map[v1alpha1.SecretRef]keyData{ref: data}, nil
	}

	// Get secret
	value, err := source.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Update internal store
	s.SetKey(ref, value)
	data, _ := s.GetKey(ref)
	return map[v1alpha1.SecretRef]keyData{ref: data}, nil
}

// FetchFromQuery fetches v1alpha1.SecretRef data from query or internal store.
func (s *kvStore) FetchFromQuery(ctx context.Context, source v1alpha1.StoreReader, query v1alpha1.SecretQuery) (map[v1alpha1.SecretRef]keyData, error) {
	// List secrets
	listKeys, err := source.ListSecretKeys(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed while doing query %v: %w", query, err)
	}

	// Fetch key values from source in parallel
	data := make(map[v1alpha1.SecretRef]keyData)
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
func (s *kvStore) FetchFromSelectors(ctx context.Context, source v1alpha1.StoreReader, selectors []v1alpha1.SecretsSelector) (map[v1alpha1.SecretRef]keyData, error) {
	// Fetch key values from source in parallel
	data := make(map[v1alpha1.SecretRef]keyData)
	dataMu := sync.RWMutex{}
	procGroup, procCtx := errgroup.WithContext(ctx)
	for _, selector := range selectors {
		func(selector v1alpha1.SecretsSelector) {
			procGroup.Go(func() error {
				// Fetch
				var err error
				var result map[v1alpha1.SecretRef]keyData
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
func (s *kvStore) Fetch(ctx context.Context, source v1alpha1.StoreReader, item v1alpha1.SyncItem) (map[v1alpha1.SecretRef]keyData, error) {
	if item.FromRef != nil {
		return s.FetchFromRef(ctx, source, *item.FromRef)
	} else if item.FromQuery != nil {
		return s.FetchFromQuery(ctx, source, *item.FromQuery)
	} else if len(item.FromSources) > 0 {
		return s.FetchFromSelectors(ctx, source, item.FromSources)
	}
	return nil, fmt.Errorf("no sources specified")
}

func (s *kvStore) GetKey(key v1alpha1.SecretRef) (keyData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if res, ok := s.data[key]; ok {
		return res, true
	}
	return keyData{}, false
}

func (s *kvStore) SetKey(key v1alpha1.SecretRef, value []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = keyData{
		value: value,
	}
}

func (s *kvStore) Sync(ctx context.Context, source v1alpha1.StoreReader, dest v1alpha1.StoreWriter, item v1alpha1.SyncItem) error {
	// Fetch data
	data, err := s.Fetch(ctx, source, item)
	if err != nil {
		return err
	}

	// Get template
	var syncValue []byte
	if item.Template != nil {

	}

	// Sync to a single key
    keysToSync := make(map[v1alpha1.SecretRef][]byte)
	if item.Target.Key != nil {
		data, err :=
	} else if item.Target.KeyPrefix != nil {

	}

	return nil
}

func dataToTemplateData(data map[v1alpha1.SecretRef]keyData) struct{ Data interface{} } {
	extracted := make(map[string][]byte)
	var lastValue []byte
	for key, value := range data {
		lastValue = value.value
		extracted[key.GetProperty()] = value.value
	}
	if len(data) == 1 {
		return struct{ Data interface{} }{Data: lastValue}
	}
	return struct{ Data interface{} }{Data: extracted}
}

func getTemplatedValue(item v1alpha1.SyncItem, data interface{}) ([]byte, error) {
	if item.Template == nil {
		return nil, nil
	}

	if item.Template.RawData != nil {
		return applyTemplate(*item.Template.RawData, data)
	}

	if len(item.Template.Data) > 0 {
		var templateMap map[string][]byte
		for key, valueTemplate := range item.Template.Data {
			result, err := applyTemplate(valueTemplate, data)
			if err != nil {
				return nil, err
			}
			templateMap[key] = result
		}
		return json.Marshal(templateMap)
	}

	return nil, fmt.Errorf("empty template")
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
