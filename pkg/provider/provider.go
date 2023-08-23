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
func NewClient(ctx context.Context, backend *v1alpha1.SecretStoreProvider) (v1alpha1.StoreClient, error) {
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
