package core

import (
	"context"
	"time"
)

// ConnectionState represents the current state of a connection
type ConnectionState int

const (
	Disconnected ConnectionState = iota
	Connecting
	Connected
)

// String returns the string representation of the connection state
func (c ConnectionState) String() string {
	switch c {
	case Disconnected:
		return "Disconnected"
	case Connecting:
		return "Connecting"
	case Connected:
		return "Connected"
	default:
		return "Unknown"
	}
}

type Checkpointer interface {
	Checkpoint(ctx context.Context, name string, sequenceNumber uint64) error
	GetCheckpoint(ctx context.Context, name string) (uint64, error)
}

// ConnectionStateHandler is called when connection state changes
type ConnectionStateHandler func(ConnectionState)

type ConnectionHealth struct {
	LastSuccessfulHeartbeat time.Time
	LastSuccessfulUpdate    time.Time
	CurrentState            ConnectionState
	ConsecutiveFailures     int
	LastError               error
	ConnectedSince          time.Time
}
