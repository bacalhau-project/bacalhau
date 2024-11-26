//go:generate mockgen --source types.go --destination mocks.go --package transport
package transport

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
)

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

// GenerateMsgID Message ID generation helper
func GenerateMsgID(event watcher.Event) string {
	return fmt.Sprintf("seq-%d", event.SeqNum)
}
