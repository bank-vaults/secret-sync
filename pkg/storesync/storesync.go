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

package storesync

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

// syncRequest defines data required to perform a key sync.
// This is the minimal unit of work for full v1alpha1.SecretRef sync.
type syncRequest struct {
	SecretKey    v1alpha1.SecretRef
	KeyTransform []v1alpha1.SecretKeyTransform
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
func Sync(ctx context.Context, source v1alpha1.StoreReader, dest v1alpha1.StoreWriter, refs []v1alpha1.SecretRemoteRef) (*Status, error) {
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
			go func(ref v1alpha1.SecretRemoteRef) {
				defer extractWg.Done()

				// Fetch keys
				secretKeys, err := getKeys(ctx, source, ref)
				if err != nil {
					logrus.Warnf("Failed to extract keys, reason: %v", err)
				}

				// Submit keys for sync
				for i := range secretKeys {
					syncQueue <- syncRequest{
						SecretKey:    secretKeys[i],    // use fetched key
						KeyTransform: ref.KeyTransform, // use same transform
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

// getKeys fetches (one or multiple) v1alpha1.SecretRef for a single v1alpha1.SecretRemoteRef.
// Performs an API list request on source if ref Query is specified to get multiple v1alpha1.SecretRef.
func getKeys(ctx context.Context, source v1alpha1.StoreReader, ref v1alpha1.SecretRemoteRef) ([]v1alpha1.SecretRef, error) {
	// Validate
	if ref.Secret == nil && ref.Query == nil {
		return nil, fmt.Errorf("both SecretRef and Query are empty, at least one required")
	}

	// Get keys
	var keys []v1alpha1.SecretRef
	if ref.Secret != nil {
		// Add static key
		keys = append(keys, *ref.Secret)
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
func doRequest(ctx context.Context, source v1alpha1.StoreReader, dest v1alpha1.StoreWriter, req syncRequest) (v1alpha1.SecretRef, error) {
	// Get from source
	key := req.SecretKey
	value, err := source.GetSecret(ctx, key)
	if err != nil {
		return key, err
	}

	// TODO: Consider adding a check to see if the secret needs to be updated.
	// TODO: This adds additional option to Sync CRD => skip API set if get didn't change since last time

	// Transform before writing to dest
	updatedKey, err := applyTransform(key, req.KeyTransform)
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

// applyTransform applies transform to v1alpha1.SecretRef and returns updated key or error.
func applyTransform(secretKey v1alpha1.SecretRef, transforms []v1alpha1.SecretKeyTransform) (v1alpha1.SecretRef, error) {
	for _, transform := range transforms {
		// Update Regexp field
		keyRegex := transform.Regexp
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
