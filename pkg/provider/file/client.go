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

package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

type client struct {
	dir string
}

func (c *client) GetSecret(_ context.Context, key v1alpha1.SecretRef) ([]byte, error) {
	// Read file
	fpath := filepath.Join(c.dir, pathForKey(key))
	data, err := os.ReadFile(fpath)
	if err != nil {
		return nil, v1alpha1.ErrKeyNotFound
	}
	return data, nil
}

func (c *client) ListSecretKeys(_ context.Context, query v1alpha1.SecretQuery) ([]v1alpha1.SecretRef, error) {
	// Get query dir (if empty, use root)
	queryDir := c.dir
	if query.Path != nil {
		queryDir = filepath.Join(c.dir, *query.Path)
	}

	// Add all files that match filter from queried dir
	var result []v1alpha1.SecretRef
	err := filepath.WalkDir(queryDir, func(path string, entry os.DirEntry, err error) error {
		// Only add files
		if entry != nil && entry.Type().IsRegular() {
			// Extract secret key from the relative OS system path
			relativePath := strings.ReplaceAll(path, c.dir+string(os.PathSeparator), "")
			key := strings.ReplaceAll(relativePath, string(os.PathSeparator), "/")

			// Add key if it matches regexp query
			if matches, _ := regexp.MatchString(query.Key.Regexp, key); matches {
				result = append(result, v1alpha1.SecretRef{
					Key: "/" + key,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list failed to query: %w", err)
	}
	return result, nil
}

func (c *client) SetSecret(_ context.Context, key v1alpha1.SecretRef, value []byte) error {
	// Create parent dir for file
	fpath := filepath.Join(c.dir, pathForKey(key))
	parentDir := filepath.Dir(fpath)
	if err := os.MkdirAll(parentDir, os.ModePerm); err != nil {
		return fmt.Errorf("set failed to create dir %s: %w", parentDir, err)
	}

	// Write file
	if err := os.WriteFile(fpath, value, 0600); err != nil {
		return fmt.Errorf("set failed to write file %s: %w", fpath, err)
	}

	return nil
}

func pathForKey(key v1alpha1.SecretRef) string {
	return filepath.Join(append(key.GetPath(), key.GetProperty())...)
}
