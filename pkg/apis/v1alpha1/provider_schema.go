// Copyright © 2023 Cisco
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
	"fmt"
	"reflect"
	"sync"
)

var providers = map[string]Provider{}
var providerMu = sync.RWMutex{}

// Register a secret store backend type. Panics if a backend with for the same
// store is already registered.
func Register(provider Provider, backend *SecretStoreProvider) {
	providerName, err := getProviderName(backend)
	if err != nil {
		panic(fmt.Errorf("error registering secret backend: %w", err))
	}

	providerMu.Lock()
	defer providerMu.Unlock()
	if _, exists := providers[providerName]; exists {
		panic(fmt.Errorf("store backend %s already registered", providerName))
	}

	providers[providerName] = provider
}

// GetProvider returns the provider for SecretStoreSpec.
func GetProvider(backend *SecretStoreProvider) (Provider, error) {
	providerName, err := getProviderName(backend)
	if err != nil {
		return nil, fmt.Errorf("failed to find store backend: %w", err)
	}

	providerMu.RLock()
	provider, ok := providers[providerName]
	providerMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("failed to find registered store backend for %s", providerName)
	}

	return provider, nil
}

// getProviderName returns the name of the configured provider or an error if the
// provider is invalid/not configured.
func getProviderName(backend *SecretStoreProvider) (string, error) {
	if backend == nil {
		return "", fmt.Errorf("no StoreConfig provided")
	}
	nilKey, nilCount := "", 0
	v := reflect.ValueOf(*backend)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsNil() {
			nilKey = v.Type().Field(i).Name
			nilCount++
		}
	}
	if nilCount != 1 {
		return "", fmt.Errorf("only one store backend required for StoreConfig, found %d", nilCount)
	}
	return nilKey, nil
}
