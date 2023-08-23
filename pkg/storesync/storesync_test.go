package storesync_test

import (
	"context"
	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
	"github.com/bank-vaults/secret-sync/pkg/provider"
	"github.com/bank-vaults/secret-sync/pkg/storesync"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"regexp"
	"testing"
)

func BenchmarkSync(b *testing.B) {
	b.ReportAllocs()

	// Prepare
	logrus.SetOutput(io.Discard)
	request := storesync.Request{
		Source: &fakeClient{},
		Dest:   &fakeClient{},
		Keys:   convertKeys("a", "b/b", "c/c/c"),
	}

	// Run
	for i := 0; i < b.N; i++ {
		_, _ = storesync.Sync(context.Background(), request)
	}
}

func TestSync(t *testing.T) {
	testCtx := context.Background()

	// Define sync data
	source := createFileStore(t, "from-dir")
	dest := createFileStore(t, "to-dir")
	destConverter := func(key v1alpha1.StoreKey) (*v1alpha1.StoreKey, error) {
		key.Key = key.Key + "/suffix"
		return &key, nil
	}
	// source := createVaultStore(t, "http://0.0.0.0:8200", "root")
	// dest := createVaultStore(t, "http://0.0.0.0:8201", "root")
	expected := convertKeys("a", "b/b", "c/c/c", "d/d/d/0", "d/d/d/1", "d/d/d/2", "d/d/d/d/d")
	request := storesync.Request{
		Source:       source,
		Dest:         dest,
		Keys:         convertKeys("a", "b/b", "c/c/c"),
		ListFilters:  convertFilters("d/d/d"),
		SetConverter: destConverter,
	}

	// Init source store
	initStore(t, source, expected)

	// Sync
	resp, err := storesync.Sync(testCtx, request)
	assert.Nil(t, err)

	// Validate that dest is synced
	assert.Equal(t, true, resp.Success)
	assert.Equal(t, true, resp.Synced > 0)
	for _, key := range expected {
		newKey, _ := request.SetConverter(key)
		gotVal, err := dest.GetSecret(testCtx, *newKey)
		assert.Nil(t, err)
		assert.Equal(t, []byte(key.Key), gotVal)
	}
}

func initStore(t *testing.T, store v1alpha1.StoreClient, keyData []v1alpha1.StoreKey) {
	for _, keyReq := range keyData {
		assert.Nil(t, store.SetSecret(context.Background(), keyReq, []byte(keyReq.Key)))
	}
}

func convertKeys(keys ...string) []v1alpha1.StoreKey {
	result := make([]v1alpha1.StoreKey, 0)
	for _, key := range keys {
		result = append(result, v1alpha1.StoreKey{
			Key: key,
		})
	}
	return result
}

func convertFilters(filters ...string) []*regexp.Regexp {
	result := make([]*regexp.Regexp, 0)
	for _, filter := range filters {
		result = append(result, regexp.MustCompile(filter))
	}
	return result
}

func createFileStore(t *testing.T, name string) v1alpha1.StoreClient {
	path, err := os.MkdirTemp("", name)
	assert.Nil(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(path) })

	client, err := provider.NewClient(context.Background(), &v1alpha1.SecretStoreProvider{
		File: &v1alpha1.SecretStoreProviderFile{
			DirPath: path,
		},
	})
	assert.Nil(t, err)
	return client
}

func createVaultStore(t *testing.T, addr, token string) v1alpha1.StoreClient {
	client, err := provider.NewClient(context.Background(), &v1alpha1.SecretStoreProvider{
		Vault: &v1alpha1.SecretStoreProviderVault{
			Address:        addr,
			UnsealKeysPath: "secret",
			AuthPath:       "userpass",
			Token:          token,
		},
	})
	assert.Nil(t, err)
	return client
}

type fakeClient struct{}

func (c *fakeClient) GetSecret(_ context.Context, key v1alpha1.StoreKey) ([]byte, error) {
	return []byte(""), nil
}

func (c *fakeClient) ListSecretKeys(_ context.Context) ([]v1alpha1.StoreKey, error) {
	return []v1alpha1.StoreKey{{}, {}}, nil
}

func (c *fakeClient) SetSecret(_ context.Context, key v1alpha1.StoreKey, value []byte) error {
	return nil
}
