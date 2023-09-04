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

package vault

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
	"github.com/bank-vaults/vault-sdk/vault"
)

type Provider struct{}

func (p *Provider) NewClient(_ context.Context, backend v1alpha1.SecretStoreProvider) (v1alpha1.StoreClient, error) {
	providerCfg := backend.Vault
	apiClient, err := vault.NewClientWithOptions(
		vault.ClientURL(providerCfg.Address),
		vault.ClientRole(providerCfg.Role),
		vault.ClientAuthPath(providerCfg.AuthPath),
		vault.ClientTokenPath(providerCfg.TokenPath),
		vault.ClientToken(providerCfg.Token))
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	return &client{
		apiClient:  apiClient,
		apiKeyPath: providerCfg.UnsealKeysPath,
	}, nil
}

func (p *Provider) Validate(backend v1alpha1.SecretStoreProvider) error {
	providerCfg := backend.Vault
	if providerCfg == nil {
		return fmt.Errorf("empty Vault config")
	}
	if providerCfg.Address == "" {
		return fmt.Errorf("empty .Vault.Address")
	}
	if providerCfg.UnsealKeysPath == "" {
		return fmt.Errorf("empty .Vault.UnsealKeysPath")
	}
	if providerCfg.AuthPath == "" {
		return fmt.Errorf("empty .Vault.AuthPath")
	}
	if providerCfg.Token == "" {
		return fmt.Errorf("empty .Vault.Token")
	}
	return nil
}

func init() {
	v1alpha1.Register(&Provider{}, &v1alpha1.SecretStoreProvider{
		Vault: &v1alpha1.SecretStoreProviderVault{},
	})
}
