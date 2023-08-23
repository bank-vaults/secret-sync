package vault

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
	"github.com/bank-vaults/vault-sdk/vault"
)

type Provider struct{}

func (p *Provider) NewClient(_ context.Context, backend v1alpha1.SecretStoreProvider) (v1alpha1.StoreClient, error) {
	vaultCfg := backend.Vault
	apiClient, err := vault.NewClientWithOptions(
		vault.ClientURL(vaultCfg.Address),
		vault.ClientRole(vaultCfg.Role),
		vault.ClientAuthPath(vaultCfg.AuthPath),
		vault.ClientTokenPath(vaultCfg.TokenPath),
		vault.ClientToken(vaultCfg.Token))
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	return &client{
		apiClient:  apiClient,
		apiKeyPath: vaultCfg.UnsealKeysPath,
	}, nil
}

func (p *Provider) Validate(backend v1alpha1.SecretStoreProvider) error {
	vaultCfg := backend.Vault
	if vaultCfg == nil {
		return fmt.Errorf("empty Vault config")
	}
	if vaultCfg.Address == "" {
		return fmt.Errorf("empty .Vault.Address")
	}
	if vaultCfg.UnsealKeysPath == "" {
		return fmt.Errorf("empty .Vault.UnsealKeysPath")
	}
	if vaultCfg.AuthPath == "" {
		return fmt.Errorf("empty .Vault.AuthPath")
	}
	if vaultCfg.Token == "" {
		return fmt.Errorf("empty .Vault.Token")
	}
	return nil
}

func init() {
	v1alpha1.Register(&Provider{}, &v1alpha1.SecretStoreProvider{
		Vault: &v1alpha1.SecretStoreProviderVault{},
	})
}
