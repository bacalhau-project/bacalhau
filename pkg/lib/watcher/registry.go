package watcher

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
)

// registry manages multiple event watchers
type registry struct {
	store    EventStore
	watchers map[string]*watcher
	mu       sync.RWMutex
}

// NewRegistry creates a new Registry with the given EventStore.
//
// Example usage:
//
//	store := // initialize your event store
//	registry := NewRegistry(store)
//	defer registry.Stop(context.Background())
//
//	watcher, err := registry.Watch(context.Background(), "myWatcher", myEventHandler)
//	if err != nil {
//	    // handle error
//	}
func NewRegistry(store EventStore) Registry {
	return &registry{
		store:    store,
		watchers: make(map[string]*watcher),
	}
}

// Watch starts watching for events with the given options
func (r *registry) Watch(ctx context.Context, watcherID string, handler EventHandler, opts ...WatchOption) (Watcher, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if a watcher with this ID already exists
	if w, exists := r.watchers[watcherID]; exists {
		if w.Stats().State != StateStopped {
			return nil, NewWatcherError(watcherID, ErrWatcherAlreadyExists)
		}
	}

	w, err := newWatcher(ctx, watcherID, handler, r.store, opts...)
	if err != nil {
		return nil, err
	}

	r.watchers[w.ID()] = w
	go w.Start()
	return w, nil
}

// GetWatcher retrieves a specific watcher by ID
func (r *registry) GetWatcher(watcherID string) (Watcher, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	w, exists := r.watchers[watcherID]
	if !exists {
		return nil, NewWatcherError(watcherID, ErrWatcherNotFound)
	}

	return w, nil
}

// Stop gracefully shuts down the registry and all its watchers
func (r *registry) Stop(ctx context.Context) error {
	log.Ctx(ctx).Info().Msg("Shutting down registry")

	done := make(chan struct{})

	go func() {
		r.mu.RLock()
		defer r.mu.RUnlock()
		var wg sync.WaitGroup
		for _, w := range r.watchers {
			wg.Add(1)
			go func(w *watcher) {
				defer wg.Done()
				w.Stop(ctx)
			}(w)
		}
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Ctx(ctx).Info().Msg("registry shutdown complete")
		return nil
	case <-ctx.Done():
		log.Ctx(ctx).Warn().Msg("registry shutdown timed out")
		return ctx.Err()
	}
}

// compile time check for interface implementation
var _ Registry = &registry{}
