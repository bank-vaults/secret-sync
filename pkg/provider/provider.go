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

package provider

import (
	"context"
	"fmt"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
	// Register providers
	_ "github.com/bank-vaults/secret-sync/pkg/provider/file"
	_ "github.com/bank-vaults/secret-sync/pkg/provider/vault"
)

// NewClient creates a store client for provided store backend config.
func NewClient(ctx context.Context, backend *v1alpha1.ProviderBackend) (v1alpha1.StoreClient, error) {
	// Get provider
	provider, err := v1alpha1.GetProvider(backend)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Validate
	if err = provider.Validate(*backend); err != nil {
		return nil, fmt.Errorf("failed to validate store backend: %w", err)
	}

	// Create
	client, err := provider.NewClient(ctx, *backend)
	if err != nil {
		return nil, fmt.Errorf("failed to create store backend client: %w", err)
	}

	return client, nil
}
