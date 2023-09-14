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

package storesync_test

//func BenchmarkSync(b *testing.B) {
//	b.ReportAllocs()
//
//	// Prepare
//	source := &fakeClient{}
//	dest := &fakeClient{}
//	requests := refKeys("a", "b/b", "c/c/c")
//	logrus.SetOutput(io.Discard)
//
//	// Run
//	for i := 0; i < b.N; i++ {
//		_, _ = storesync.Sync(context.Background(), source, dest, requests)
//	}
//}
//
//func TestSync(t *testing.T) {
//	testCtx := context.Background()
//
//	// Define sync data
//	source := createFileStore(t, "from-dir")
//	dest := createFileStore(t, "to-dir")
//	_ = createVaultStore(t, "http://0.0.0.0:8200", "root")
//	_ = createVaultStore(t, "http://0.0.0.0:8201", "root")
//
//	expected := fromKeys("a", "b/b", "c/c/c", "d/d/d/0", "d/d/d/1", "d/d/d/2", "d/d/d/d")
//	requests := append(
//		refKeys("a", "b/b", "c/c/c"),
//		refFilter("d/d/d", ".*"),
//	)
//
//	// Init source store
//	initStore(t, source, expected)
//
//	// Sync
//	resp, err := storesync.Sync(testCtx, source, dest, requests)
//	assert.Nil(t, err)
//
//	// Validate that dest is synced
//	assert.Equal(t, true, resp.Success)
//	assert.Equal(t, true, resp.Synced > 0)
//	for _, key := range expected {
//		gotVal, err := dest.GetSecret(testCtx, key)
//		assert.Nil(t, err, key)
//		assert.Equal(t, []byte(key.Key), gotVal, key)
//	}
//}
//
//func initStore(t *testing.T, store v1alpha1.StoreClient, keys []v1alpha1.SecretRef) {
//	for _, key := range keys {
//		assert.Nil(t, store.SetSecret(context.Background(), key, []byte(key.Key)))
//	}
//}
//
//func fromKeys(keys ...string) []v1alpha1.SecretRef {
//	result := make([]v1alpha1.SecretRef, 0)
//	for _, key := range keys {
//		result = append(result, v1alpha1.SecretRef{
//			Key: key,
//		})
//	}
//	return result
//}
//
//func refFrom(keys ...string) []v1alpha1.StrategyDataFrom {
//	result := make([]v1alpha1.StrategyDataFrom, 0)
//	for _, key := range keys {
//		result = append(result, v1alpha1.StrategyDataFrom{
//			Name:      "",
//			SecretRef: nil,
//		})
//	}
//	return result
//}
//
//func createFileStore(t *testing.T, name string) v1alpha1.StoreClient {
//	path, err := os.MkdirTemp("", name)
//	assert.Nil(t, err)
//	t.Cleanup(func() { _ = os.RemoveAll(path) })
//
//	client, err := provider.NewClient(context.Background(), &v1alpha1.ProviderBackend{
//		File: &v1alpha1.FileProvider{
//			DirPath: path,
//		},
//	})
//	assert.Nil(t, err)
//	return client
//}
//
//func createVaultStore(t *testing.T, addr, token string) v1alpha1.StoreClient {
//	client, err := provider.NewClient(context.Background(), &v1alpha1.ProviderBackend{
//		Vault: &v1alpha1.VaultProvider{
//			Address:        addr,
//			storePath: "secret",
//			AuthPath:       "userpass",
//			Token:          token,
//		},
//	})
//	assert.Nil(t, err)
//	return client
//}
//
//type fakeClient struct{}
//
//func (c *fakeClient) GetSecret(_ context.Context, key v1alpha1.SecretRef) ([]byte, error) {
//	return []byte(""), nil
//}
//
//func (c *fakeClient) ListSecretKeys(_ context.Context, _ v1alpha1.SecretKeyQuery) ([]v1alpha1.SecretRef, error) {
//	return []v1alpha1.SecretRef{{}, {}}, nil
//}
//
//func (c *fakeClient) SetSecret(_ context.Context, key v1alpha1.SecretRef, value []byte) error {
//	return nil
//}
