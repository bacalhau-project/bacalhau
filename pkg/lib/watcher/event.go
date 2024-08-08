package watcher

import (
	"time"
)

// Event represents a single event in the event store.
type Event struct {
	SeqNum     uint64      `json:"seqNum"`
	Operation  Operation   `json:"operation"`
	ObjectType string      `json:"objectType"`
	Object     interface{} `json:"object"`
	Timestamp  time.Time   `json:"timestamp"`
}

// Operation represents the type of operation performed in an event.
type Operation string

const (
	// OperationCreate represents a creation event.
	OperationCreate Operation = "CREATE"
	// OperationUpdate represents an update event.
	OperationUpdate Operation = "UPDATE"
	// OperationDelete represents a deletion event.
	OperationDelete Operation = "DELETE"
)
