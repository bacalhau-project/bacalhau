package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type ManagementHandlerParams struct {
	Conn               *nats.Conn
	ManagementEndpoint compute.ManagementEndpoint
}

// Management handles NATS messages for cluster management
type ManagementHandler struct {
	conn     *nats.Conn
	endpoint compute.ManagementEndpoint
}

func NewManagementHandler(params ManagementHandlerParams) (*ManagementHandler, error) {
	handler := &ManagementHandler{
		conn:     params.Conn,
		endpoint: params.ManagementEndpoint,
	}

	subject := managementSubscribeSubject()
	_, err := handler.conn.Subscribe(subject, func(m *nats.Msg) {
		handler.handle(m)
	})
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("ManagementHandler subscribed to %s", subject)
	return handler, nil
}

// handle handles incoming NATS messages.
func (h *ManagementHandler) handle(msg *nats.Msg) {
	ctx := context.Background()

	subjectParts := strings.Split(msg.Subject, ".")
	method := subjectParts[len(subjectParts)-1]

	fmt.Println("--------------", method)

	switch method {
	case RegisterNode:
		if _, err := h.processRegistration(ctx, msg); err != nil {
			log.Ctx(ctx).Error().Msgf("error processing registration: %s", err)
		}
	case UpdateNodeInfo:
		if _, err := h.processUpdateInfo(ctx, msg); err != nil {
			log.Ctx(ctx).Error().Msgf("error processing info update: %s", err)
		}
	default:
		return
	}
}

func (h *ManagementHandler) processRegistration(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	request := new(requests.RegisterRequest)
	err := json.Unmarshal(msg.Data, request)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error decoding %s: %s", reflect.TypeOf(request), err)
		return nil, err
	}

	_, err = h.endpoint.Register(ctx, *request)
	// TODO(ross): Process respose (and turn into error if necessary)
	return nil, err
}

func (h *ManagementHandler) processUpdateInfo(ctx context.Context, msg *nats.Msg) (*nats.Msg, error) {
	request := new(requests.UpdateInfoRequest)
	err := json.Unmarshal(msg.Data, request)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error decoding %s: %s", reflect.TypeOf(request), err)
		return nil, err
	}

	_, err = h.endpoint.UpdateInfo(ctx, *request)
	// TODO(ross): Process respose (and turn into error if necessary)
	return nil, err
}
