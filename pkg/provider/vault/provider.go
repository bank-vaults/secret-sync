package vault

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis"
	"github.com/bank-vaults/vault-sdk/vault"
)

type Provider struct{}

var _ apis.Provider = &Provider{}

func (p *Provider) NewClient(_ context.Context, store apis.SecretStoreSpec) (apis.StoreClient, error) {
	provider := store.Provider.Vault
	apiClient, err := vault.NewClientWithOptions(
		vault.ClientURL(provider.Address),
		vault.ClientRole(provider.Role),
		vault.ClientAuthPath(provider.AuthPath),
		vault.ClientTokenPath(provider.TokenPath),
		vault.ClientToken(provider.Token))
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	return &client{
		apiClient:  apiClient,
		apiKeyPath: provider.UnsealKeysPath,
	}, nil
}

func (p *Provider) Validate(store apis.SecretStoreSpec) error {
	provider := store.Provider.Vault
	if provider == nil {
		return fmt.Errorf("empty .Vault")
	}
	if provider.Address == "" {
		return fmt.Errorf("empty .Vault.Address")
	}
	if provider.UnsealKeysPath == "" {
		return fmt.Errorf("empty .Vault.UnsealKeysPath")
	}
	if provider.AuthPath == "" {
		return fmt.Errorf("empty .Vault.AuthPath")
	}
	if provider.Token == "" {
		return fmt.Errorf("empty .Vault.Token")
	}
	return nil
}

func init() {
	apis.Register(&Provider{}, &apis.SecretStoreProvider{
		Vault: &apis.SecretStoreProviderVault{},
	})
}
