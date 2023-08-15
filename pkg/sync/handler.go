package sync

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis"
	"github.com/sirupsen/logrus"
	"regexp"
	"sync"
	"time"
)

type Handler interface {
	Stop()
	Wait()
}

type handler struct {
	mu      sync.Mutex
	stopped bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// HandleSync will start synchronization from source to dest based on provided options.
// Returns Manager which can be used to manage synchronization or an error.
func HandleSync(req apis.SyncSecretStoreSpec) (Handler, error) {
	// Validate request
	if req.SourceStore == nil {
		return nil, fmt.Errorf("cannot sync for nil store source")
	}
	if req.DestStore == nil {
		return nil, fmt.Errorf("cannot sync for nil store destination")
	}
	if len(req.KeyFilters) == 0 && len(req.Keys) == 0 {
		return nil, fmt.Errorf("both keys and list key filters are empty, cannot sync")
	}

	// Validate permissions
	if !req.SourceStore.GetPermissions().CanPerform(apis.SecretStoreRead) {
		return nil, fmt.Errorf("source store does not have read permissions")
	}
	if !req.DestStore.GetPermissions().CanPerform(apis.SecretStoreWrite) {
		return nil, fmt.Errorf("destination store does not have write permissions")
	}

	// Get stores
	sourceClient, err := CreateClientForStore(*req.SourceStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create source store: %w", err)
	}
	destClient, err := CreateClientForStore(*req.DestStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create dest store: %w", err)
	}

	// Notify
	logrus.WithField("keys", req.Keys).WithField("filters", req.KeyFilters).Infof("Handling sync")

	// Spawn sync orchestrator which handles sync requests
	stopCh := make(chan struct{}, 1)
	doneCh := make(chan struct{}, 1)
	go func() {
		defer close(doneCh)

		// Handle sync
		syncTicker := time.NewTicker(req.GetSyncPeriod())
		defer syncTicker.Stop()
		for syncID := 1; ; syncID++ {
			// Synchronize
			doSync(syncID, sourceClient, destClient, req.Keys, req.KeyFilters)

			// Handle once
			if req.SyncOnce {
				return
			}

			// Handle triggers
			select {
			case <-syncTicker.C:
				continue

			case <-stopCh:
				return
			}
		}
	}()

	// Return manager
	return &handler{
		stopCh: stopCh,
		doneCh: doneCh,
	}, nil
}

// Stop will stop synchronization. Safe for concurrent usage.
func (h *handler) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.stopped {
		h.stopped = true
		close(h.stopCh)
	}
}

// Wait will block until sync is completed. Safe for concurrent usage.
func (h *handler) Wait() {
	<-h.doneCh
}

// doSync executes synchronization logic for provided params.
func doSync(syncID int, source apis.StoreReader, dest apis.StoreWriter, keys []apis.StoreKey, listFilters []string) {
	log := logrus.WithField("id", syncID)

	log.Infof("Sync id=%d triggered, refreshing...", syncID)

	// Fetch filtered list keys from source
	listKeys, err := getFilteredListKeys(log, listFilters, source)
	if err != nil {
		log.WithError(err).Errorf("Failed to list keys from source")
	} else if len(listKeys) > 0 {
		log.Infof("Found %d keys that match list filters from source", len(listKeys))
	}

	// Sync all keys between source and dest
	<-syncKeys(log, append(listKeys, keys...), source, dest)

	log.Infof("Sync id=%d completed", syncID)
}

// syncKeys handles key synchronization from source to dest.
// Each key sync request will be processed in a separate goroutine.
// Returns a channel that can be consumed to wait for processing.
func syncKeys(log *logrus.Entry, keys []apis.StoreKey, source apis.StoreReader, dest apis.StoreWriter) <-chan struct{} {
	// Do sync for each key in a separate goroutine
	wg := sync.WaitGroup{}
	for _, key := range removeDuplicates(keys) {
		wg.Add(1)
		go func(key apis.StoreKey) {
			defer wg.Done()

			// Get
			value, err := source.GetSecret(context.Background(), key)
			if err != nil {
				log.WithError(err).Errorf("Failed to sync key '%s'", key.Key)
				return
			}

			// Set
			err = dest.SetSecret(context.Background(), key, value)
			if err != nil {
				log.WithError(err).Errorf("Failed to sync key '%s'", key.Key)
				return
			}

			log.Infof("Successfully synced key '%s'", key.Key)
		}(key)
	}

	// Close channel to notify all subscribers
	doneCh := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	return doneCh
}

// getFilteredListKeys returns keys fetched from source that match regex any regex filters.
// Returns nil slice when no filters have passed.
func getFilteredListKeys(log *logrus.Entry, filters []string, source apis.StoreReader) ([]apis.StoreKey, error) {
	// Skip on no filters
	if len(filters) == 0 {
		return nil, nil
	}

	// List
	keys, err := source.ListSecretKeys(context.Background())
	if err != nil {
		return nil, err
	}

	// Filter
	var filteredKeys []apis.StoreKey
	for _, key := range keys {
		for _, filter := range filters {
			if matches, err := regexp.MatchString(filter, key.Key); err != nil {
				log.WithError(err).Errorf("Failed while applying filter '%s' to key %#v", filter, key)
			} else if matches {
				filteredKeys = append(filteredKeys, key)
				break
			}
		}
	}

	return filteredKeys, nil
}

// removeDuplicates removes all duplicates from a slice.
func removeDuplicates(slice []apis.StoreKey) []apis.StoreKey {
	allKeys := make(map[string]bool)
	list := make([]apis.StoreKey, 0, len(slice))
	for _, key := range slice {
		if _, value := allKeys[key.Key]; !value {
			allKeys[key.Key] = true
			list = append(list, key)
		}
	}
	return list
}
