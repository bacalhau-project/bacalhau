package proxy

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type CallbackHandlerParams struct {
	Name     string
	Conn     *nats.Conn
	Callback compute.Callback
}

// CallbackHandler is a handler for callback events that registers for incoming nats requests to Bacalhau callback
// protocol, and delegates the handling of the request to the provided callback.
type CallbackHandler struct {
	name     string
	conn     *nats.Conn
	callback compute.Callback
}

type callbackHandler[Request any] func(context.Context, Request)

func NewCallbackHandler(params CallbackHandlerParams) (*CallbackHandler, error) {
	handler := &CallbackHandler{
		name:     params.Name,
		conn:     params.Conn,
		callback: params.Callback,
	}

	subject := callbackSubscribeSubject(handler.name)
	_, err := handler.conn.Subscribe(subject, func(m *nats.Msg) {
		handler.handle(m)
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("ComputeHandler %s subscribed to %s", handler.name, subject)
	return handler, nil
}

// handle handles incoming NATS messages.
func (h *CallbackHandler) handle(msg *nats.Msg) {
	ctx := context.Background()

	subjectParts := strings.Split(msg.Subject, ".")
	method := subjectParts[len(subjectParts)-1]

	switch method {
	case OnBidComplete:
		processCallback(ctx, msg, h.callback.OnBidComplete) //nolint
	case OnRunComplete:
		processCallback(ctx, msg, h.callback.OnRunComplete) //nolint
	case OnCancelComplete:
		processCallback(ctx, msg, h.callback.OnCancelComplete) //nolint
	case OnComputeFailure:
		processCallback(ctx, msg, h.callback.OnComputeFailure) //nolint
	default:
		// Noop, not subscribed to this method
		return
	}
}

func processCallback[Request any](
	ctx context.Context,
	msg *nats.Msg,
	f callbackHandler[Request]) {
	request := new(Request)
	err := json.Unmarshal(msg.Data, request)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error decoding %s: %s", reflect.TypeOf(request), err)
		return
	}

	go f(ctx, *request)
}
