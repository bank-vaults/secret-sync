package sync_test

import (
	"context"
	"github.com/bank-vaults/secret-sync/pkg/kv"
	"github.com/bank-vaults/secret-sync/pkg/kv/file"
	"github.com/bank-vaults/secret-sync/pkg/kv/vault"
	"github.com/bank-vaults/secret-sync/pkg/sync"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

var testCtx = context.Background()

func TestSync(t *testing.T) {
	// Create KV stores
	source := createFileStore(t, "from-dir")
	dest := createFileStore(t, "to-dir")
	//source := createVaultStore(t, "http://0.0.0.0:8200", "root")
	//dest := createVaultStore(t, "http://0.0.0.0:8201", "root")

	// Init source store
	expected := map[string]interface{}{
		"a": "value-a",
		"b": "value-b",
		"c": "value-c",
	}
	initStore(t, source, expected)

	// Empty dest store
	for key := range expected {
		err := dest.Set(testCtx, key, nil)
		assert.Nil(t, err)
	}

	// Sync dest using both Keys and Paths
	manager, err := sync.Start(source, dest,
		sync.WithKeys("a"),
		sync.WithPaths("/"),
		sync.WithPeriod(10*time.Millisecond),
	)
	assert.Nil(t, err)

	// Wait and stop
	time.Sleep(15 * time.Millisecond)
	manager.Stop()
	time.Sleep(1 * time.Second)

	// Validate that dest is synced
	for key, expectedVal := range expected {
		gotVal, err := dest.Get(testCtx, key)
		assert.Nil(t, err)
		assert.Equal(t, expectedVal, gotVal)
	}
}

func createFileStore(t *testing.T, name string) kv.Store {
	path, err := os.MkdirTemp("", name)
	assert.Nil(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(path) })
	store, err := file.New(path)
	assert.Nil(t, err)
	return store
}

func createVaultStore(t *testing.T, addr, token string) kv.Store {
	store, err := vault.New(addr, "secret", "", "userpass", "", token)
	assert.Nil(t, err)
	return store
}

func initStore(t *testing.T, store kv.Store, kv map[string]interface{}) {
	for key, value := range kv {
		assert.Nil(t, store.Set(testCtx, key, value))
	}
}
