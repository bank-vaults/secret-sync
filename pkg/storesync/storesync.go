// Copyright © 2023 Bank-Vaults Maintainers
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
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/bank-vaults/secret-sync/pkg/apis/v1alpha1"
)

var syncMu sync.Mutex

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
	actions []v1alpha1.SyncAction,
) (*Status, error) {
	// Validate
	if source == nil {
		return nil, errors.New("source is nil")
	}

	if target == nil {
		return nil, errors.New("target is nil")
	}

	if len(actions) == 0 {
		return nil, errors.New("no actions provided")
	}

	// Define data stores
	syncRequests := make(map[v1alpha1.SecretRef]syncRequest)
	processor := newProcessor(source)

	// Get sync plan for each request in a separate goroutine.
	// If the same secret needs to be synced more than once, abort sync.
	fetchGroup, fetchCtx := errgroup.WithContext(ctx)

	for id, action := range actions {
		func(id int, action v1alpha1.SyncAction) {
			fetchGroup.Go(func() error {
				// Fetch keys to store
				requests, err := processor.GetSyncRequests(fetchCtx, id, action)
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("Failed to fetch sync action: %v", err), slog.Any("id", id))
					return nil
				}

				// Add to sync data
				syncMu.Lock()
				defer syncMu.Unlock()
				for ref, request := range requests {
					if _, exists := syncRequests[ref]; exists {
						// This is a critical error; stop everything
						return fmt.Errorf("key %v was schedule for sync more than once", ref)
					}

					syncRequests[ref] = request
				}

				return nil
			})
		}(id, action)
	}

	// Wait fetch
	if err := fetchGroup.Wait(); err != nil {
		return nil, fmt.Errorf("aborted syncing, reason: %w", err)
	}

	// Sync requests from source to target store.
	// Do sync for each plan item in a separate goroutine.
	var syncWg sync.WaitGroup
	var syncCounter atomic.Uint32
	for ref, req := range syncRequests {
		syncWg.Add(1)
		go func(ref v1alpha1.SecretRef, req syncRequest) {
			defer syncWg.Done()

			// Sync
			var err error
			if len(req.Data) == 0 {
				err = errors.New("empty value")
			} else {
				err = target.SetSecret(ctx, ref, req.Data)
			}

			// Handle response
			if err != nil {
				if errors.Is(err, v1alpha1.ErrKeyNotFound) { // not found, soft warn
					slog.WarnContext(ctx, fmt.Sprintf("Skipped sync action: %v", err), slog.Any("id", req.RequestID), slog.Any("key", ref.Key))
				} else { // otherwise, log error
					slog.ErrorContext(ctx, fmt.Errorf("failed to sync action: %w", err).Error(), slog.Any("id", req.RequestID), slog.Any("key", ref.Key))
				}

				return
			}
			slog.InfoContext(ctx, "Successfully synced action", slog.Any("id", req.RequestID), slog.Any("key", ref.Key))
			syncCounter.Add(1)
		}(ref, req)
	}
	syncWg.Wait()

	// Return response
	syncCount := syncCounter.Load()
	totalCount := uint32(len(syncRequests))

	return &Status{
		Total:    totalCount,
		Synced:   syncCount,
		Success:  totalCount == syncCount,
		Status:   fmt.Sprintf("Synced %d out of total %d keys", syncCount, totalCount),
		SyncedAt: time.Now(),
	}, nil
}
