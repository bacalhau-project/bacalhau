package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// ComputeHandlerParams defines parameters for creating a new ComputeHandler.
type ComputeHandlerParams struct {
	Name            string
	Conn            *nats.Conn
	ComputeEndpoint compute.Endpoint
}

// ComputeHandler handles NATS messages for compute operations.
type ComputeHandler struct {
	name            string
	conn            *nats.Conn
	computeEndpoint compute.Endpoint
	subscription    *nats.Subscription
}

// handlerWithResponse represents a function that processes a request and returns a response.
type handlerWithResponse[Request, Response any] func(context.Context, Request) (Response, error)

// NewComputeHandler creates a new ComputeHandler.
func NewComputeHandler(params ComputeHandlerParams) (*ComputeHandler, error) {
	handler := &ComputeHandler{
		name:            params.Name,
		conn:            params.Conn,
		computeEndpoint: params.ComputeEndpoint,
	}

	subject := computeEndpointSubscribeSubject(handler.name)
	subscription, err := handler.conn.Subscribe(subject, func(m *nats.Msg) {
		handleRequest(m, handler)
	})
	if err != nil {
		return nil, err
	}
	handler.subscription = subscription
	log.Debug().Msgf("ComputeHandler %s subscribed to %s", handler.name, subject)
	return handler, nil
}

// handleRequest handles incoming NATS messages.
func handleRequest(msg *nats.Msg, handler *ComputeHandler) {
	ctx := context.Background()

	subjectParts := strings.Split(msg.Subject, ".")
	method := subjectParts[len(subjectParts)-1]

	switch method {
	case AskForBid:
		processAndRespond(ctx, msg, handler.computeEndpoint.AskForBid)
	case BidAccepted:
		processAndRespond(ctx, msg, handler.computeEndpoint.BidAccepted)
	case BidRejected:
		processAndRespond(ctx, msg, handler.computeEndpoint.BidRejected)
	case CancelExecution:
		processAndRespond(ctx, msg, handler.computeEndpoint.CancelExecution)
	case ExecutionLogs:
		processAndRespond(ctx, msg, handler.computeEndpoint.ExecutionLogs)
	default:
		// Noop, not subscribed to this method
		return
	}
}

// processAndRespond processes the request and sends a response.
func processAndRespond[Request, Response any](ctx context.Context, msg *nats.Msg, f handlerWithResponse[Request, Response]) {
	response, err := processRequest(ctx, msg, f)
	if err != nil {
		log.Ctx(ctx).Error().Err(err)
	}

	// We will wrap up the response/error in a Result type which can be decoded by the proxy itself.
	result := newResult(response, err)

	err = sendResponse(result, msg)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error sending response: %s", err)
	}
}

// processRequest decodes the request, invokes the handler, and returns the response.
func processRequest[Request, Response any](
	ctx context.Context, msg *nats.Msg, f handlerWithResponse[Request, Response]) (*Response, error) {
	request := new(Request)
	err := json.Unmarshal(msg.Data, request)
	if err != nil {
		return nil, fmt.Errorf("error decoding %s: %s", reflect.TypeOf(request).Name(), err)
	}

	response, err := f(ctx, *request)
	if err != nil {
		return nil, fmt.Errorf("error in handler %s: %s", reflect.TypeOf(request).Name(), err)
	}

	return &response, nil
}

// sendResponse marshals the response and sends it back to the requester.
func sendResponse[Response any](result Result[Response], msg *nats.Msg) error {
	resultData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("error encoding %s: %s", reflect.TypeOf(result.Response).Name(), err)
	}

	return msg.Respond(resultData)
}
