package sync

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/kv"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Manager is used to manage synchronization process.
// It should only be created via Start.
type Manager struct {
	mu      sync.Mutex
	stopped bool
	stopCh  chan struct{}
}

// Start will start synchronization from source to dest based on provided options.
// Returns Manager which can be used to manage synchronization or an error.
func Start(source kv.Reader, dest kv.Writer, opts ...Option) (*Manager, error) {
	option := newOptions(opts...)

	// Validate
	if source == nil {
		return nil, fmt.Errorf("cannot sync for nil kv source")
	}
	if dest == nil {
		return nil, fmt.Errorf("cannot sync for nil kv destination")
	}
	if len(option.Paths) == 0 && len(option.Keys) == 0 {
		return nil, fmt.Errorf("both paths and keys empty, nothing to sync")
	}

	// Spawn orchestrator
	stopCh := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(option.Period)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C: // Handles sync
				logrus.Infof("Sync triggered, refreshing...")

				syncPaths(option.Paths, source, dest)
				syncKeys(option.Keys, source, dest)

			case <-stopCh: // Handles closing
				logrus.Infof("Sync terminated, closing...")
				return
			}
		}
	}()

	// Return manager
	return &Manager{
		stopCh: stopCh,
	}, nil
}

// Stop will stop synchronization. Safe for concurrent usage.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stopped {
		m.stopped = true
		close(m.stopCh)
	}
}

// syncPaths syncs keys returned by List from source to dest.
// Each path will be processed in a separate goroutine.
func syncPaths(paths []string, source kv.Reader, dest kv.Writer) {
	for _, path := range paths {
		go func(path string) {
			from := source.Type()
			logrus.Infof("Fetching keys for path '%s' from '%s'", path, from)

			// List
			keys, err := source.List(context.Background(), path)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to fetch keys for path '%s' from '%s'", path, from)
				return
			}

			logrus.Infof("Fetched %d keys for path '%s' from '%s'", len(keys), path, from)

			// Sync fetched keys
			syncKeys(keys, source, dest)
		}(path)
	}
}

// syncKeys handles key synchronization from source to dest.
// Each key will be processed in a separate goroutine.
func syncKeys(keys []string, source kv.Reader, dest kv.Writer) {
	for _, key := range keys {
		go func(key string) {
			from, to := source.Type(), dest.Type()
			logrus.Infof("Syncing key '%s' from '%s' to '%s'", key, from, to)

			// Get
			value, err := source.Get(context.Background(), key)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to sync key '%s' from '%s' to '%s'", key, from, to)
				return
			}

			// Set
			err = dest.Set(context.Background(), key, value)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to sync key '%s' from '%s' to '%s'", key, from, to)
				return
			}

			logrus.Infof("Successfully synced key '%s' from '%s' to '%s'", key, from, to)
		}(key)
	}
}
