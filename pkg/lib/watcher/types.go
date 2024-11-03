//go:generate mockgen -source=types.go -destination=mocks.go -package=watcher
package watcher

import (
	"context"
	"time"
)

type State string

const (
	StateIdle     State = "idle"
	StateRunning  State = "running"
	StateStopping State = "stopping"
	StateStopped  State = "stopped"
)

type Stats struct {
	ID                     string
	State                  State
	NextEventIterator      EventIterator // Next event iterator for the watcher
	LastProcessedSeqNum    uint64        // SeqNum of the last event processed by this watcher
	LastProcessedEventTime time.Time     // timestamp of the last processed event
	LastListenTime         time.Time     // timestamp of the last successful listen operation
}

// StoreEventRequest represents the input for creating an event.
type StoreEventRequest struct {
	Operation  Operation   `json:"operation"`
	ObjectType string      `json:"objectType"`
	Object     interface{} `json:"object"`
}

// Watcher represents a single event watcher.
type Watcher interface {
	// ID returns the unique identifier for the watcher.
	ID() string

	// Stats returns the current statistics for the watcher.
	Stats() Stats

	// Stop gracefully stops the watcher.
	Stop(ctx context.Context)

	// Checkpoint saves the current progress of the watcher.
	Checkpoint(ctx context.Context, eventSeqNum uint64) error

	// SeekToOffset moves the watcher to a specific event sequence number.
	SeekToOffset(ctx context.Context, eventSeqNum uint64) error
}

// EventHandler is an interface for handling events.
type EventHandler interface {
	// HandleEvent processes a single event.
	// It returns an error if the event processing fails.
	// Implementations MUST honor context cancellation and return immediately when ctx.Done()
	HandleEvent(ctx context.Context, event Event) error
}

// Registry manages multiple event watchers and provides methods to watch for events.
type Registry interface {
	// Watch starts watching for events with the given options.
	// It returns a Watcher that can be used to receive events.
	Watch(ctx context.Context, watcherID string, handler EventHandler, opts ...WatchOption) (Watcher, error)

	// GetWatcher retrieves an existing watcher by its ID.
	GetWatcher(watcherID string) (Watcher, error)

	// Stop gracefully shuts down the registry and all its watchers.
	Stop(ctx context.Context) error
}

// EventStore defines the interface for event storage and retrieval.
type EventStore interface {
	// StoreEvent stores a new event in the event store.
	StoreEvent(ctx context.Context, request StoreEventRequest) error

	// GetEvents retrieves events based on the provided query parameters.
	GetEvents(ctx context.Context, request GetEventsRequest) (*GetEventsResponse, error)

	// GetLatestEventNum returns the sequence number of the latest event.
	GetLatestEventNum(ctx context.Context) (uint64, error)

	// StoreCheckpoint saves a checkpoint for a specific watcher.
	StoreCheckpoint(ctx context.Context, watcherID string, eventSeqNum uint64) error

	// GetCheckpoint retrieves the checkpoint for a specific watcher.
	GetCheckpoint(ctx context.Context, watcherID string) (uint64, error)

	// Close closes the event store.
	Close(ctx context.Context) error
}

// Serializer defines the interface for event serialization and deserialization.
type Serializer interface {
	// Marshal serializes an Event into a byte slice.
	Marshal(event Event) ([]byte, error)

	// Unmarshal deserializes a byte slice into an Event.
	Unmarshal(data []byte, event *Event) error
}

// EventFilter defines criteria for filtering events.
type EventFilter struct {
	// ObjectTypes is a list of object types to include in the filter.
	ObjectTypes []string

	// Operations is a list of operations to include in the filter.
	Operations []Operation
}

// GetEventsRequest defines parameters for querying events.
type GetEventsRequest struct {
	// EventIterator specifies where to start reading events.
	EventIterator EventIterator

	// Limit is the maximum number of events to return.
	Limit int

	// Filter specifies criteria for filtering events.
	Filter EventFilter
}

type GetEventsResponse struct {
	Events            []Event
	NextEventIterator EventIterator
}
