package syncjob

import (
	"context"
	"fmt"
	"github.com/bank-vaults/secret-sync/pkg/apis"
	"github.com/bank-vaults/secret-sync/pkg/provider"
	"github.com/bank-vaults/secret-sync/pkg/sync"
	"github.com/krayzpipes/cronticker/cronticker"
	"github.com/sirupsen/logrus"
	"sync/atomic"
)

// Handler handles a single apis.SyncJobSpec sync request. Safe for concurrent usage.
type Handler interface {
	// LastStatus returns info about latest refresh state.
	LastStatus() *apis.SyncJobRefreshStatus
	// Stop will stop synchronization.
	Stop()
	// Wait will block until sync is completed/stopped.
	Wait()
}

type handler struct {
	stopCh  chan struct{}
	doneCh  chan struct{}
	stopped atomic.Bool
	status  atomic.Pointer[apis.SyncJobRefreshStatus]
}

// Handle will start synchronization from source to dest based on provided params.
// Returns Handler which can be used to manage synchronization state.
func Handle(params apis.SyncJobSpec) (Handler, error) {
	// Validate
	if len(params.KeyFilters) == 0 && len(params.Keys) == 0 {
		return nil, fmt.Errorf("both .KeyFilters and .Keys empty, at least one required")
	}
	sourcePerms := params.SourceStore.GetPermissions()
	if !sourcePerms.CanPerform(apis.SecretStoreRead) {
		return nil, fmt.Errorf("source requires Read permissions, got %s", sourcePerms)
	}
	destPerms := params.DestStore.GetPermissions()
	if !destPerms.CanPerform(apis.SecretStoreWrite) {
		return nil, fmt.Errorf("dest requires Write permissions, got %s", destPerms)
	}

	// Get stores
	sourceClient, err := provider.CreateClient(context.Background(), params.SourceStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create source store client: %w", err)
	}
	destClient, err := provider.CreateClient(context.Background(), params.DestStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create dest store client: %w", err)
	}

	// Create and run handler
	handler := &handler{
		stopCh:  make(chan struct{}, 1),
		doneCh:  make(chan struct{}, 1),
		stopped: atomic.Bool{},
		status:  atomic.Pointer[apis.SyncJobRefreshStatus]{},
	}
	go handler.handle(params, sourceClient, destClient)

	return handler, nil
}

func (h *handler) Wait() {
	<-h.doneCh
}

func (h *handler) Stop() {
	if h.stopped.CompareAndSwap(false, true) {
		close(h.stopCh)
	}
}

func (h *handler) LastStatus() *apis.SyncJobRefreshStatus {
	return h.status.Load()
}

// handle runs processing loop for provided params. This should only be called once.
func (h *handler) handle(params apis.SyncJobSpec, source apis.StoreReader, dest apis.StoreWriter) {
	defer close(h.doneCh)

	// Notify
	logrus.WithField("params", params).Infof("Handling sync")

	// Define request
	req := sync.RefreshRequest{
		Source:     source,
		Dest:       dest,
		Keys:       params.Keys,
		KeyFilters: params.KeyFilters,
		Template:   params.GetTemplate(),
	}

	// Handle once
	if params.RunOnce {
		status := sync.Refresh(context.Background(), req)
		h.status.Store(&status)
		return
	}

	// Handle sync
	syncPeriod, _ := cronticker.NewTicker(params.GetSchedule())
	defer syncPeriod.Stop()
	for {
		// Handle triggers
		select {
		case <-syncPeriod.C:
			status := sync.Refresh(context.Background(), req)
			h.status.Store(&status)

		case <-h.stopCh:
			return
		}
	}
}
