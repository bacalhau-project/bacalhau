package orchestrator

import (
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
)

const errComponent = "ConnectionManager"

// NewErrHandshakeRequired returns a standardized error for when a handshake is required
func NewErrHandshakeRequired(nodeID string) bacerrors.Error {
	return nodes.NewErrHandshakeRequired(nodeID).
		WithComponent(errComponent)
}
