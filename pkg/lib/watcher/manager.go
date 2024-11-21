package watcher

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	DefaultShutdownTimeout = 30 * time.Second
)

// manager handles lifecycle of multiple watchers with shared resources
type manager struct {
	store    EventStore
	watchers map[string]Watcher
	mu       sync.RWMutex
}

// NewManager creates a new Manager with the given EventStore.
//
// Example usage:
//
//	store := // initialize your event store
//	manager := NewManager(store)
//	defer manager.Stop(context.Background())
//
//	watcher, err := manager.Create(context.Background(), "myWatcher")
//	if err != nil {
//	    // handle error
//	}
func NewManager(store EventStore) Manager {
	return &manager{
		store:    store,
		watchers: make(map[string]Watcher),
	}
}

// Create creates an unstarted watcher. SetHandler must be called before
// Start can be called successfully.
func (m *manager) Create(ctx context.Context, watcherID string, opts ...WatchOption) (Watcher, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if a watcher with this ID already exists
	if _, exists := m.watchers[watcherID]; exists {
		return nil, NewWatcherError(watcherID, ErrWatcherAlreadyExists)
	}

	w, err := New(ctx, watcherID, m.store, opts...)
	if err != nil {
		return nil, err
	}

	m.watchers[w.ID()] = w
	return w, nil
}

// Lookup retrieves a specific watcher by ID
func (m *manager) Lookup(watcherID string) (Watcher, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w, exists := m.watchers[watcherID]
	if !exists {
		return nil, NewWatcherError(watcherID, ErrWatcherNotFound)
	}

	return w, nil
}

// Stop gracefully shuts down the manager and all its watchers
func (m *manager) Stop(ctx context.Context) error {
	log.Ctx(ctx).Debug().Msg("Shutting down manager")

	// Create a timeout context if the parent context doesn't have a deadline
	timeoutCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		timeoutCtx, cancel = context.WithTimeout(ctx, DefaultShutdownTimeout)
		defer cancel()
	}

	var wg sync.WaitGroup

	// Take a snapshot of watchers under lock
	m.mu.RLock()
	watchers := make([]Watcher, 0, len(m.watchers))
	for _, w := range m.watchers {
		watchers = append(watchers, w)
	}
	m.mu.RUnlock()

	// Stop all watchers concurrently
	for i := range watchers {
		w := watchers[i]
		wg.Add(1)
		go func(w Watcher) {
			defer wg.Done()
			w.Stop(timeoutCtx)
		}(w)
	}

	// Wait for completion or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Ctx(ctx).Debug().Msg("manager shutdown complete")
		return nil
	case <-timeoutCtx.Done():
		log.Ctx(ctx).Warn().Msg("manager shutdown timed out")
		return timeoutCtx.Err()
	}
}

// compile time check for interface implementation
var _ Manager = &manager{}
