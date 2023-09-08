package storesync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"text/template"

	"golang.org/x/sync/errgroup"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

type kvData struct {
	key   v1alpha1.SecretRef
	value []byte
}

type kvStore struct {
	mu   sync.RWMutex
	data map[string]kvData
}

func newKvStore() *kvStore {
	return &kvStore{
		mu:   sync.RWMutex{},
		data: map[string]kvData{},
	}
}

// Fetch fetches v1alpha1.SecretRef data for a v1alpha1.StrategyDataFrom.
func (s *kvStore) Fetch(ctx context.Context, source v1alpha1.StoreReader, ref v1alpha1.StrategyDataFrom) error {
	// Validate
	if ref.Name == "" {
		return fmt.Errorf("cannot use empty Name")
	}
	if ref.SecretRef == nil && ref.SecretQuery == nil {
		return fmt.Errorf("both SecretRef and SecretQuery are empty, at least one required")
	}
	if ref.SecretRef != nil && ref.SecretQuery != nil {
		return fmt.Errorf("both SecretRef and SecretQuery are provided, only one required")
	}

	// Handle ref key
	if ref.SecretRef != nil {
		value, err := source.GetSecret(ctx, *ref.SecretRef)
		if err != nil {
			return err
		}
		s.setKeyValue(ref.Name, *ref.SecretRef, value, false)
		return nil
	}

	// Handle ref query
	listKeys, err := source.ListSecretKeys(ctx, *ref.SecretQuery)
	if err != nil {
		return fmt.Errorf("failed while doing query %v: %w", *ref.SecretQuery, err)
	}

	// Fetch key values from source parallelly
	procGroup, procCtx := errgroup.WithContext(ctx)
	for _, key := range listKeys {
		func(key v1alpha1.SecretRef) {
			procGroup.Go(func() error {
				value, err := source.GetSecret(procCtx, key)
				if err != nil {
					return err
				}
				s.setKeyValue(ref.Name, key, value, true)
				return nil
			})
		}(key)
	}

	// Wait for error
	return procGroup.Wait()
}

// Sync syncs v1alpha1.SecretRef data for a v1alpha1.StrategyDataTo.
func (s *kvStore) Sync(ctx context.Context, dest v1alpha1.StoreWriter, ref v1alpha1.StrategyDataTo) (*v1alpha1.SecretRef, error) {
	// Validate
	if ref.Key == "" {
		return nil, fmt.Errorf("cannot use empty Key")
	}
	if ref.Value == nil && ref.ValueMap == nil {
		return nil, fmt.Errorf("both Value and ValueMap are empty, at least one required")
	}
	if ref.Value != nil && ref.ValueMap != nil {
		return nil, fmt.Errorf("both Value and ValueMap are provided, only one required")
	}

	// Get key
	keyName, err := s.getTemplatedKey(ref.Key)
	if err != nil {
		return nil, err
	}
	keyToSync := &v1alpha1.SecretRef{
		Key: keyName,
	}

	// Handle ref value
	if ref.Value != nil {
		value, err := s.getTemplatedValue(*ref.Value)
		if err != nil {
			return nil, err
		}
		return keyToSync, dest.SetSecret(ctx, *keyToSync, value)
	}

	// Handle ref value map
	valueMap := make(map[string][]byte)
	for key, value := range ref.ValueMap {
		templatedValue, err := s.getTemplatedValue(value)
		if err != nil {
			return nil, err
		}
		valueMap[key] = templatedValue
	}

	rawValueMap, err := json.Marshal(valueMap)
	if err != nil {
		return nil, err
	}
	return keyToSync, dest.SetSecret(ctx, *keyToSync, rawValueMap)
}

func (s *kvStore) getKeyValue(name string) (kvData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if res, ok := s.data[name]; ok {
		return res, true
	}
	return kvData{}, false
}

func (s *kvStore) setKeyValue(name string, key v1alpha1.SecretRef, value []byte, isQuery bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	strKey := name
	if isQuery {
		strKey = name + "." + key.GetProperty()
	}

	s.data[strKey] = kvData{
		key:   key,
		value: value,
	}
}

func (s *kvStore) getTemplatedKey(key string) (string, error) {
	tmpl, err := template.New("key").Funcs(template.FuncMap{
		"pathTo": func(obj any) (string, error) {
			key, _ := obj.(string)
			value, ok := s.getKeyValue(key)
			if !ok {
				return "", fmt.Errorf("key %v not found", obj)
			}
			return value.key.Key, nil
		},
	}).Parse(key)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *kvStore) getTemplatedValue(key string) ([]byte, error) {
	tmpl, err := template.New("key").Funcs(template.FuncMap{
		"valueOf": func(obj any) ([]byte, error) {
			key, _ := obj.(string)
			value, ok := s.getKeyValue(key)
			if !ok {
				return nil, fmt.Errorf("key %v not found", obj)
			}
			return value.value, nil
		},
	}).Parse(key)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, nil)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
