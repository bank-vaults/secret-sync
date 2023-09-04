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
