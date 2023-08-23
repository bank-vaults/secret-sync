package v1alpha1

import (
	"context"
	"errors"
)

var ErrStoreKeyNotFound = errors.New("secret key not found")

// Provider defines methods to manage store clients.
type Provider interface {
	// NewClient creates a new secret StoreClient for provided backend.
	NewClient(ctx context.Context, backend SecretStoreProvider) (StoreClient, error)

	// Validate checks if the provided backend is valid.
	Validate(backend SecretStoreProvider) error
}

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
