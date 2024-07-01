// Copyright © 2023 Bank-Vaults Maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

var (
	stores  = map[string]SecretStore{}
	storeMu sync.RWMutex
)

// Register a SecretStore for a given backend. Panics if a given backend is already registered.
func Register(store SecretStore, backend *SecretStoreSpec) {
	storeName, err := getSecretStoreName(backend)
	if err != nil {
		panic(fmt.Errorf("error registering secret backend: %w", err))
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	if _, exists := stores[storeName]; exists {
		panic(fmt.Errorf("store backend %s already registered", storeName))
	}

	stores[storeName] = store
}

// GetSecretStore returns the SecretStore for given SecretStoreSpec.
func GetSecretStore(backend *SecretStoreSpec) (SecretStore, error) {
	storeName, err := getSecretStoreName(backend)
	if err != nil {
		return nil, fmt.Errorf("failed to find store backend: %w", err)
	}

	storeMu.RLock()
	defer storeMu.RUnlock()

	store, ok := stores[storeName]
	if !ok {
		return nil, fmt.Errorf("failed to find registered store backend for %s", storeName)
	}

	return store, nil
}

// getSecretStoreName returns the name of the configured SecretStoreSpec or an error if the
// SecretStore is invalid/not configured.
func getSecretStoreName(backend *SecretStoreSpec) (string, error) {
	if backend == nil {
		return "", errors.New("no StoreConfig provided")
	}

	nonNilKey, nonNilCount := "", 0
	v := reflect.ValueOf(*backend)
	for num := range v.NumField() {
		if !v.Field(num).IsNil() {
			nonNilKey = v.Type().Field(num).Name
			nonNilCount++
		}

		if nonNilCount > 1 {
			break
		}
	}

	if nonNilCount != 1 {
		return "", fmt.Errorf("only one store backend required for StoreConfig, found %d", nonNilCount)
	}

	return nonNilKey, nil
}
