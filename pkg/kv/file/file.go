package file

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/kv"
	"os"
	"path/filepath"
	"strings"
)

type file struct {
	path string
}

// New creates a new kv.Service backed by files, without any encryption
func New(path string) (kv.Store, error) {
	return &file{path: path}, nil
}

func (f *file) Type() string {
	return fmt.Sprintf("file(%s)", f.path)
}

func (f *file) Get(ctx context.Context, key string) (interface{}, error) {
	// Load
	data, err := os.ReadFile(filepath.Join(f.path, key))
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("key '%s' is not present in file", key)
	}

	// Convert
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (f *file) List(ctx context.Context, path string) ([]string, error) {
	// List
	entries, err := os.ReadDir(filepath.Join(f.path, path))
	if err != nil {
		return nil, err
	}

	// Convert
	result := make([]string, len(entries))
	for i, entry := range entries {
		result[i] = strings.ReplaceAll(entry.Name(), filepath.Join(f.path, "/"), "")
	}
	return result, nil
}

func (f *file) Set(ctx context.Context, key string, value interface{}) error {
	path := filepath.Join(f.path, key)

	// Create path
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return err
	}

	// Convert
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Write
	return os.WriteFile(path, data, 0600)
}
