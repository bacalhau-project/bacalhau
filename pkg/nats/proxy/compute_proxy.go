package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/nats/stream"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

const (
	// asyncRequestChanLen is the channel length for buffering asynchronous results.
	asyncRequestChanLen = 8
)

type ComputeProxyParams struct {
	Conn *nats.Conn
}

// ComputeProxy is a proxy to a compute node endpoint that will forward requests to remote compute nodes, or
// to a local compute node if the target peer ID is the same as the local host, and a LocalEndpoint implementation
// is provided.
type ComputeProxy struct {
	conn            *nats.Conn
	streamingClient *stream.ConsumerClient
}

func NewComputeProxy(params ComputeProxyParams) (*ComputeProxy, error) {
	sc, err := stream.NewConsumerClient(stream.ConsumerClientParams{
		Conn: params.Conn,
		Config: stream.StreamConsumerClientConfig{
			StreamCancellationBufferDuration: 5 * time.Second, //nolinter:gomnd
		},
	})
	if err != nil {
		return nil, err
	}
	proxy := &ComputeProxy{
		conn:            params.Conn,
		streamingClient: sc,
	}
	return proxy, nil
}

func (p *ComputeProxy) AskForBid(ctx context.Context, request compute.AskForBidRequest) (compute.AskForBidResponse, error) {
	return proxyRequest[compute.AskForBidRequest, compute.AskForBidResponse](
		ctx, p.conn, &BaseRequest[compute.AskForBidRequest]{
			TargetNodeID: request.TargetPeerID,
			Method:       AskForBid,
			Body:         request,
		})
}

func (p *ComputeProxy) BidAccepted(ctx context.Context, request compute.BidAcceptedRequest) (compute.BidAcceptedResponse, error) {
	return proxyRequest[compute.BidAcceptedRequest, compute.BidAcceptedResponse](
		ctx, p.conn, &BaseRequest[compute.BidAcceptedRequest]{
			TargetNodeID: request.TargetPeerID,
			Method:       BidAccepted,
			Body:         request,
		})
}

func (p *ComputeProxy) BidRejected(ctx context.Context, request compute.BidRejectedRequest) (compute.BidRejectedResponse, error) {
	return proxyRequest[compute.BidRejectedRequest, compute.BidRejectedResponse](
		ctx, p.conn, &BaseRequest[compute.BidRejectedRequest]{
			TargetNodeID: request.TargetPeerID,
			Method:       BidRejected,
			Body:         request,
		})
}

func (p *ComputeProxy) CancelExecution(
	ctx context.Context, request compute.CancelExecutionRequest) (compute.CancelExecutionResponse, error) {
	return proxyRequest[compute.CancelExecutionRequest, compute.CancelExecutionResponse](
		ctx, p.conn, &BaseRequest[compute.CancelExecutionRequest]{
			TargetNodeID: request.TargetPeerID,
			Method:       CancelExecution,
			Body:         request,
		})
}

func (p *ComputeProxy) ExecutionLogs(ctx context.Context, request compute.ExecutionLogsRequest) (
	<-chan *concurrency.AsyncResult[models.ExecutionLog], error) {
	return proxyStreamingRequest[compute.ExecutionLogsRequest, models.ExecutionLog](
		ctx, p.streamingClient, &BaseRequest[compute.ExecutionLogsRequest]{
			TargetNodeID: request.TargetPeerID,
			Method:       ExecutionLogs,
			Body:         request,
		})
}

func proxyRequest[Request any, Response any](
	ctx context.Context,
	conn *nats.Conn,
	request *BaseRequest[Request]) (Response, error) {
	// response object
	response := new(Response)

	subject := request.ComputeEndpoint()
	log.Ctx(ctx).Trace().Msgf("Sending request %+v to subject %s", request, subject)

	// serialize the request object
	data, err := json.Marshal(request.Body)
	if err != nil {
		return *response, fmt.Errorf("%T: failed to marshal request: %w", request.Body, err)
	}

	res, err := conn.RequestWithContext(ctx, subject, data)
	if err != nil {
		return *response, fmt.Errorf("%T: failed to send request to node %s: %w", request.Body, request.TargetNodeID, err)
	}

	// The handler will have wrapped the response in a Result[T] along with
	// any error that occurred, so we will decode it and pass the
	// inner response/error on to the caller.
	result := new(concurrency.AsyncResult[Response])
	err = json.Unmarshal(res.Data, result)
	if err != nil {
		return *response, fmt.Errorf("%T: failed to decode response from peer %s: %w", request.Body, request.TargetNodeID, err)
	}

	return result.ValueOrError()
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
	res, err := client.OpenStream(ctx, subject, request.TargetNodeID, data)
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
var _ compute.Endpoint = (*ComputeProxy)(nil)
