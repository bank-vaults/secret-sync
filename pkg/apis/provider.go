package apis

import (
	"context"
)

// StoreReader implements read ops for a secret backend.
type StoreReader interface {
	// GetSecret returns a single secret fetched from secret store.
	GetSecret(ctx context.Context, key StoreKey) ([]byte, error)

	// ListSecretKeys lists all keys for the current secret store.
	ListSecretKeys(ctx context.Context) ([]StoreKey, error)
}

// StoreWriter implements write ops for a secret backend.
type StoreWriter interface {
	// SetSecret writes data to a key in a secret store.
	SetSecret(ctx context.Context, key StoreKey, value []byte) error
}

// StoreClient unifies read and write ops for a specific secret backend.
type StoreClient interface {
	StoreReader
	StoreWriter
}

// Provider defines methods to interact with secret backends.
type Provider interface {
	// NewClient creates a new secret StoreClient for provided store config.
	// TODO: This should accept SecretStore CR.
	NewClient(ctx context.Context, store SecretStoreSpec) (StoreClient, error)

	// Validate checks if the provided store config is valid.
	// TODO: This should accept SecretStore CR.
	Validate(store SecretStoreSpec) error
}
