package vault

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis"
	"github.com/bank-vaults/vault-sdk/vault"
	"github.com/spf13/cast"
	"strings"
)

type client struct {
	apiClient  *vault.Client
	apiKeyPath string
}

func (c *client) GetSecret(_ context.Context, key apis.StoreKey) ([]byte, error) {
	// Get secret from API
	keyPath := pathForKey(key)
	response, err := c.apiClient.RawClient().Logical().Read(fmt.Sprintf("%s/data/%s", c.apiKeyPath, keyPath))
	if err != nil {
		return nil, fmt.Errorf("api get request failed for key '%s': %w", keyPath, err)
	}
	if response == nil || response.Data == nil {
		return nil, fmt.Errorf("api get request returned empty response for key '%s'", keyPath)
	}

	// Extract key value data
	secretData, ok := response.Data["data"]
	if !ok || secretData == nil {
		return nil, fmt.Errorf("api get returned empty data for key '%s'", key)
	}
	data, err := cast.ToStringMapE(secretData)
	if err != nil {
		return nil, fmt.Errorf("api get request findind data for key '%s': %w", keyPath, err)
	}

	// Get property
	property := key.GetProperty()
	propertyData, ok := data[property]
	if !ok {
		return nil, fmt.Errorf("could not find property '%s' for in get response for '%s'", property, keyPath)
	}
	return base64.StdEncoding.DecodeString(propertyData.(string))
}

func (c *client) ListSecretKeys(ctx context.Context) ([]apis.StoreKey, error) {
	return c.recursiveList(ctx, "")
}

func (c *client) SetSecret(_ context.Context, key apis.StoreKey, value []byte) error {
	// Write secret to API
	keyPath := pathForKey(key)
	_, err := c.apiClient.RawClient().Logical().Write(
		fmt.Sprintf("%s/data/%s", c.apiKeyPath, keyPath),
		map[string]interface{}{
			"data": map[string]interface{}{
				key.GetProperty(): value,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("api set request failed for key '%s': %w", keyPath, err)
	}

	return nil
}

func (c *client) recursiveList(ctx context.Context, path string) ([]apis.StoreKey, error) {
	// List API request
	response, err := c.apiClient.RawClient().Logical().List(fmt.Sprintf("%s/metadata/%s", c.apiKeyPath, path))
	if err != nil {
		return nil, fmt.Errorf("api list request failed: %w", err)
	}
	if response == nil || response.Data == nil {
		return nil, fmt.Errorf("api list request returned empty response")
	}

	// Read from response
	listData, ok := response.Data["keys"]
	if !ok || listData == nil {
		return nil, fmt.Errorf("api list returned empty data for key '%s'", path)
	}
	listSlice, ok := listData.([]interface{})
	if !ok || listSlice == nil {
		return nil, fmt.Errorf("api list returned invalid data for key '%s'", path)
	}

	// Extract keys from response.
	// A key in a KV store can be either a secret or a dir (marked by a suffix '/').
	// For dirs, keep recursively listing them and adding their result results.
	// TODO: Track changes to Vault API https://github.com/hashicorp/vault/issues/5275.
	var result []apis.StoreKey
	for _, listKey := range listSlice {
		subKey := fmt.Sprintf("%s%v", path, listKey)
		if !strings.HasSuffix(subKey, "/") { // key
			result = append(result, apis.StoreKey{
				Key: subKey,
			})
		} else { // dir
			// Recursive list
			subKeys, err := c.recursiveList(ctx, subKey)
			if err != nil {
				return nil, err
			}

			// Add to resulting keys
			for _, subKey := range subKeys {
				result = append(result, subKey)
			}
		}
	}

	return result, nil
}

func pathForKey(key apis.StoreKey) string {
	return strings.Join(append(key.GetPath(), key.GetProperty()), "/")
}
