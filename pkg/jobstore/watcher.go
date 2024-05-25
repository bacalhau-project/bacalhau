package jobstore

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const defaultWatchChannelSize = 64
const maxBlockingTimeToLog = 20 * time.Millisecond

type StoreWatcherType int

const (
	JobWatcher StoreWatcherType = 1 << iota
	ExecutionWatcher
	EvaluationWatcher
)

func (s StoreWatcherType) String() string {
	switch s {
	case JobWatcher:
		return "JobWatcher"
	case ExecutionWatcher:
		return "ExecutionWatcher"
	case EvaluationWatcher:
		return "EvaluationWatcher"
	}
	return "Unknown type"
}

type StoreEventType int

const (
	CreateEvent StoreEventType = 1 << iota
	UpdateEvent
	DeleteEvent
)

func (s StoreEventType) String() string {
	switch s {
	case CreateEvent:
		return "CreateEvent"
	case UpdateEvent:
		return "UpdateEvent"
	case DeleteEvent:
		return "DeleteEvent"
	}
	return "Unknown event"
}

// WatchEvent is the message passed through the watcher whenever a
// specific event occurs on a specific type, as requested when creating
// the watcher.
type WatchEvent struct {
	Kind      StoreWatcherType
	Event     StoreEventType
	Object    any
	Timestamp int64
}

func NewWatchEvent(kind StoreWatcherType, event StoreEventType, object any) *WatchEvent {
	return &WatchEvent{
		Kind:   kind,
		Event:  event,
		Object: object,
		// TODO(ross): Add a timestamp from the actual time of the event (rather than the
		// time at which this event was created).
	}
}

type WatcherOption func(*Watcher)

func WithChannelSize(size int) WatcherOption {
	return func(w *Watcher) {
		w.channel = make(chan *WatchEvent, size)
	}
}

type FullChannelBehavior int

const (
	WatcherDrop FullChannelBehavior = iota
	WatcherDropOldest
	WatcherBlock
)

func WithFullChannelBehavior(behavior FullChannelBehavior) WatcherOption {
	return func(w *Watcher) {
		w.fullChannelBehavior = behavior
	}
}

// Watcher is used by the jobstore to keep a record of parties interested in events happening
// in the jobstore.  This allows for watching of job and execution types (or both), and for
// create, update and delete events (or any combination).
type Watcher struct {
	types               StoreWatcherType // a bitmask of types being watched
	events              StoreEventType   // a bitmask of events being watched
	fullChannelBehavior FullChannelBehavior
	channel             chan *WatchEvent
	closed              bool
	mu                  sync.Mutex
}

func NewWatcher(ctx context.Context, types StoreWatcherType, events StoreEventType, options ...WatcherOption) *Watcher {
	w := &Watcher{
		types:               types,
		events:              events,
		fullChannelBehavior: WatcherDrop,
	}

	for _, option := range options {
		option(w)
	}

	if w.channel == nil {
		w.channel = make(chan *WatchEvent, defaultWatchChannelSize)
	}

	// Close the worker when the context is done
	go func() {
		<-ctx.Done()
		w.Close()
	}()

	return w
}

func (w *Watcher) IsWatchingType(kind StoreWatcherType) bool {
	return w.types&kind > 0
}

func (w *Watcher) IsWatchingEvent(event StoreEventType) bool {
	return w.events&event > 0
}

func (w *Watcher) Channel() chan *WatchEvent {
	return w.channel
}

func (w *Watcher) write(kind StoreWatcherType, event StoreEventType, object any) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	startTime := time.Now()

	newEvent := NewWatchEvent(kind, event, object)

	switch w.fullChannelBehavior {
	case WatcherBlock:
		w.channel <- newEvent
		if time.Since(startTime) > maxBlockingTimeToLog {
			log.Debug().Msgf("Watcher blocked for %v writing event %s:%s", time.Since(startTime), kind, event)
		}
		return true
	case WatcherDrop:
		select {
		case w.channel <- newEvent:
		default:
			// Channel is full, drop the new event
			log.Debug().Msgf("Watcher queue is full, dropping new event %s:%s", kind, event)
			return false
		}
	case WatcherDropOldest:
		select {
		case w.channel <- newEvent:
		default:
			// Channel is full, drop the oldest event and try again
			<-w.channel
			w.channel <- newEvent
			log.Debug().Msgf("Watcher queue is full, dropping oldest event and adding new event %s:%s", kind, event)
			return true
		}
	}
	return false
}

func (w *Watcher) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed {
		close(w.channel)
		w.closed = true
	}
}

// WatchersManager is a helper type that can be used by the different
// jobstore implementations to manage watchers. It allows for the creation
// of new watchers, the writing of events to all interested watchers and
// the cleanup of closed watchers.
type WatchersManager struct {
	watchers    map[string]*Watcher
	watcherLock sync.Mutex
}

func NewWatchersManager() *WatchersManager {
	return &WatchersManager{
		watchers: make(map[string]*Watcher),
	}
}

// NewWatcher creates a new Watcher managed by the WatchersManager
func (w *WatchersManager) NewWatcher(
	ctx context.Context, types StoreWatcherType, events StoreEventType, options ...WatcherOption) *Watcher {
	watcher := NewWatcher(ctx, types, events, options...)
	watcherID := uuid.NewString()

	w.watcherLock.Lock()
	defer w.watcherLock.Unlock()
	w.watchers[watcherID] = watcher
	return watcher
}

// Write writes an event to all interested watchers
// If a watcher is closed, it is removed from the list of watchers
func (w *WatchersManager) Write(kind StoreWatcherType, event StoreEventType, object any) {
	closedWatchers := make([]string, 0)
	for id, watcher := range w.watchers {
		if watcher.closed {
			closedWatchers = append(closedWatchers, id)
			continue
		}
		if watcher.IsWatchingType(kind) && watcher.IsWatchingEvent(event) {
			watcher.write(kind, event, object)
		}
	}
	for _, id := range closedWatchers {
		delete(w.watchers, id)
	}
}

// Close closes all watchers
func (w *WatchersManager) Close() {
	w.watcherLock.Lock()
	defer w.watcherLock.Unlock()
	for _, watcher := range w.watchers {
		watcher.Close()
	}
}
