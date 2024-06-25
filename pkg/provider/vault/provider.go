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

package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/bank-vaults/vault-sdk/vault"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

type Provider struct{}

func (p *Provider) NewClient(_ context.Context, backend v1alpha1.SecretStoreSpec) (v1alpha1.StoreClient, error) {
	apiClient, err := vault.NewClientWithOptions(
		vault.ClientURL(backend.Vault.Address),
		vault.ClientRole(backend.Vault.Role),
		vault.ClientAuthPath(backend.Vault.AuthPath),
		vault.ClientTokenPath(backend.Vault.TokenPath),
		vault.ClientToken(backend.Vault.Token))
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	return &client{
		apiClient:  apiClient,
		apiKeyPath: backend.Vault.StorePath,
	}, nil
}

func (p *Provider) Validate(backend v1alpha1.SecretStoreSpec) error {
	if backend.Vault == nil {
		return errors.New("empty Vault config")
	}
	if backend.Vault.Address == "" {
		return errors.New("empty .Vault.Address")
	}
	if backend.Vault.StorePath == "" {
		return errors.New("empty .Vault.StorePath")
	}
	if backend.Vault.AuthPath == "" {
		return errors.New("empty .Vault.AuthPath")
	}
	if backend.Vault.Token == "" {
		return errors.New("empty .Vault.Token")
	}

	return nil
}

func init() {
	v1alpha1.Register(&Provider{}, &v1alpha1.SecretStoreSpec{
		Vault: &v1alpha1.VaultStore{},
	})
}
