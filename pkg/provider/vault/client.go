// Copyright © 2023 Bank-Vaults Maintainers
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

package vault

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/bank-vaults/vault-sdk/vault"
	"github.com/spf13/cast"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

type client struct {
	apiClient  *vault.Client
	apiKeyPath string
}

func (c *client) GetSecret(ctx context.Context, key v1alpha1.SecretRef) ([]byte, error) {
	// Get secret from API
	response, err := c.apiClient.RawClient().Logical().ReadWithContext(
		ctx,
		fmt.Sprintf("%s/data/%s", c.apiKeyPath, pathForKey(key)),
	)
	if err != nil {
		return nil, fmt.Errorf("api get request failed: %w", err)
	}

	if response == nil || response.Data == nil {
		// TODO: check if this is valid err return
		return nil, v1alpha1.ErrKeyNotFound
	}

	// Extract key value data
	secretData, ok := response.Data["data"]
	if !ok || secretData == nil {
		return nil, errors.New("api get returned empty data")
	}

	data, err := cast.ToStringMapE(secretData)
	if err != nil {
		return nil, fmt.Errorf("api get request findind data: %w", err)
	}

	// Get name
	keyData, ok := data[key.GetName()]
	if !ok {
		return nil, fmt.Errorf("could not find %s for in get response", key.GetName())
	}

	return []byte(keyData.(string)), nil
}

func (c *client) ListSecretKeys(ctx context.Context, query v1alpha1.SecretQuery) ([]v1alpha1.SecretRef, error) {
	// Get relative path to dir
	queryPath := ""
	if query.Path != nil {
		queryPath = *query.Path
	}

	// List API request
	response, err := c.apiClient.RawClient().Logical().ListWithContext(
		ctx,
		fmt.Sprintf("%s/metadata/%s", c.apiKeyPath, queryPath),
	)
	if err != nil {
		return nil, fmt.Errorf("api list request failed: %w", err)
	}

	if response == nil || response.Data == nil {
		// TODO: check if this is valid err return
		return nil, v1alpha1.ErrKeyNotFound
	}

	// Read from response
	listData, ok := response.Data["keys"]
	if !ok || listData == nil {
		return nil, errors.New("api list returned empty data")
	}

	listSlice, ok := listData.([]interface{})
	if !ok || listSlice == nil {
		return nil, errors.New("api list returned invalid data")
	}

	// Extract keys from response
	var result []v1alpha1.SecretRef
	for _, listKey := range listSlice {
		// Skip values in KV store that are not keys (marked by a suffix '/').
		keyName := fmt.Sprintf("%v", listKey)
		if strings.HasSuffix(keyName, "/") {
			continue
		}

		// Add key if it matches regexp query
		if matches, _ := regexp.MatchString(query.Key.Regexp, keyName); matches {
			result = append(result, v1alpha1.SecretRef{
				Key: fmt.Sprintf("%s%s", queryPath, keyName),
			})
		}
	}

	return result, nil
}

func (c *client) SetSecret(ctx context.Context, key v1alpha1.SecretRef, value []byte) error {
	// Write secret to API
	_, err := c.apiClient.RawClient().Logical().WriteWithContext(
		ctx,
		fmt.Sprintf("%s/data/%s", c.apiKeyPath, pathForKey(key)),
		map[string]interface{}{
			"data": map[string]interface{}{
				key.GetName(): string(value),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("api set request failed: %w", err)
	}

	return nil
}

// recursiveList will recursively list all items in a Vault.
// Not used since it has high memory footprint and does not handle search.
// It could (potentially) be useful.
// DEPRECATED
func (c *client) recursiveList(ctx context.Context, path string) ([]v1alpha1.SecretRef, error) { //nolint: unused
	// List API request
	response, err := c.apiClient.RawClient().Logical().ListWithContext(
		ctx,
		fmt.Sprintf("%s/metadata/%s", c.apiKeyPath, path),
	)
	if err != nil {
		return nil, fmt.Errorf("api list request failed: %w", err)
	}
	if response == nil || response.Data == nil {
		return nil, errors.New("api list request returned empty response")
	}

	// Read from response
	listData, ok := response.Data["keys"]
	if !ok || listData == nil {
		return nil, fmt.Errorf("api list returned empty data for key %s", path)
	}

	listSlice, ok := listData.([]interface{})
	if !ok || listSlice == nil {
		return nil, fmt.Errorf("api list returned invalid data for key %s", path)
	}

	// Extract keys from response.
	// A key in a KV store can be either a secret or a dir (marked by a suffix '/').
	// For dirs, keep recursively listing them and adding their result results.
	// TODO: Track changes to Vault API https://github.com/hashicorp/vault/issues/5275.
	var result []v1alpha1.SecretRef
	for _, listKey := range listSlice {
		subKey := fmt.Sprintf("%s%v", path, listKey)
		if !strings.HasSuffix(subKey, "/") { // key
			result = append(result, v1alpha1.SecretRef{
				Key: subKey,
			})
		} else { // dir
			// Recursive list
			subKeys, err := c.recursiveList(ctx, subKey)
			if err != nil {
				return nil, err
			}

			// Add to resulting keys
			result = append(result, subKeys...)
		}
	}

	return result, nil
}

func pathForKey(key v1alpha1.SecretRef) string {
	// If key has no path, return name as path
	if len(key.GetPath()) == 0 {
		return key.GetName()
	}

	return strings.Join(key.GetPath(), "/")
}
