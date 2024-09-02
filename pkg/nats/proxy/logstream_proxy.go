package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
	"github.com/bacalhau-project/bacalhau/pkg/nats/stream"
)

const (
	// asyncRequestChanLen is the channel length for buffering asynchronous results.
	asyncRequestChanLen = 8
)

type LogStreamProxyParams struct {
	Conn *nats.Conn
}

// LogStreamProxy is a proxy to a compute node endpoint that will forward requests to remote compute nodes, or
// to a local compute node if the target peer ID is the same as the local host, and a LocalEndpoint implementation
// is provided.
type LogStreamProxy struct {
	conn            *nats.Conn
	streamingClient *stream.ConsumerClient
}

func NewLogStreamProxy(params LogStreamProxyParams) (*LogStreamProxy, error) {
	sc, err := stream.NewConsumerClient(stream.ConsumerClientParams{
		Conn: params.Conn,
		Config: stream.StreamConsumerClientConfig{
			StreamCancellationBufferDuration: 5 * time.Second, //nolinter:gomnd
		},
	})
	if err != nil {
		return nil, err
	}
	proxy := &LogStreamProxy{
		conn:            params.Conn,
		streamingClient: sc,
	}
	return proxy, nil
}

func (p *LogStreamProxy) GetLogStream(ctx context.Context, request requests.LogStreamRequest) (<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	return proxyStreamingRequest[requests.LogStreamRequest, models.ExecutionLog](
		ctx, p.streamingClient, &BaseRequest[requests.LogStreamRequest]{
			TargetNodeID: request.NodeID,
			Method:       ExecutionLogs,
			Body:         request,
		})
}

func proxyStreamingRequest[Request any, Response any](
	ctx context.Context,
	client *stream.ConsumerClient,
	request *BaseRequest[Request]) (
	<-chan *concurrency.AsyncResult[Response], error) {
	subject := request.ComputeEndpoint()
	log.Ctx(ctx).Trace().Msgf("Sending streaming request %+v to subject %s", request.Body, subject)

	// serialize the request object
	data, err := json.Marshal(request.Body)
	if err != nil {
		return nil, fmt.Errorf("%T: failed to marshal request: %w", request.Body, err)
	}
	res, err := client.OpenStream(ctx, subject, data)
	if err != nil {
		return nil, fmt.Errorf("%T: failed to send request to node %s: %w", request.Body, request.TargetNodeID, err)
	}

	return concurrency.AsyncChannelTransform[[]byte, Response](ctx, res, asyncRequestChanLen,
		func(r []byte) (Response, error) {
			response := new(concurrency.AsyncResult[Response])
			if err := json.Unmarshal(r, response); err != nil {
				return *new(Response), fmt.Errorf("%T: failed to decode response from node %s: %w", request.Body, request.TargetNodeID, err)
			}
			return response.ValueOrError()
		}), nil
}

// Compile-time interface check:
var _ logstream.Server = (*LogStreamProxy)(nil)
