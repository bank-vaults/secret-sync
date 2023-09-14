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
	requests []v1alpha1.SyncRequest,
) (*Status, error) {
	// Validate
	if source == nil {
		return nil, fmt.Errorf("source is nil")
	}
	if dest == nil {
		return nil, fmt.Errorf("dest is nil")
	}
	if len(requests) == 0 {
		return nil, fmt.Errorf("nothing to sync")
	}

	// Get sync data for each sync request and .
	// Do each fetch in a separate goroutine (there could be API requests).
	syncMu := sync.Mutex{}
	syncPlan := make(map[v1alpha1.SecretRef]SyncPlan)
	processor := newProcessor(source)
	{
		extractWg := sync.WaitGroup{}
		for id, req := range requests {
			extractWg.Add(1)
			go func(id int, req v1alpha1.SyncRequest) {
				defer extractWg.Done()

				// Fetch keys to store
				plans, err := processor.GetSyncPlan(ctx, id, req)
				if err != nil {
					logrus.WithField("z-req", req).Warnf("Failed to fetch reqID = %d sync plan, reason: %v", id, err)
					return
				}

				// Add to sync data
				syncMu.Lock()
				for ref, plan := range plans {
					syncPlan[ref] = plan
				}
				syncMu.Unlock()
			}(id, req)
		}
		extractWg.Wait()
	}

	// Sync keys between source and dest read from sync queue.
	// Do sync for each key in a separate goroutine (there will be API requests).
	var successCounter atomic.Uint32
	{
		syncWg := sync.WaitGroup{}
		for ref, plan := range syncPlan {
			syncWg.Add(1)
			go func(ref v1alpha1.SecretRef, plan SyncPlan) {
				defer syncWg.Done()

				err := dest.SetSecret(ctx, ref, plan.Data)
				if err != nil {
					if err == v1alpha1.ErrKeyNotFound { // not found, soft warn
						logrus.WithField("z-req", plan.Request).
							Warnf("Skipped syncing reqID = %d for key %s, reason: %v", plan.RequestID, ref.Key, err)
					} else { // otherwise, log error
						logrus.WithField("z-req", plan.Request).
							Errorf("Failed to sync reqID = %d for key %s, reason: %v", plan.RequestID, ref.Key, err)
					}
					return
				}

				logrus.WithField("z-req", plan.Request).
					Infof("Successfully synced reqID = %d for key %s", plan.RequestID, ref.Key /* , string(plan.Data) */)
				successCounter.Add(1)
			}(ref, plan)
		}
		syncWg.Wait()
	}

	// Return response
	totalCount := uint32(len(syncPlan))
	successCount := successCounter.Load()
	return &Status{
		Total:    totalCount,
		Synced:   successCount,
		Success:  totalCount == successCount,
		Status:   fmt.Sprintf("Synced %d out of total %d keys", successCount, totalCount),
		SyncedAt: time.Now(),
	}, nil
}
