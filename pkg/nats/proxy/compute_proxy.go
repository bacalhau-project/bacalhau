package proxy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

type ComputeProxyParams struct {
	Conn *nats.Conn
}

// ComputeProxy is a proxy to a compute node endpoint that will forward requests to remote compute nodes, or
// to a local compute node if the target peer ID is the same as the local host, and a LocalEndpoint implementation
// is provided.
type ComputeProxy struct {
	conn *nats.Conn
}

func NewComputeProxy(params ComputeProxyParams) (*ComputeProxy, error) {
	proxy := &ComputeProxy{
		conn: params.Conn,
	}
	return proxy, nil
}

func (p *ComputeProxy) AskForBid(ctx context.Context, request legacy.AskForBidRequest) (legacy.AskForBidResponse, error) {
	return proxyRequest[legacy.AskForBidRequest, legacy.AskForBidResponse](
		ctx, p.conn, &BaseRequest[legacy.AskForBidRequest]{
			TargetNodeID: request.TargetPeerID,
			Method:       AskForBid,
			Body:         request,
		})
}

func (p *ComputeProxy) BidAccepted(ctx context.Context, request legacy.BidAcceptedRequest) (legacy.BidAcceptedResponse, error) {
	return proxyRequest[legacy.BidAcceptedRequest, legacy.BidAcceptedResponse](
		ctx, p.conn, &BaseRequest[legacy.BidAcceptedRequest]{
			TargetNodeID: request.TargetPeerID,
			Method:       BidAccepted,
			Body:         request,
		})
}

func (p *ComputeProxy) BidRejected(ctx context.Context, request legacy.BidRejectedRequest) (legacy.BidRejectedResponse, error) {
	return proxyRequest[legacy.BidRejectedRequest, legacy.BidRejectedResponse](
		ctx, p.conn, &BaseRequest[legacy.BidRejectedRequest]{
			TargetNodeID: request.TargetPeerID,
			Method:       BidRejected,
			Body:         request,
		})
}

func (p *ComputeProxy) CancelExecution(
	ctx context.Context, request legacy.CancelExecutionRequest) (legacy.CancelExecutionResponse, error) {
	return proxyRequest[legacy.CancelExecutionRequest, legacy.CancelExecutionResponse](
		ctx, p.conn, &BaseRequest[legacy.CancelExecutionRequest]{
			TargetNodeID: request.TargetPeerID,
			Method:       CancelExecution,
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

// Compile-time interface check:
var _ compute.Endpoint = (*ComputeProxy)(nil)
