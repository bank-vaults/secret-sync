package vault

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/kv"
	"github.com/bank-vaults/vault-sdk/vault"
	"github.com/pkg/errors"
)

type kvVault struct {
	client *vault.Client
	path   string
}

func New(addr, unsealKeysPath, role, authPath, tokenPath, token string) (kv.Store, error) {
	client, err := vault.NewClientWithOptions(
		vault.ClientURL(addr),
		vault.ClientRole(role),
		vault.ClientAuthPath(authPath),
		vault.ClientTokenPath(tokenPath),
		vault.ClientToken(token))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create vault client")
	}

	return &kvVault{
		client: client,
		path:   unsealKeysPath,
	}, nil
}

func (k *kvVault) Type() string {
	return fmt.Sprintf("vault(%s)", k.client.RawClient().Address())
}

func (k *kvVault) Get(ctx context.Context, key string) (interface{}, error) {
	// Get from API
	response, err := k.client.RawClient().Logical().Read(fmt.Sprintf("%s/data/%s", k.path, key))
	if err != nil {
		return nil, fmt.Errorf("get failed for key '%s': %w", key, err)
	}
	if response == nil || response.Data == nil {
		return nil, fmt.Errorf("api returned empty response for key '%s'", key)
	}

	// Read from secret
	secretData, ok := response.Data["data"]
	if !ok || secretData == nil {
		return nil, fmt.Errorf("empty response data for key '%s'", key)
	}
	secretMap, ok := secretData.(map[string]interface{})
	if !ok || secretMap == nil {
		return nil, fmt.Errorf("invalid response data type for key '%s'", key)
	}

	// Return key value data
	keyValue, ok := secretMap[key]
	if !ok || keyValue == nil {
		return nil, fmt.Errorf("key '%s' not found", key)
	}
	return keyValue, nil
}

func (k *kvVault) List(ctx context.Context, path string) ([]string, error) {
	// Fetch
	response, err := k.client.RawClient().Logical().List(fmt.Sprintf("%s/%s/metadata", k.path, path))
	if err != nil {
		return nil, fmt.Errorf("list failed for path '%s': %w", path, err)
	}
	if response == nil || response.Data == nil {
		return nil, fmt.Errorf("api returned empty response for path '%s'", path)
	}

	// Read from response
	listData, ok := response.Data["keys"]
	if !ok || listData == nil {
		return nil, fmt.Errorf("empty response data for path '%s'", path)
	}
	listSlice, ok := listData.([]interface{})
	if !ok || listSlice == nil {
		return nil, fmt.Errorf("invalid response data type for path '%s'", path)
	}

	// Extract
	keys := make([]string, len(listSlice))
	for i, key := range listSlice {
		keys[i] = fmt.Sprintf("%v", key)
	}
	return keys, nil
}

func (k *kvVault) Set(ctx context.Context, key string, value interface{}) error {
	path := fmt.Sprintf("%s/data/%s", k.path, key)
	_, err := k.client.RawClient().Logical().Write(
		path,
		map[string]interface{}{
			"data": map[string]interface{}{
				key: value,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error writing key '%s' to vault addr '%s' and path '%s': %w",
			key, k.client.RawClient().Address(), path, err)
	}
	return nil
}
