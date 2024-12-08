package nodes

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

const errComponent = "NodesManager"

const (
	MultipleNodesFound bacerrors.ErrorCode = "MultipleNodesFound"
	ConflictNodeState  bacerrors.ErrorCode = "ConflictNodeState"
	HandshakeRequired  bacerrors.ErrorCode = "HandshakeRequired"
	ConcurrentUpdate   bacerrors.ErrorCode = "ConcurrentUpdate"
)

// NewErrNodeNotFound returns a standardized error for when a node is not found
func NewErrNodeNotFound(nodeID string) bacerrors.Error {
	return bacerrors.New("node not found: %s", nodeID).
		WithCode(bacerrors.NotFoundError).
		WithComponent(errComponent)
}

// NewErrMultipleNodesFound returns a standardized error for when multiple nodes match a prefix
func NewErrMultipleNodesFound(nodeIDPrefix string, matchingNodeIDs []string) bacerrors.Error {
	if len(matchingNodeIDs) > 3 {
		matchingNodeIDs = matchingNodeIDs[:3]
		matchingNodeIDs = append(matchingNodeIDs, "...")
	}
	return bacerrors.New("multiple nodes found for prefix: %s, matching IDs: %v", nodeIDPrefix, matchingNodeIDs).
		WithCode(MultipleNodesFound).
		WithHTTPStatusCode(http.StatusConflict).
		WithComponent(errComponent).
		WithHint("Use a more specific node ID prefix")
}

// NewErrHandshakeRequired returns a standardized error for when a handshake is required
func NewErrHandshakeRequired(nodeID string) bacerrors.Error {
	return bacerrors.New("node %s not connected - handshake required", nodeID).
		WithCode(HandshakeRequired).
		WithComponent(errComponent).
		WithHTTPStatusCode(http.StatusPreconditionRequired).
		WithRetryable()
}

// NewErrNodeAlreadyApproved returns a standardized error for when a node is already approved
func NewErrNodeAlreadyApproved(nodeID string) bacerrors.Error {
	return bacerrors.New("node %s already approved", nodeID).
		WithCode(ConflictNodeState).
		WithHTTPStatusCode(http.StatusConflict).
		WithComponent(errComponent)
}

// NewErrNodeAlreadyRejected returns a standardized error for when a node is already rejected
func NewErrNodeAlreadyRejected(nodeID string) bacerrors.Error {
	return bacerrors.New("node %s already rejected", nodeID).
		WithCode(ConflictNodeState).
		WithHTTPStatusCode(http.StatusConflict).
		WithComponent(errComponent)
}

// NewErrConcurrentModification returns a standardized error for concurrent update conflicts
func NewErrConcurrentModification() bacerrors.Error {
	return bacerrors.New("concurrent modification detected").
		WithCode(ConcurrentUpdate).
		WithHTTPStatusCode(http.StatusConflict).
		WithComponent(errComponent).
		WithRetryable().
		WithHint("Request should be retried")
}
