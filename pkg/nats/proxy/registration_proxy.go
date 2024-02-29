package proxy

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type RegistrationProxyParams struct {
	Conn *nats.Conn
}

// RegistrationProxy is a proxy for a compute node to register itself with a requester node.
type RegistrationProxy struct {
	conn *nats.Conn
}

// NewRegistrationProxy creates a new RegistrationProxy for the local compute node
// bound to a provided NATS connection.
func NewRegistrationProxy(params RegistrationProxyParams) *RegistrationProxy {
	proxy := &RegistrationProxy{
		conn: params.Conn,
	}
	return proxy
}

// Register sends a `requester.RegisterRequest` containing the current compute node's
// NodeID to the requester node.
func (p *RegistrationProxy) Register(ctx context.Context, request requests.RegisterRequest) error {
	data, err := json.Marshal(request)
	if err != nil {
		log.Ctx(ctx).Error().Err(errors.WithStack(err)).Msgf("%s: failed to marshal request", reflect.TypeOf(request))
		return err
	}

	// We submit registration requests to all subscribers to the registration subject,
	// and not to a single specific node.
	subject := registrationPublishSubject(Register)
	log.Ctx(ctx).Trace().Msgf("Sending registration request to subject %s", subject)

	if err = p.conn.Publish(subject, data); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("error sending request to subject %s", subject)
		return err
	}

	return nil
}
