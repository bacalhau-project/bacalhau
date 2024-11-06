package ncl

import (
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

// NoopMessageFilter is a no-op message filter
type NoopMessageFilter struct{}

// ShouldFilter always returns false
func (n NoopMessageFilter) ShouldFilter(_ *envelope.Metadata) bool {
	return false
}

// compile time check for the NoopMessageFilter interface
var _ MessageFilter = NoopMessageFilter{}
