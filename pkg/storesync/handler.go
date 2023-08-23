package storesync

import (
	"context"
	"github.com/sirupsen/logrus"
	"runtime"
	"sync"
	"sync/atomic"
)

var numParallelWorkers = 2 * runtime.NumCPU()

// Handler handles processing of Request asynchronously. It can also serve as a
// general-purpose async processor.
// TODO: This should not be used, abstracts k8s operator reconcile loop.
type Handler interface {
	// Stop will stop Handler.
	Stop()
	// Wait will block until the Handler stops.
	Wait()
	// LastSync returns info about latest successful sync.
	LastSync() *Response
	// Add adds the Request to processing queue. Request will be performed using
	// given context. Add on stopped Handler does nothing. If Response channel is
	// supplied, sync result will be writen to it.
	Add(ctx context.Context, req Request, respCh chan<- Response)
}

type handler struct {
	stopCh   chan struct{}
	doneCh   chan struct{}
	stopped  atomic.Bool
	status   atomic.Pointer[Response]
	reqQueCh chan queRequest
}

// queRequest defines all the data for queue to process a request
type queRequest struct {
	Context    context.Context // ctx to use for Sync
	Request    Request         // sync request data
	ResponseCh chan<- Response // must be a buffered channel
}

// newHandler creates a synchronization orchestrator that performs syncs on
// demand. Returns Handler which can be used to manage synchronization states or
// submit Request. Returned Handler will process maximum of numParallelWorkers
// requests parallely.
func newHandler() (Handler, error) {
	// Create and run handler
	handler := &handler{
		stopCh:   make(chan struct{}, 1),
		doneCh:   make(chan struct{}, 1),
		stopped:  atomic.Bool{},
		status:   atomic.Pointer[Response]{},
		reqQueCh: make(chan queRequest, 4096), // sufficiently large queue to avoid Add blocks
	}
	go handler.handle()

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

func (h *handler) LastSync() *Response {
	return h.status.Load()
}

func (h *handler) Add(ctx context.Context, req Request, respCh chan<- Response) {
	if h.stopped.Load() {
		return
	}
	h.reqQueCh <- queRequest{
		Context:    ctx,
		Request:    req,
		ResponseCh: respCh,
	}
}

// handle runs processing loop for provided params. This should only be called once.
// Every request from reqQueCh will be processed by one of the workers.
func (h *handler) handle() {
	// Handle sync in parallel
	var reqID atomic.Int32
	var wg sync.WaitGroup
	for workerID := 1; workerID <= numParallelWorkers; workerID++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			logrus.Debugf("Started sync worker id=%d", workerID)

			for {
				select {
				case queReq := <-h.reqQueCh:
					reqID := reqID.Add(1) // increment and fetch
					request := queReq.Request
					logrus.Infof("Sync id=%d accepted by worker %d", reqID, workerID)

					status, err := Sync(queReq.Context, request)
					if err != nil {
						logrus.Errorf("Sync id=%d failed, reason: %v", reqID, err)
					} else {
						logrus.Errorf("Sync id=%d done, synced %d of %d keys", reqID, status.Synced, status.Total)
						h.status.Store(status)
						if queReq.ResponseCh != nil {
							queReq.ResponseCh <- *status
						}
					}

				case <-h.stopCh:
					logrus.Debugf("Stopping sync worker id=%d", workerID)
					return
				}
			}
		}(workerID)
	}

	// Close done channel when all the workers have shut down
	go func() {
		wg.Wait()
		close(h.doneCh)
	}()
}
