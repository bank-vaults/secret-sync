package provider

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis"

	// Register providers
	_ "github.com/bank-vaults/secret-sync/pkg/provider/file"
	_ "github.com/bank-vaults/secret-sync/pkg/provider/vault"
)

// CreateClient creates an apis.StoreClient for provided apis.SecretStoreSpec.
func CreateClient(ctx context.Context, store apis.SecretStoreSpec) (apis.StoreClient, error) {
	provider, err := apis.GetProvider(&store.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Validate
	if err = provider.Validate(store); err != nil {
		return nil, fmt.Errorf("failed to validate secret store: %w", err)
	}

	// Create
	client, err := provider.NewClient(ctx, store)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret store client: %w", err)
	}

	return client, nil
}
