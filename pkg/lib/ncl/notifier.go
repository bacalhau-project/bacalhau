package ncl

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

// NoopNotifier is a ProcessingNotifier that does nothing
type NoopNotifier struct{}

// OnProcessed does nothing
func (n *NoopNotifier) OnProcessed(ctx context.Context, message *envelope.Message) {
	// no-op
}

// compile time check for interface conformance
var _ ProcessingNotifier = &NoopNotifier{}
