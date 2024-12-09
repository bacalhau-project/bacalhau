package proxy

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
)

type ManagementHandlerParams struct {
	Conn               *nats.Conn
	ManagementEndpoint bprotocol.ManagementEndpoint
}

// Management handles NATS legacy for cluster management
type ManagementHandler struct {
	conn     *nats.Conn
	endpoint bprotocol.ManagementEndpoint
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

// handle handles incoming NATS legacy.
func (h *ManagementHandler) handle(msg *nats.Msg) {
	ctx := context.Background()

	subjectParts := strings.Split(msg.Subject, ".")
	method := subjectParts[len(subjectParts)-1]

	switch method {
	case RegisterNode:
		response, err := h.processRegistration(ctx, msg)
		asyncResponse := concurrency.NewAsyncResult(response, err)

		if err := sendResponse(h.conn, msg.Reply, asyncResponse); err != nil {
			log.Ctx(ctx).Error().Msgf("error sending registration response: %s", err)
		}

	case UpdateNodeInfo:
		response, err := h.processUpdateInfo(ctx, msg)
		asyncResponse := concurrency.NewAsyncResult(response, err)

		if err := sendResponse(h.conn, msg.Reply, asyncResponse); err != nil {
			log.Ctx(ctx).Error().Msgf("error sending update info response: %s", err)
		}

	case UpdateResources:
		response, err := h.processUpdateResources(ctx, msg)
		asyncResponse := concurrency.NewAsyncResult(response, err)

		if err := sendResponse(h.conn, msg.Reply, asyncResponse); err != nil {
			log.Ctx(ctx).Error().Msgf("error sending update resources response: %s", err)
		}
	default:
		return
	}
}

func (h *ManagementHandler) processRegistration(ctx context.Context, msg *nats.Msg) (*legacy.RegisterResponse, error) {
	request := new(legacy.RegisterRequest)
	err := json.Unmarshal(msg.Data, request)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error decoding %s: %s", reflect.TypeOf(request), err)
		return nil, err
	}

	return h.endpoint.Register(ctx, *request)
}

func (h *ManagementHandler) processUpdateInfo(ctx context.Context, msg *nats.Msg) (*legacy.UpdateInfoResponse, error) {
	request := new(legacy.UpdateInfoRequest)
	err := json.Unmarshal(msg.Data, request)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error decoding %s: %s", reflect.TypeOf(request), err)
		return nil, err
	}

	return h.endpoint.UpdateInfo(ctx, *request)
}

func (h *ManagementHandler) processUpdateResources(ctx context.Context, msg *nats.Msg) (*legacy.UpdateResourcesResponse, error) {
	request := new(legacy.UpdateResourcesRequest)
	err := json.Unmarshal(msg.Data, request)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error decoding %s: %s", reflect.TypeOf(request), err)
		return nil, err
	}

	return h.endpoint.UpdateResources(ctx, *request)
}
