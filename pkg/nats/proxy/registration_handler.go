package proxy

import (
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/nats-io/nats.go"
)

// RegistrationHandlerParams defines parameters for creating a new RegistrationHandler
type RegistrationHandlerParams struct {
	Conn                 *nats.Conn
	RegistrationEndpoint requester.RegistrationEndpoint
}

// ComputeHandler handles NATS messages for compute operations.
type RegistrationHandler struct {
	conn *nats.Conn
}

// NewComputeHandler creates a new ComputeHandler.
func NewRegistrationHandler(params RegistrationHandlerParams) (*RegistrationHandler, error) {
	return &RegistrationHandler{
		conn: params.Conn,
	}, nil
}
