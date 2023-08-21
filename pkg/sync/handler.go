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
	"time"
)

type Handler interface {
	Set(config apis.SyncSecretStoreSpec) error
	Validate(config apis.SyncSecretStoreSpec) error
	Stop()
	Wait()
}

type handler struct {
	mu      sync.Mutex
	stopped bool
	stopCh  chan struct{}
	doneCh  chan struct{}

	options apis.SyncSecretStoreSpec
	source  apis.StoreReader
	dest    apis.StoreWriter
}

// HandleSync will start synchronization from source to dest based on provided options.
// Returns Manager which can be used to manage synchronization or an error.
func HandleSync(req apis.SyncSecretStoreSpec) (Handler, error) {
	// Create handler
	h := &handler{
		mu:      sync.Mutex{},
		stopped: false,
		stopCh:  make(chan struct{}, 1),
		doneCh:  make(chan struct{}, 1),
	}
	if err := h.Set(req); err != nil {
		return nil, fmt.Errorf("could not set config: %w", err)
	}

	// Spawn sync orchestrator which handles sync requests
	go h.handle()

	// Return manager
	return h, nil
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

func (h *handler) Set(config apis.SyncSecretStoreSpec) error {
	if err := h.Validate(config); err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}

	// Get stores
	sourceClient, err := CreateClientForStore(*config.SourceStore)
	if err != nil {
		return fmt.Errorf("failed to create source store client: %w", err)
	}
	destClient, err := CreateClientForStore(*config.DestStore)
	if err != nil {
		return fmt.Errorf("failed to create dest store client: %w", err)
	}

	// Update
	h.mu.Lock()
	defer h.mu.Unlock()

	h.source = sourceClient
	h.dest = destClient
	h.options = config

	return nil
}

func (h *handler) Validate(config apis.SyncSecretStoreSpec) error {
	// Validate request
	if config.SourceStore == nil {
		return fmt.Errorf("empty .SourceStore")
	}
	if config.DestStore == nil {
		return fmt.Errorf("empty .DestStore")
	}
	if len(config.KeyFilters) == 0 && len(config.Keys) == 0 {
		return fmt.Errorf("both .KeyFilters and .Keys empty, at least one required")
	}

	// Validate permissions
	sourcePerms := config.SourceStore.GetPermissions()
	if !sourcePerms.CanPerform(apis.SecretStoreRead) {
		return fmt.Errorf("source requires Read permissions, got %s", sourcePerms)
	}
	destPerms := config.DestStore.GetPermissions()
	if !destPerms.CanPerform(apis.SecretStoreWrite) {
		return fmt.Errorf("dest requires Write permissions, got %s", destPerms)
	}

	return nil
}

func (h *handler) handle() {
	defer close(h.doneCh)

	// Notify
	logrus.WithField("keys", h.options.Keys).WithField("filters", h.options.KeyFilters).Infof("Handling sync")

	// Handle sync
	syncTicker := time.NewTicker(h.options.GetSyncPeriod())
	defer syncTicker.Stop()
	for syncID := 1; ; syncID++ {
		// Synchronize
		h.doSync(syncID)

		// Handle once
		if h.options.SyncOnce {
			return
		}

		// Handle triggers
		select {
		case <-syncTicker.C:
			continue

		case <-h.stopCh:
			return
		}
	}
}

// doSync executes synchronization logic for provided params.
func (h *handler) doSync(syncID int) {
	log := logrus.WithField("id", syncID)

	log.Infof("Sync id=%d triggered, refreshing...", syncID)

	// Fetch filtered list keys from source
	listKeys, err := h.getFilteredListKeys(log)
	if err != nil {
		log.WithError(err).Errorf("Failed to list keys from source")
	} else if len(listKeys) > 0 {
		log.Infof("Found %d keys that match list filters from source", len(listKeys))
	}

	// Sync all keys between source and dest
	<-h.syncKeys(log, append(listKeys, h.options.Keys...))

	log.Infof("Sync id=%d completed", syncID)
}

// syncKeys handles key synchronization from source to dest.
// Each key sync request will be processed in a separate goroutine.
// Returns a channel that can be consumed to wait for processing.
func (h *handler) syncKeys(log *logrus.Entry, keys []apis.StoreKey) <-chan struct{} {
	// Do sync for each key in a separate goroutine
	wg := sync.WaitGroup{}
	for _, key := range removeDuplicates(keys) {
		wg.Add(1)
		go func(key apis.StoreKey) {
			defer wg.Done()

			// Get
			value, err := h.source.GetSecret(context.Background(), key)
			if err != nil {
				log.WithError(err).Errorf("Failed to sync key '%s'", key.Key)
				return
			}

			// Execute template
			if tpl := h.options.GetSyncTemplate(); tpl != nil {
				// Apply template
				var data []byte
				writer := bytes.NewBuffer(data)
				err := tpl.Execute(writer, key)
				if err != nil {
					log.WithError(err).Errorf("Failed to apply template on key '%s'", key.Key)
					return
				}

				// Get updated object
				var updatedKey apis.StoreKey
				bytesData := writer.Bytes()
				if err := json.Unmarshal(bytesData, &updatedKey); err != nil {
					log.WithError(err).Errorf("Failed to unmarshal templated output on key '%s'", key.Key)
					return
				}
				key = updatedKey
			}

			// Set
			err = h.dest.SetSecret(context.Background(), key, value)
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
func (h *handler) getFilteredListKeys(log *logrus.Entry) ([]apis.StoreKey, error) {
	// Skip on no filters
	if len(h.options.KeyFilters) == 0 {
		return nil, nil
	}

	// List
	keys, err := h.source.ListSecretKeys(context.Background())
	if err != nil {
		return nil, err
	}

	// Filter
	var filteredKeys []apis.StoreKey
	for _, key := range keys {
		for _, filter := range h.options.KeyFilters {
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
