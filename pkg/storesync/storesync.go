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

// TODO: Expose a way to handle key collisions (for both fetch and sync)

package storesync

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

// Status defines response data returned by Sync.
type Status struct {
	Total    uint32    //  total number of keys marked for sync
	Synced   uint32    //  number of successful syncs
	Success  bool      //  if Sync was successful
	Status   string    //  an arbitrary status message
	SyncedAt time.Time //  completion timestamp
}

// Sync will synchronize keys from source to dest based on provided specs.
func Sync(ctx context.Context,
	source v1alpha1.StoreReader,
	dest v1alpha1.StoreWriter,
	items []v1alpha1.SyncItem,
) (*Status, error) {
	// Validate
	if source == nil {
		return nil, fmt.Errorf("source is nil")
	}
	if dest == nil {
		return nil, fmt.Errorf("dest is nil")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("nothing to sync")
	}

	// Define intermediate store
	syncData := make(map[v1alpha1.SecretRef]kvData)
	syncMu := sync.Mutex{}
	keyStore := newKvStore()

	// Fetch keys from source and add them to internal store.
	// Do each fetch in a separate goroutine (there could be API requests).
	{
		extractWg := sync.WaitGroup{}
		for i := range items {
			extractWg.Add(1)
			go func(item v1alpha1.SyncItem) {
				defer extractWg.Done()

				// Fetch keys to store
				plan, err := keyStore.GetSyncData(ctx, source, item)
				if err != nil {
					logrus.WithField("from", item).Warnf("Failed to fetch, reason: %v", err)
					return
				}

				// Add to sync data
				syncMu.Lock()
				for key, keyData := range plan {
					syncData[key] = keyData
				}
				syncMu.Unlock()
			}(items[i])
		}
		extractWg.Wait()
	}

	// Sync keys between source and dest read from sync queue.
	// Do sync for each key in a separate goroutine (there will be API requests).
	var successCounter atomic.Uint32
	{
		syncWg := sync.WaitGroup{}
		for key, keyData := range syncData {
			syncWg.Add(1)
			go func(key v1alpha1.SecretRef, keyData kvData) {
				defer syncWg.Done()

				err := dest.SetSecret(ctx, key, keyData.value)
				if err != nil {
					if err == v1alpha1.ErrKeyNotFound { // not found, soft warn
						logrus.WithField("key", key).Warnf("Skipped syncing, reason: %v", err)
					} else { // otherwise, log error
						logrus.WithField("key", key).Errorf("Failed to sync, reason: %v", err)
					}
					return
				}

				logrus.WithField("key", key).Infof("Successfully synced key '%s'", key.Key)
				successCounter.Add(1)
			}(key, keyData)
		}
		syncWg.Wait()
	}

	// Return response
	totalCount := uint32(len(syncData))
	successCount := successCounter.Load()
	return &Status{
		Total:    totalCount,
		Synced:   successCount,
		Success:  totalCount == successCount,
		Status:   fmt.Sprintf("Synced %d out of total %d keys", successCount, totalCount),
		SyncedAt: time.Now(),
	}, nil
}
