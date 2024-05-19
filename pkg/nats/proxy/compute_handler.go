package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/nats/stream"
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
	streamingClient *stream.ProducerClient
}

// handlerWithResponse represents a function that processes a request and returns a response.
type handlerWithResponse[Request, Response any] func(context.Context, Request) (Response, error)

// NewComputeHandler creates a new ComputeHandler.
func NewComputeHandler(params ComputeHandlerParams) (*ComputeHandler, error) {
	streamingClient, err := stream.NewProducerClient(stream.ProducerClientParams{
		Conn: params.Conn,
	})
	if err != nil {
		return nil, err
	}
	handler := &ComputeHandler{
		name:            params.Name,
		conn:            params.Conn,
		computeEndpoint: params.ComputeEndpoint,
		streamingClient: streamingClient,
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
		processAndRespond(ctx, handler.conn, msg, handler.computeEndpoint.AskForBid)
	case BidAccepted:
		processAndRespond(ctx, handler.conn, msg, handler.computeEndpoint.BidAccepted)
	case BidRejected:
		processAndRespond(ctx, handler.conn, msg, handler.computeEndpoint.BidRejected)
	case CancelExecution:
		processAndRespond(ctx, handler.conn, msg, handler.computeEndpoint.CancelExecution)
	case ExecutionLogs:
		processAndStream(ctx, handler.streamingClient, msg, handler.computeEndpoint.ExecutionLogs)
	default:
		// Noop, not subscribed to this method
		return
	}
}

// processAndRespond processes the request and sends a response.
func processAndRespond[Request, Response any](
	ctx context.Context, conn *nats.Conn, msg *nats.Msg, f handlerWithResponse[Request, Response]) {
	response, err := processRequest(ctx, msg, f)
	if err != nil {
		log.Ctx(ctx).Error().Err(err)
	}

	// We will wrap up the response/error in a Result type which can be decoded by the proxy itself.
	result := concurrency.NewAsyncResult(response, err)

	err = sendResponse(conn, msg.Reply, result)
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
func sendResponse[Response any](conn *nats.Conn, reply string, result *concurrency.AsyncResult[Response]) error {
	resultData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("error encoding %T: %s", result.Value, err)
	}

	return conn.Publish(reply, resultData)
}

func processAndStream[Request, Response any](ctx context.Context, streamingClient *stream.ProducerClient, msg *nats.Msg,
	f handlerWithResponse[Request, <-chan *concurrency.AsyncResult[Response]]) {
	if msg.Reply == "" {
		log.Ctx(ctx).Error().Msgf("streaming request on %s has no reply subject", msg.Subject)
		return
	}

	writer := streamingClient.NewWriter(msg.Reply)
	request := new(Request)
	err := json.Unmarshal(msg.Data, request)
	if err != nil {
		_ = writer.CloseWithCode(stream.CloseBadRequest,
			fmt.Sprintf("error decoding %s: %s", reflect.TypeOf(request).Name(), err))
		return
	}

	connDetails := &stream.ConnectionDetails{
		StreamId:            msg.Header.Get("StreamId"),
		ConnId:              msg.Header.Get("ConnId"),
		HeartBeatRequestSub: msg.Header.Get("StreamHeartBeatSub"),
	}

	streamingClient.AddConnDetails(ctx, connDetails)
	ch, err := f(ctx, *request)
	if err != nil {
		_ = writer.CloseWithCode(stream.CloseInternalServerErr,
			fmt.Sprintf("error in handler %s: %s", reflect.TypeOf(request).Name(), err))
		return
	}

	for res := range ch {
		_, err = writer.WriteObject(res)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("error writing response to stream: %s", err)
		}
	}
	streamingClient.RemoveConnDetails(connDetails)
	_ = writer.Close()
}
