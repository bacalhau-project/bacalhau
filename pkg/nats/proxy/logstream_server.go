package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/nats/stream"
)

// LogStreamHandlerParams defines parameters for creating a new LogStreamHandler.
type LogStreamHandlerParams struct {
	Name                       string
	Conn                       *nats.Conn
	LogstreamServer            logstream.Server
	StreamProducerClientConfig stream.StreamProducerClientConfig
}

// LogStreamHandler handles NATS messages for compute operations.
type LogStreamHandler struct {
	name            string
	conn            *nats.Conn
	logstreamServer logstream.Server
	subscription    *nats.Subscription
	streamingClient *stream.ProducerClient
}

// handlerWithResponse represents a function that processes a request and returns a response.
type handlerWithResponse[Request, Response any] func(context.Context, Request) (Response, error)

// NewLogStreamHandler creates a new LogStreamHandler.
func NewLogStreamHandler(ctx context.Context, params LogStreamHandlerParams) (*LogStreamHandler, error) {
	streamingClient, err := stream.NewProducerClient(ctx, stream.ProducerClientParams{
		Conn: params.Conn,
		Config: stream.StreamProducerClientConfig{
			HeartBeatIntervalDuration:        stream.DefaultHeartBeatIntervalDuration,
			HeartBeatRequestTimeout:          stream.DefaultHeartBeatRequestTimeout,
			StreamCancellationBufferDuration: stream.DefaultStreamCancellationBufferDuration,
		},
	})
	if err != nil {
		return nil, err
	}
	handler := &LogStreamHandler{
		name:            params.Name,
		conn:            params.Conn,
		logstreamServer: params.LogstreamServer,
		streamingClient: streamingClient,
	}

	subject := computeEndpointSubscribeSubject(handler.name)
	subscription, err := handler.conn.Subscribe(subject, func(m *nats.Msg) {
		handler.handleRequest(m)
	})
	if err != nil {
		return nil, err
	}
	handler.subscription = subscription
	log.Debug().Msgf("NATS log stream subscribed to %s", subject)
	return handler, nil
}

// handleRequest handles incoming NATS messages.
func (handler *LogStreamHandler) handleRequest(msg *nats.Msg) {
	ctx := context.Background()

	subjectParts := strings.Split(msg.Subject, ".")
	method := subjectParts[len(subjectParts)-1]

	switch method {
	case ExecutionLogs:
		processAndStream(ctx, handler.streamingClient, msg, handler.logstreamServer.GetLogStream)
	default:
		// Noop, not subscribed to this method
		return
	}
}

func processAndStream[Request, Response any](ctx context.Context, streamingClient *stream.ProducerClient, msg *nats.Msg,
	f handlerWithResponse[Request, <-chan *concurrency.AsyncResult[Response]],
) {
	if msg.Reply == "" {
		log.Ctx(ctx).Error().Msgf("streaming request on %s has no reply subject", msg.Subject)
		return
	}

	writer := streamingClient.NewWriter(msg.Reply)
	streamRequest := new(stream.Request)
	err := json.Unmarshal(msg.Data, streamRequest)
	if err != nil {
		_ = writer.CloseWithCode(stream.CloseBadRequest,
			fmt.Sprintf("error decoding %s: %s", reflect.TypeOf(streamRequest).Name(), err))
		return
	}

	request := new(Request)
	err = json.Unmarshal(streamRequest.Data, request)
	if err != nil {
		_ = writer.CloseWithCode(stream.CloseBadRequest,
			fmt.Sprintf("error decoding %s: %s", reflect.TypeOf(request).Name(), err))
		return
	}

	// This context is passed down to particular engine serving the stream. The cancel function is stored as part of
	// the StreamInfo. When the consumer client informs the producer client via heartBeat that it is no longer interested
	// in the stream, we call this cancel function. This also informs the engine that we are no longer interested in this
	// stream and hence close it. There might be scenarios where few logs will make it through after context cancelation
	// due to race conditions. This should be find and won't result in nil pointers or writing to a closed writer as
	// we only close the writer after source channel is closed.
	childCtx, cancel := context.WithCancel(ctx)
	err = streamingClient.AddStream(
		streamRequest.ConsumerID,
		streamRequest.StreamID,
		msg.Subject,
		streamRequest.HeartBeatRequestSub,
		cancel,
	)

	defer streamingClient.RemoveStream(streamRequest.ConsumerID, streamRequest.StreamID) //nolint:errcheck
	if err != nil {
		_ = writer.CloseWithCode(stream.CloseInternalServerErr,
			fmt.Sprintf("error in handler %s: %s", reflect.TypeOf(request).Name(), err))
		return
	}

	ch, err := f(childCtx, *request)
	if err != nil {
		closeError := writer.CloseWithCode(stream.CloseInternalServerErr,
			fmt.Sprintf("error in handler %s: %s", reflect.TypeOf(request).Name(), err))
		if closeError != nil {
			log.Err(closeError).Msg("error while closing NATS Stream")
		}
		return
	}

	for res := range ch {
		_, err := writer.WriteObject(res)
		if err != nil {
			log.Err(err).Msg("error writing log to NATS subject")
			break
		}
	}
	_ = writer.Close()
}
