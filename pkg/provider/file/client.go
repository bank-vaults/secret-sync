package file

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type client struct {
	dir string
}

func (c *client) GetSecret(_ context.Context, key v1alpha1.StoreKey) ([]byte, error) {
	// Read file
	fpath := filepath.Join(c.dir, pathForKey(key))
	data, err := os.ReadFile(fpath)
	if err != nil {
		return nil, v1alpha1.ErrStoreKeyNotFound
	}
	return data, nil
}

func (c *client) ListSecretKeys(_ context.Context) ([]v1alpha1.StoreKey, error) {
	// List whole store dir tree recursively
	// Add all file paths stripped from store path (relative paths)
	var result []v1alpha1.StoreKey
	err := filepath.Walk(c.dir, func(path string, info fs.FileInfo, err error) error {
		// Only add files
		if info != nil && info.Mode().IsRegular() {
			relPath := strings.ReplaceAll(path, filepath.Clean(c.dir)+"/", "")
			result = append(result, v1alpha1.StoreKey{
				Key: filepath.ToSlash(relPath),
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list failed to read dir %s: %w", c.dir, err)
	}
	return result, nil
}

func (c *client) SetSecret(_ context.Context, key v1alpha1.StoreKey, value []byte) error {
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

func pathForKey(key v1alpha1.StoreKey) string {
	return filepath.Join(append(key.GetPath(), key.GetProperty())...)
}
