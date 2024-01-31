package jobstore

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
	Object    interface{}
	Timestamp int64
}

func NewWatchEvent(kind StoreWatcherType, event StoreEventType, object interface{}) WatchEvent {
	return WatchEvent{
		Kind:   kind,
		Event:  event,
		Object: object,
		// TODO(ross): Add a timestamp from the actual time of the event (rather than the
		// time at which this event was created).
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
	closed      bool
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

func (w *Watcher) WriteEvent(kind StoreWatcherType, event StoreEventType, object interface{}, allowBlock bool) bool {
	// If we don't want to block, return a fail when the channel is currently full.
	// By default, we'll block and wait for a space in the channel
	if len(w.channel) == w.channelSize && !allowBlock {
		return false
	}

	if w.closed {
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
	w.closed = true
	close(w.channel)
}
