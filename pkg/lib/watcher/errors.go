package watcher

import (
	"errors"
	"fmt"
)

var (
	ErrWatcherAlreadyExists = errors.New("watcher already exists")
	ErrWatcherNotFound      = errors.New("watcher not found")
	ErrCheckpointNotFound   = errors.New("checkpoint not found")
	ErrNoHandler            = errors.New("no handler configured")
	ErrHandlerExists        = errors.New("handler already exists")
)

// WatcherError represents an error related to a specific watcher
type WatcherError struct {
	WatcherID string
	Err       error
}

func NewWatcherError(watcherID string, err error) *WatcherError {
	return &WatcherError{WatcherID: watcherID, Err: err}
}

func (e *WatcherError) Error() string {
	return fmt.Sprintf("watcher error for watcher %s: %v", e.WatcherID, e.Err)
}

func (e *WatcherError) Unwrap() error {
	return e.Err
}

// CheckpointError represents an error related to checkpointing
type CheckpointError struct {
	WatcherID string
	Err       error
}

// NewCheckpointError creates a new CheckpointError
func NewCheckpointError(watcherID string, err error) *CheckpointError {
	return &CheckpointError{WatcherID: watcherID, Err: err}
}

func (e *CheckpointError) Error() string {
	return fmt.Sprintf("checkpoint error for watcher %s: %v", e.WatcherID, e.Err)
}

func (e *CheckpointError) Unwrap() error {
	return e.Err
}

// EventHandlingError represents an error that occurred during event processing
type EventHandlingError struct {
	WatcherID string
	EventID   uint64
	Err       error
}

// NewEventHandlingError creates a new EventHandlingError
func NewEventHandlingError(watcherID string, eventID uint64, err error) *EventHandlingError {
	return &EventHandlingError{WatcherID: watcherID, EventID: eventID, Err: err}
}

func (e *EventHandlingError) Error() string {
	return fmt.Sprintf("error processing event %d for watcher %s: %v", e.EventID, e.WatcherID, e.Err)
}

func (e *EventHandlingError) Unwrap() error {
	return e.Err
}

type SerializationError struct {
	Event Event
	Err   error
}

func NewSerializationError(event Event, err error) *SerializationError {
	return &SerializationError{Event: event, Err: err}
}

func (e *SerializationError) Error() string {
	return fmt.Sprintf("serialization error for event %+v: %v", e.Event, e.Err)
}

func (e *SerializationError) Unwrap() error {
	return e.Err
}

type DeserializationError struct {
	Err error
}

func NewDeserializationError(err error) *DeserializationError {
	return &DeserializationError{Err: err}
}

func (e *DeserializationError) Error() string {
	return fmt.Sprintf("deserialization error: %v", e.Err)
}

func (e *DeserializationError) Unwrap() error {
	return e.Err
}
