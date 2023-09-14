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
	"golang.org/x/sync/errgroup"

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

// Sync will synchronize keys from source to target based on provided specs.
func Sync(ctx context.Context,
	source v1alpha1.StoreReader,
	target v1alpha1.StoreWriter,
	requests []v1alpha1.SyncRequest,
) (*Status, error) {
	// Validate
	if source == nil {
		return nil, fmt.Errorf("source is nil")
	}
	if target == nil {
		return nil, fmt.Errorf("target is nil")
	}
	if len(requests) == 0 {
		return nil, fmt.Errorf("nothing to sync")
	}

	// Define data stores
	syncMu := sync.Mutex{}
	syncPlan := make(map[v1alpha1.SecretRef]SyncPlan)
	processor := newProcessor(source)

	// Get sync plan for each request in a separate goroutine.
	// If the same secret needs to be synced more than once, abort sync.
	fetchGroup, fetchCtx := errgroup.WithContext(ctx)
	for id, req := range requests {
		func(id int, req v1alpha1.SyncRequest) {
			fetchGroup.Go(func() error {
				// Fetch keys to store
				plans, err := processor.GetSyncPlan(fetchCtx, id, req)
				if err != nil {
					logrus.WithField("z-req", req).Warnf("Failed to fetch reqID = %d sync plan, reason: %v", id, err)
					return nil
				}

				// Add to sync data
				syncMu.Lock()
				defer syncMu.Unlock()
				for ref, plan := range plans {
					if _, exists := syncPlan[ref]; exists {
						// This is a critical error; stop everything
						return fmt.Errorf("key %v was schedule for sync more than once", ref)
					}
					syncPlan[ref] = plan
				}
				return nil
			})
		}(id, req)
	}

	// Wait fetch
	if err := fetchGroup.Wait(); err != nil {
		return nil, fmt.Errorf("aborted syncing, reason: %w", err)
	}

	// Sync requests from source to target store.
	// Do sync for each plan item in a separate goroutine.
	var syncWg sync.WaitGroup
	var syncCounter atomic.Uint32
	for ref, plan := range syncPlan {
		syncWg.Add(1)
		go func(ref v1alpha1.SecretRef, plan SyncPlan) {
			defer syncWg.Done()

			// Sync
			var err error
			if len(plan.Data) == 0 {
				err = fmt.Errorf("empty value")
			} else {
				err = target.SetSecret(ctx, ref, plan.Data)
			}

			// Handle response
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
			syncCounter.Add(1)
		}(ref, plan)
	}
	syncWg.Wait()

	// Return response
	syncCount := syncCounter.Load()
	totalCount := uint32(len(syncPlan))
	return &Status{
		Total:    totalCount,
		Synced:   syncCount,
		Success:  totalCount == syncCount,
		Status:   fmt.Sprintf("Synced %d out of total %d keys", syncCount, totalCount),
		SyncedAt: time.Now(),
	}, nil
}
