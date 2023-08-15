package sync_test

import (
	"context"
	"github.com/bank-vaults/secret-sync/pkg/apis"
	"github.com/bank-vaults/secret-sync/pkg/sync"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var testCtx = context.Background()

func TestSync(t *testing.T) {
	// Create secret stores
	//sourceSpec, sourceClient := createFileStore(t, "from-dir")
	//destSpec, destClient := createFileStore(t, "to-dir")
	sourceSpec, sourceClient := createVaultStore(t, "http://0.0.0.0:8200", "root")
	destSpec, destClient := createVaultStore(t, "http://0.0.0.0:8201", "root")

	// Define store and sync data
	syncKeys := convertKeys("a", "b/b", "c/c/c")
	syncFilters := []string{"d/d/d"}
	expected := convertKeys(
		"a",
		"b/b",
		"c/c/c",
		"d/d/d",
		"d/d/d/0",
		"d/d/d/1",
		"d/d/d/2",
		"d/d/d/d/d",
	)

	// Init source store
	initStore(t, sourceClient, expected)

	// Empty dest store
	for _, key := range expected {
		err := destClient.SetSecret(testCtx, key, nil)
		assert.Nil(t, err)
	}

	// Sync dest using both Keys and KeyFilters
	manager, err := sync.HandleSync(apis.SyncSecretStoreSpec{
		SourceStore: &sourceSpec,
		DestStore:   &destSpec,
		Keys:        syncKeys,
		KeyFilters:  syncFilters,
		SyncOnce:    true,
	})
	assert.Nil(t, err)

	// Wait for first sync
	manager.Wait()

	// Validate that dest is synced
	for _, key := range expected {
		gotVal, err := destClient.GetSecret(testCtx, key)
		assert.Nil(t, err)
		assert.Equal(t, []byte(key.Key), gotVal)
	}
}

func createFileStore(t *testing.T, name string) (apis.SecretStoreSpec, apis.StoreClient) {
	path, err := os.MkdirTemp("", name)
	assert.Nil(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(path) })

	store := apis.SecretStoreSpec{
		Provider: &apis.SecretStoreProvider{
			File: &apis.SecretStoreProviderFile{
				ParentDir: path,
			},
		},
	}
	client, err := sync.CreateClientForStore(store)
	assert.Nil(t, err)
	return store, client
}

func createVaultStore(t *testing.T, addr, token string) (apis.SecretStoreSpec, apis.StoreClient) {
	store := apis.SecretStoreSpec{
		Provider: &apis.SecretStoreProvider{
			Vault: &apis.SecretStoreProviderVault{
				Address:        addr,
				UnsealKeysPath: "secret",
				AuthPath:       "userpass",
				Token:          token,
			},
		},
	}
	client, err := sync.CreateClientForStore(store)
	assert.Nil(t, err)
	return store, client
}

func initStore(t *testing.T, store apis.StoreClient, keyData []apis.StoreKey) {
	for _, keyReq := range keyData {
		assert.Nil(t, store.SetSecret(testCtx, keyReq, []byte(keyReq.Key)))
	}
}

func convertKeys(keys ...string) []apis.StoreKey {
	result := make([]apis.StoreKey, 0)
	for _, key := range keys {
		result = append(result, apis.StoreKey{
			Key: key,
		})
	}
	return result
}
