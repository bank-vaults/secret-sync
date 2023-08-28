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

// syncRequest defines data required to perform a key sync.
// This is the minimal unit of work for full v1alpha1.SecretKey sync.
type syncRequest struct {
	SecretKey v1alpha1.SecretKey
	Rewrite   []v1alpha1.SecretKeyRewrite
}

// Status defines response data returned by Sync.
type Status struct {
	Total    uint32    //  total number of keys marked for sync
	Synced   uint32    //  number of successful syncs
	Success  bool      //  if Sync was successful
	Status   string    //  an arbitrary status message
	SyncedAt time.Time //  completion timestamp
}

// Sync will synchronize keys from source to dest based on provided specs.
func Sync(ctx context.Context, source v1alpha1.StoreReader, dest v1alpha1.StoreWriter, refs []v1alpha1.SecretKeyFromRef) (*Status, error) {
	// Validate
	if source == nil {
		return nil, fmt.Errorf("source is nil")
	}
	if dest == nil {
		return nil, fmt.Errorf("dest is nil")
	}
	if len(refs) == 0 {
		return nil, fmt.Errorf("no sync data")
	}

	// All sync requests will be concurrently sent to this channel
	syncQueue := make(chan syncRequest, 1)

	// Fetch keys based on ref params and add them to sync queue.
	// Do each fetch in a separate goroutine (there could be API requests).
	{
		extractWg := sync.WaitGroup{}
		for i := range refs {
			extractWg.Add(1)
			go func(ref v1alpha1.SecretKeyFromRef) {
				defer extractWg.Done()

				// Fetch keys
				secretKeys, err := getKeys(ctx, source, ref)
				if err != nil {
					logrus.Warnf("Failed to extract keys, reason: %v", err)
				}

				// Submit keys for sync
				for i := range secretKeys {
					syncQueue <- syncRequest{
						SecretKey: secretKeys[i], // use fetched key
						Rewrite:   ref.Rewrite,   // use same rewrite
					}
				}
			}(refs[i])
		}

		// Close sync request channel when everything has been extracted to stop the loop
		go func() {
			extractWg.Wait()
			close(syncQueue)
		}()
	}

	// Sync keys between source and dest read from sync queue.
	// Do sync for each key in a separate goroutine (there will be API requests).
	var totalCount uint32
	var successCounter atomic.Uint32
	{
		syncWg := sync.WaitGroup{}
		syncKeyMap := make(map[string]bool)
		for req := range syncQueue {
			// Check if the key has already been synced
			if _, exists := syncKeyMap[req.SecretKey.Key]; exists {
				continue
			}
			syncKeyMap[req.SecretKey.Key] = true
			totalCount++

			// Sync key in a separate goroutine
			syncWg.Add(1)
			go func(req syncRequest) {
				defer syncWg.Done()

				key := req.SecretKey
				destKey, err := doRequest(ctx, source, dest, req)
				if err != nil {
					if err == v1alpha1.ErrKeyNotFound { // not found, soft warn
						logrus.Warnf("Skipped syncing key '%s', reason: %v", key.Key, err)
					} else { // otherwise, log error
						logrus.Errorf("Failed to sync key '%s', reason: %v", key.Key, err)
					}
					return
				}

				logrus.Infof("Successfully synced key '%s' to '%s'", key.Key, destKey.Key)
				successCounter.Add(1)
			}(req)
		}
		syncWg.Wait()
	}

	// Return response
	successCount := successCounter.Load()
	return &Status{
		Total:    totalCount,
		Synced:   successCount,
		Success:  totalCount == successCount,
		Status:   fmt.Sprintf("Synced %d out of total %d keys", successCount, totalCount),
		SyncedAt: time.Now(),
	}, nil
}

// getKeys fetches (one or multiple) v1alpha1.SecretKey for a single v1alpha1.SecretKeyFromRef.
// Performs an API list request on source if ref Query is specified to get multiple v1alpha1.SecretKey.
func getKeys(ctx context.Context, source v1alpha1.StoreReader, ref v1alpha1.SecretKeyFromRef) ([]v1alpha1.SecretKey, error) {
	// Validate
	if ref.SecretKey == nil && ref.Query == nil {
		return nil, fmt.Errorf("both SecretKey and Query are empty, at least one required")
	}

	// Get keys
	var keys []v1alpha1.SecretKey
	if ref.SecretKey != nil {
		// Add static key
		keys = append(keys, *ref.SecretKey)
	}
	if ref.Query != nil {
		// Get keys from API
		listKeys, err := source.ListSecretKeys(ctx, *ref.Query)
		if err != nil {
			return nil, fmt.Errorf("failed while doing query %v: %w", *ref.Query, err)
		}
		keys = append(listKeys, keys...)
	}

	return keys, nil
}

// doRequest will sync a given syncRequest from source to dest. Returns key that was synced to dest or error.
func doRequest(ctx context.Context, source v1alpha1.StoreReader, dest v1alpha1.StoreWriter, req syncRequest) (v1alpha1.SecretKey, error) {
	// Get from source
	key := req.SecretKey
	value, err := source.GetSecret(ctx, key)
	if err != nil {
		return key, err
	}

	// TODO: Consider adding a check to see if the secret needs to be updated.
	// TODO: This adds additional option to Sync CRD => skip API set if get didn't change since last time

	// Rewrite before writing to dest
	updatedKey, err := applyRewrites(key, req.Rewrite)
	if err != nil {
		return key, err
	}

	// Set to dest
	err = dest.SetSecret(ctx, updatedKey, value)
	if err != nil {
		return updatedKey, err
	}

	return updatedKey, nil
}

// applyRewrites applies rewrites to v1alpha1.SecretKey and returns updated key or error.
func applyRewrites(secretKey v1alpha1.SecretKey, rewrites []v1alpha1.SecretKeyRewrite) (v1alpha1.SecretKey, error) {
	for _, rewrite := range rewrites {
		// Update Regexp field
		keyRegex := rewrite.Regexp
		if keyRegex == nil {
			continue
		}
		keyGroup, err := regexp.Compile(keyRegex.Source)
		if err != nil {
			return secretKey, fmt.Errorf("failed to compile regex %s: %w", keyRegex.Source, err)
		}
		secretKey.Key = keyGroup.ReplaceAllString(secretKey.Key, keyRegex.Target)
	}
	return secretKey, nil
}
