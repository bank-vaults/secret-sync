package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis"
	"github.com/sirupsen/logrus"
	"regexp"
	"sync"
	"sync/atomic"
	"text/template"
	"time"
)

type RefreshRequest struct {
	Source     apis.StoreReader
	Dest       apis.StoreWriter
	Keys       []apis.StoreKey
	KeyFilters []string
	Template   *template.Template
}

// Refresh refreshes executes synchronization logic.
func Refresh(ctx context.Context, req RefreshRequest) apis.SyncJobRefreshStatus {
	// Fetch filtered list keys from source
	filteredList, err := getFilteredListKeys(ctx, req.Source, req.KeyFilters)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to list keys from source")
	} else if len(filteredList) > 0 {
		logrus.Infof("Found %d keys on source that match list filters", len(filteredList))
	}

	// Sync requested and filtered keys between source and dest.
	// Do sync for each key in a separate goroutine.
	wg := sync.WaitGroup{}
	syncedKeys := atomic.Uint32{}
	keysToSync := removeDuplicates(append(req.Keys, filteredList...))
	for _, key := range keysToSync {
		wg.Add(1)
		go func(key apis.StoreKey) {
			defer wg.Done()
			if err := syncKey(ctx, key, req.Source, req.Dest, req.Template); err != nil {
				logrus.WithError(err).Errorf("Failed to sync key '%s'", key.Key)
				return
			}
			logrus.Infof("Successfully synced key '%s'", key.Key)
			syncedKeys.Add(1)
		}(key)
	}
	wg.Wait()

	// Return status
	synced, total := syncedKeys.Load(), uint32(len(keysToSync))
	return apis.SyncJobRefreshStatus{
		Success:  synced == total,
		Status:   fmt.Sprintf("Synced %d out of total %d", synced, total),
		SyncedAt: time.Now(),
	}
}

// syncKey synchronizes a specific key from source to dest.
func syncKey(ctx context.Context, key apis.StoreKey, source apis.StoreReader, dest apis.StoreWriter, template *template.Template) error {
	// Get
	value, err := source.GetSecret(ctx, key)
	if err != nil {
		return err
	}

	// Apply templating if requested
	if template != nil {
		// Apply template
		buffer := &bytes.Buffer{}
		if err = template.Execute(buffer, key); err != nil {
			return err
		}

		// Get updated key from template
		var updatedKey apis.StoreKey
		if err = json.Unmarshal(buffer.Bytes(), &updatedKey); err != nil {
			return err
		}
		key = updatedKey
	}

	// Set
	err = dest.SetSecret(ctx, key, value)
	if err != nil {
		return err
	}

	return nil
}

// getFilteredListKeys returns keys listed from apis.StoreReader that match any
// of the provided regex filters. Returns nil slice for empty keyFilters.
func getFilteredListKeys(ctx context.Context, source apis.StoreReader, keyFilters []string) ([]apis.StoreKey, error) {
	// Skip on no filters
	if len(keyFilters) == 0 {
		return nil, nil
	}

	// List
	listKeys, err := source.ListSecretKeys(ctx)
	if err != nil {
		return nil, err
	}

	// Filter
	var filteredKeys []apis.StoreKey
	for _, listKey := range listKeys {
		for _, keyFilter := range keyFilters {
			if matches, err := regexp.MatchString(keyFilter, listKey.Key); err != nil {
				logrus.WithError(err).Errorf("Failed while applying filter '%s' to key %#v", keyFilter, listKey)
			} else if matches {
				filteredKeys = append(filteredKeys, listKey)
				break
			}
		}
	}

	return filteredKeys, nil
}

// removeDuplicates returns apis.StoreKey slice without duplicates.
func removeDuplicates(keys []apis.StoreKey) []apis.StoreKey {
	uniqueKeys := make(map[string]bool)
	results := make([]apis.StoreKey, 0, len(keys))
	for _, key := range keys {
		keyReq := fmt.Sprintf("%s-%s", key.Key, key.Version)
		if _, exists := uniqueKeys[keyReq]; !exists {
			uniqueKeys[keyReq] = true
			results = append(results, key)
		}
	}
	return results
}
