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
	providerVault := store.Provider.Vault
	apiClient, err := vault.NewClientWithOptions(
		vault.ClientURL(providerVault.Address),
		vault.ClientRole(providerVault.Role),
		vault.ClientAuthPath(providerVault.AuthPath),
		vault.ClientTokenPath(providerVault.TokenPath),
		vault.ClientToken(providerVault.Token))
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	return &client{
		apiClient:  apiClient,
		apiKeyPath: providerVault.UnsealKeysPath,
	}, nil
}

func (p *Provider) Validate(store apis.SecretStoreSpec) error {
	providerVault := store.Provider.Vault
	if providerVault == nil {
		return fmt.Errorf("empty .Vault")
	}
	if providerVault.Address == "" {
		return fmt.Errorf("empty .Vault.Address")
	}
	if providerVault.UnsealKeysPath == "" {
		return fmt.Errorf("empty .Vault.UnsealKeysPath")
	}
	if providerVault.AuthPath == "" {
		return fmt.Errorf("empty .Vault.AuthPath")
	}
	if providerVault.Token == "" {
		return fmt.Errorf("empty .Vault.Token")
	}
	return nil
}

func init() {
	apis.Register(&Provider{}, &apis.SecretStoreProvider{
		Vault: &apis.SecretStoreProviderVault{},
	})
}
