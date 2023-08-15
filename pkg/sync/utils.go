package sync

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis"
	_ "github.com/bank-vaults/secret-sync/pkg/provider" // register providers
)

func CreateClientForStore(store apis.SecretStoreSpec) (apis.StoreClient, error) {
	provider, err := apis.GetProvider(&store)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Validate
	if err := provider.Validate(store); err != nil {
		return nil, fmt.Errorf("failed to validate secret store: %w", err)
	}

	// Create
	client, err := provider.NewClient(context.Background(), store)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret store client: %w", err)
	}

	return client, nil
}
