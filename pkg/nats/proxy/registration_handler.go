package proxy

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// RegistrationHandlerParams defines parameters for creating a new RegistrationHandler
type RegistrationHandlerParams struct {
	NodeID               string
	Conn                 *nats.Conn
	RegistrationEndpoint requester.RegistrationEndpoint
}

// ComputeHandler handles NATS messages for compute operations.
type RegistrationHandler struct {
	nodeID   string
	conn     *nats.Conn
	endpoint requester.RegistrationEndpoint
}

// NewComputeHandler creates a new ComputeHandler.
func NewRegistrationHandler(params RegistrationHandlerParams) (*RegistrationHandler, error) {
	handler := &RegistrationHandler{
		nodeID:   params.NodeID,
		conn:     params.Conn,
		endpoint: params.RegistrationEndpoint,
	}

	subject := registrationSubscribeSubject()
	_, err := handler.conn.Subscribe(subject, func(m *nats.Msg) {
		handler.handle(m)
	})
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("ComputeHandler %s subscribed to %s", handler.nodeID, subject)
	return handler, nil
}

// handle handles incoming NATS messages.
func (h *RegistrationHandler) handle(msg *nats.Msg) {
	ctx := context.Background()

	subjectParts := strings.Split(msg.Subject, ".")
	method := subjectParts[len(subjectParts)-1]

	switch method {
	case Register:
		if err := h.processRegistration(ctx, msg); err != nil {
			log.Ctx(ctx).Error().Msgf("error processing registration: %s", err)
		}
	default:
		return
	}
}

func (h *RegistrationHandler) processRegistration(ctx context.Context, msg *nats.Msg) error {
	request := new(requester.RegisterRequest)
	err := json.Unmarshal(msg.Data, request)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error decoding %s: %s", reflect.TypeOf(request), err)
		return err
	}

	return h.endpoint.Register(ctx, *request)
}
