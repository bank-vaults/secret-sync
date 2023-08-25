package storesync

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
	"github.com/sirupsen/logrus"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
)

// Request defines request data to use when performing Sync to synchronize data
// from Source to Dest.
type Request struct {
	// Source defines the store from which the keys will be fetched. All keys
	// specified by Keys and keys that match ListFilters listed from Source will be
	// fetched via Source.GetSecret.
	// Required
	Source v1alpha1.StoreReader

	// Dest defines destination of fetched secrets. If Converter is present, then
	// keys will be converted before performing Dest.SetSecret.
	// Required
	Dest v1alpha1.StoreWriter

	// Keys defines which keys to sync will be added to sync queue.
	// Optional
	Keys []v1alpha1.StoreKey

	// ListFilters defines a list of regex filters that will be applied on results of
	// Source ListSecretKeys method. A key from the response will be added to sync
	// queue if it matches at least one filter.
	// Optional
	ListFilters []*regexp.Regexp

	// Converter defines the function that will be called on every key before
	// being sent to Dest. It is used to dynamically change request key data for dest.
	// For example, when needed to add suffix to key.
	// Optional
	Converter func(v1alpha1.StoreKey) (*v1alpha1.StoreKey, error)
}

// Validate validates a Request.
func (req *Request) Validate() error {
	if req.Source == nil {
		return fmt.Errorf("source is nil")
	}
	if req.Dest == nil {
		return fmt.Errorf("dest is nil")
	}
	if len(req.Keys) == 0 && len(req.ListFilters) == 0 {
		return fmt.Errorf("both Keys and ListFilters are empty, at least one required")
	}

	return nil
}

// Response defines response data returned by Sync.
type Response struct {
	Total    int32     //  total number of keys marked for sync
	Synced   int32     //  number of successful syncs
	Success  bool      //  if Sync was successful
	Status   string    //  an arbitrary status message
	SyncedAt time.Time //  completion timestamp
}

// Sync will synchronize data between source and dest based on provided
// request params.
//
// Overview:
//   - Mark all Request.Keys for sync
//   - If Request.ListFilters supplied:
//     -- List all keys on Request.Source
//     -- Filter list by checking for keys that satisfy at least one regex filter
//     -- Mark all list keys that match filters to sync
//   - Sync each key in a separate goroutine
//     -- If Request.Converter is supplied, apply on key before API Set
//   - Return Response with aggregated sync details
func Sync(ctx context.Context, req Request) (*Response, error) {
	// Validate
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("sync request validation failed: %w", err)
	}

	// Fetch filtered list keys from source
	filteredList, err := getFilteredListKeys(ctx, req.Source, req.ListFilters)
	if err != nil {
		logrus.Errorf("Failed to list keys, reason: %v", err)
	} else if len(filteredList) > 0 {
		logrus.Infof("Found %d keys that match list filters", len(filteredList))
	}

	// Sync requested and filtered keys between source and dest.
	// Do sync for each key in a separate goroutine.
	wg := sync.WaitGroup{}
	syncCount := atomic.Uint32{}
	syncKeys := removeDuplicates(append(req.Keys, filteredList...))
	logrus.Infof("Synchronizing %d keys...", len(syncKeys))
	for _, key := range syncKeys {
		wg.Add(1)
		go func(key v1alpha1.StoreKey) {
			defer wg.Done()

			destKey, err := syncKey(ctx, key, req.Source, req.Dest, req.Converter)
			if err != nil {
				if err == v1alpha1.ErrStoreKeyNotFound { // not found, soft warn
					logrus.Warnf("Skipped syncing key '%s', reason: %v", key.Key, err)
				} else { // otherwise, log error
					logrus.Errorf("Failed to sync key '%s', reason: %v", key.Key, err)
				}
				return
			}

			logrus.Infof("Successfully synced key '%s' to '%s'", key.Key, destKey.Key)
			syncCount.Add(1)
		}(key)
	}
	wg.Wait()

	// Return response
	synced, total := int32(syncCount.Load()), int32(len(syncKeys))
	return &Response{
		Total:    total,
		Synced:   synced,
		Success:  total == synced,
		Status:   fmt.Sprintf("Synced %d out of total %d keys", synced, total),
		SyncedAt: time.Now(),
	}, nil
}

// syncKey synchronizes a specific key from source to dest. Returns key synced to dest.
func syncKey(
	ctx context.Context,
	key v1alpha1.StoreKey,
	source v1alpha1.StoreReader,
	dest v1alpha1.StoreWriter,
	converter func(v1alpha1.StoreKey) (*v1alpha1.StoreKey, error),
) (v1alpha1.StoreKey, error) {
	// Get
	value, err := source.GetSecret(ctx, key)
	if err != nil {
		return key, err
	}

	// TODO: Consider adding a check to see if the secret needs to be updated.
	// TODO: This adds additional option to Sync CRD => skip API set if get didn't change since last time

	// Convert key before writing to dest
	if converter != nil {
		newKey, err := converter(key)
		if err != nil {
			return key, err
		}
		key = *newKey
	}

	// Set
	err = dest.SetSecret(ctx, key, value)
	if err != nil {
		return key, err
	}

	return key, err
}

// getFilteredListKeys returns keys listed from v1alpha1.StoreReader that match
// any of the provided regex filters. Returns nil slice for empty keyFilters.
func getFilteredListKeys(ctx context.Context, source v1alpha1.StoreReader, filters []*regexp.Regexp) ([]v1alpha1.StoreKey, error) {
	// Skip on no filters
	if len(filters) == 0 {
		return nil, nil
	}

	// List
	listKeys, err := source.ListSecretKeys(ctx)
	if err != nil {
		return nil, err
	}

	// Filter
	var filteredKeys []v1alpha1.StoreKey
	for _, listKey := range listKeys {
		for _, filter := range filters {
			if filter.MatchString(listKey.Key) {
				filteredKeys = append(filteredKeys, listKey)
				break
			}
		}
	}

	return filteredKeys, nil
}

// removeDuplicates returns v1alpha1.StoreKey slice without duplicates.
func removeDuplicates(keys []v1alpha1.StoreKey) []v1alpha1.StoreKey {
	uniqueKeys := make(map[string]bool)
	results := make([]v1alpha1.StoreKey, 0, len(keys))
	for _, key := range keys {
		if _, exists := uniqueKeys[key.Key]; !exists {
			uniqueKeys[key.Key] = true
			results = append(results, key)
		}
	}
	return results
}
