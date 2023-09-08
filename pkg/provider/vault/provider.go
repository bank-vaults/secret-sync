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

	"github.com/bank-vaults/vault-sdk/vault"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

type Provider struct{}

func (p *Provider) NewClient(_ context.Context, backend v1alpha1.ProviderBackend) (v1alpha1.StoreClient, error) {
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

func (p *Provider) Validate(backend v1alpha1.ProviderBackend) error {
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
	v1alpha1.Register(&Provider{}, &v1alpha1.ProviderBackend{
		Vault: &v1alpha1.VaultProvider{},
	})
}
