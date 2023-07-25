package jobstore

import "time"

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

const DefaultWatchChannelSize = 64

// WatchEvent is the message passed through the watcher whenever a
// specific event occurs on a specific type, as requested when creating
// the watcher.
type WatchEvent struct {
	Kind      StoreWatcherType
	Event     StoreEventType
	Object    []byte
	Timestamp int64
}

func NewWatchEvent(kind StoreWatcherType, event StoreEventType, object []byte) WatchEvent {
	return WatchEvent{
		Kind:      kind,
		Event:     event,
		Object:    append([]byte(nil), object...),
		Timestamp: time.Now().Unix(),
	}
}

// Watcher is used by the jobstore to keep a record of parties interested in events happening
// in the jobstore.  This allows for watching of job and execution types (or both), and for
// create, update and delete events (or any combination).
type Watcher struct {
	types       StoreWatcherType // a bitmask of types being watched
	events      StoreEventType   // a bitmask of events being watched
	channelSize int
	channel     chan WatchEvent
}

func NewWatcher(types StoreWatcherType, events StoreEventType) *Watcher {
	return &Watcher{
		types:       types,
		events:      events,
		channelSize: DefaultWatchChannelSize,
		channel:     make(chan WatchEvent, DefaultWatchChannelSize),
	}
}

func (w *Watcher) IsWatchingType(kind StoreWatcherType) bool {
	return w.types&kind > 0
}

func (w *Watcher) IsWatchingEvent(event StoreEventType) bool {
	return w.events&event > 0
}

func (w *Watcher) Channel() chan WatchEvent {
	return w.channel
}

func (w *Watcher) WriteEvent(kind StoreWatcherType, event StoreEventType, object []byte, allowBlock bool) bool {
	// If we don't want to block, return a fail when the channel is currently full.
	// By default, we'll block and wait for a space in the channel
	if len(w.channel) == w.channelSize && !allowBlock {
		return false
	}

	w.channel <- WatchEvent{
		Kind:   kind,
		Event:  event,
		Object: object,
	}
	return true
}

func (w *Watcher) Close() {
	close(w.channel)
}
