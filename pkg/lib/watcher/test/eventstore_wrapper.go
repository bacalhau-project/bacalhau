package test

import (
	"context"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
)

type Interceptor func() error

// EventStoreWrapper provides a way to intercept and modify the behavior of an EventStore
// for testing purposes. It wraps an actual EventStore and allows setting interceptors
// for each method.
type EventStoreWrapper struct {
	actualStore watcher.EventStore
	mu          sync.RWMutex

	storeEventInterceptor      Interceptor
	getEventsInterceptor       Interceptor
	getLatestEventInterceptor  Interceptor
	storeCheckpointInterceptor Interceptor
	getCheckpointInterceptor   Interceptor
	closeInterceptor           Interceptor
}

func NewEventStoreWrapper(actualStore watcher.EventStore) *EventStoreWrapper {
	return &EventStoreWrapper{
		actualStore: actualStore,
	}
}

func (w *EventStoreWrapper) WithStoreEventInterceptor(f Interceptor) *EventStoreWrapper {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.storeEventInterceptor = f
	return w
}

func (w *EventStoreWrapper) WithGetEventsInterceptor(f Interceptor) *EventStoreWrapper {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.getEventsInterceptor = f
	return w
}

func (w *EventStoreWrapper) WithGetLatestEventInterceptor(f Interceptor) *EventStoreWrapper {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.getLatestEventInterceptor = f
	return w
}

func (w *EventStoreWrapper) WithStoreCheckpointInterceptor(f Interceptor) *EventStoreWrapper {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.storeCheckpointInterceptor = f
	return w
}

func (w *EventStoreWrapper) WithGetCheckpointInterceptor(f Interceptor) *EventStoreWrapper {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.getCheckpointInterceptor = f
	return w
}

func (w *EventStoreWrapper) WithCloseInterceptor(f Interceptor) *EventStoreWrapper {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closeInterceptor = f
	return w
}

func (w *EventStoreWrapper) StoreEvent(ctx context.Context, operation watcher.Operation, objectType string, object interface{}) error {
	if err := w.intercept(w.storeEventInterceptor); err != nil {
		return err
	}
	return w.actualStore.StoreEvent(ctx, operation, objectType, object)
}

func (w *EventStoreWrapper) GetEvents(ctx context.Context, params watcher.GetEventsRequest) (*watcher.GetEventsResponse, error) {
	if err := w.intercept(w.getEventsInterceptor); err != nil {
		return nil, err
	}
	return w.actualStore.GetEvents(ctx, params)
}

func (w *EventStoreWrapper) GetLatestEventNum(ctx context.Context) (uint64, error) {
	if err := w.intercept(w.getLatestEventInterceptor); err != nil {
		return 0, err
	}
	return w.actualStore.GetLatestEventNum(ctx)
}

func (w *EventStoreWrapper) StoreCheckpoint(ctx context.Context, watcherID string, eventSeqNum uint64) error {
	if err := w.intercept(w.storeCheckpointInterceptor); err != nil {
		return err
	}
	return w.actualStore.StoreCheckpoint(ctx, watcherID, eventSeqNum)
}

func (w *EventStoreWrapper) GetCheckpoint(ctx context.Context, watcherID string) (uint64, error) {
	if err := w.intercept(w.getCheckpointInterceptor); err != nil {
		return 0, err
	}
	return w.actualStore.GetCheckpoint(ctx, watcherID)
}

func (w *EventStoreWrapper) Close(ctx context.Context) error {
	if err := w.intercept(w.closeInterceptor); err != nil {
		return err
	}
	return w.actualStore.Close(ctx)
}

func (w *EventStoreWrapper) intercept(f Interceptor) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if f == nil {
		return nil
	}
	return f()
}

// compile-time check whether the EventStoreWrapper implements the EventStore interface
var _ watcher.EventStore = &EventStoreWrapper{}
