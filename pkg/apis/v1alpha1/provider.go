package v1alpha1

import (
	"context"
	"errors"
)

var ErrKeyNotFound = errors.New("secret key not found")

// Provider defines methods to manage store clients.
type Provider interface {
	// NewClient creates a new secret StoreClient for provided backend.
	NewClient(ctx context.Context, backend SecretStoreProvider) (StoreClient, error)

	// Validate checks if the provided backend is valid.
	Validate(backend SecretStoreProvider) error
}

// StoreReader implements read ops for a secret backend. Must support concurrent calls.
type StoreReader interface {
	// GetSecret returns a single secret fetched from secret store.
	GetSecret(ctx context.Context, key SecretKey) ([]byte, error)

	// ListSecretKeys lists all keys matching the query from secret store.
	ListSecretKeys(ctx context.Context, query SecretKeyQuery) ([]SecretKey, error)
}

// StoreWriter implements write ops for a secret backend. Must support concurrent calls.
type StoreWriter interface {
	// SetSecret writes data to a key in a secret store.
	SetSecret(ctx context.Context, key SecretKey, value []byte) error
}

// StoreClient unifies read and write ops for a specific secret backend.
type StoreClient interface {
	StoreReader
	StoreWriter
}
