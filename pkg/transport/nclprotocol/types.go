//go:generate mockgen --source types.go --destination mocks.go --package nclprotocol

package nclprotocol

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
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
	StartTime               time.Time
	LastSuccessfulHeartbeat time.Time
	LastSuccessfulUpdate    time.Time
	CurrentState            ConnectionState
	ConsecutiveFailures     int
	LastError               error
	ConnectedSince          time.Time
}

const (
	KeySeqNum = "Bacalhau-SeqNum"
)

// MessageCreator defines how events from the watcher are converted into
// messages for publishing. This is the primary extension point for customizing
// transport behavior.
type MessageCreator interface {
	// CreateMessage converts a watcher event into a message envelope.
	// Returns nil if no message should be published for this event.
	// Any error will halt event processing.
	CreateMessage(event watcher.Event) (*envelope.Message, error)
}

type MessageCreatorFactory interface {
	CreateMessageCreator(ctx context.Context, nodeID string) MessageCreator
}

// GenerateMsgID Message ID generation helper
func GenerateMsgID(event watcher.Event) string {
	return fmt.Sprintf("seq-%d", event.SeqNum)
}
