package apis

import (
	"encoding/json"
	"fmt"
	"sync"
)

var providers = map[string]Provider{}
var providerMu = sync.RWMutex{}

// Register a secret store backend type. Panics if a backend with for the same
// store is already registered.
func Register(provider Provider, providerSpec *SecretStoreProvider) {
	providerName, err := getProviderName(providerSpec)
	if err != nil {
		panic(fmt.Errorf("secretstore error registering provider schema: %w", err))
	}

	providerMu.Lock()
	defer providerMu.Unlock()
	if _, exists := providers[providerName]; exists {
		panic(fmt.Errorf("secretstore provider %q already registered", providerName))
	}

	providers[providerName] = provider
}

// GetProvider returns the provider for SecretStoreSpec.
// TODO: This should accept SecretStore CR.
func GetProvider(spec *SecretStoreSpec) (Provider, error) {
	if spec == nil {
		return nil, fmt.Errorf("no secretstore spec found for %#v", spec)
	}
	providerName, err := getProviderName(spec.Provider)
	if err != nil {
		return nil, fmt.Errorf("secretstore error for %#v: %w", spec, err)
	}

	providerMu.RLock()
	provider, ok := providers[providerName]
	providerMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("failed to find registered store backend %q for %#v", providerName, spec)
	}

	return provider, nil
}

// getProviderName returns the name of the configured provider or an error if the
// provider is not configured.
func getProviderName(providerSpec *SecretStoreProvider) (string, error) {
	providerBytes, err := json.Marshal(providerSpec)
	if err != nil || providerBytes == nil {
		return "", fmt.Errorf("failed to marshal secretstore provider spec: %w", err)
	}

	providerMap := make(map[string]interface{})
	if err = json.Unmarshal(providerBytes, &providerMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal secretstore provider spec: %w", err)
	}
	if len(providerMap) != 1 {
		return "", fmt.Errorf("secretstore can have only one backend specified, found %d", len(providerMap))
	}

	for k := range providerMap {
		return k, nil
	}

	return "", fmt.Errorf("failed to find registered store backend")
}
