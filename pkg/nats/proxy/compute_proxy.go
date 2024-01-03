package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
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

func NewComputeProxy(params ComputeProxyParams) *ComputeProxy {
	proxy := &ComputeProxy{
		conn: params.Conn,
	}
	return proxy
}

func (p *ComputeProxy) AskForBid(ctx context.Context, request compute.AskForBidRequest) (compute.AskForBidResponse, error) {
	return proxyRequest[compute.AskForBidRequest, compute.AskForBidResponse](
		ctx, p.conn, request.TargetPeerID, AskForBid, request)
}

func (p *ComputeProxy) BidAccepted(ctx context.Context, request compute.BidAcceptedRequest) (compute.BidAcceptedResponse, error) {
	return proxyRequest[compute.BidAcceptedRequest, compute.BidAcceptedResponse](
		ctx, p.conn, request.TargetPeerID, BidAccepted, request)
}

func (p *ComputeProxy) BidRejected(ctx context.Context, request compute.BidRejectedRequest) (compute.BidRejectedResponse, error) {
	return proxyRequest[compute.BidRejectedRequest, compute.BidRejectedResponse](
		ctx, p.conn, request.TargetPeerID, BidRejected, request)
}

func (p *ComputeProxy) CancelExecution(
	ctx context.Context, request compute.CancelExecutionRequest) (compute.CancelExecutionResponse, error) {
	return proxyRequest[compute.CancelExecutionRequest, compute.CancelExecutionResponse](
		ctx, p.conn, request.TargetPeerID, CancelExecution, request)
}

func (p *ComputeProxy) ExecutionLogs(
	ctx context.Context, request compute.ExecutionLogsRequest) (compute.ExecutionLogsResponse, error) {
	return proxyRequest[compute.ExecutionLogsRequest, compute.ExecutionLogsResponse](
		ctx, p.conn, request.TargetPeerID, ExecutionLogs, request)
}

func proxyRequest[Request any, Response any](
	ctx context.Context,
	conn *nats.Conn,
	destNodeID string,
	method string,
	request Request) (Response, error) {
	// response object
	response := new(Response)

	// deserialize the request object
	data, err := json.Marshal(request)
	if err != nil {
		return *response, fmt.Errorf("%s: failed to marshal request: %w", reflect.TypeOf(request), err)
	}

	subject := computeEndpointPublishSubject(destNodeID, method)
	log.Ctx(ctx).Trace().Msgf("Sending request %+v to subject %s", request, subject)
	res, err := conn.RequestWithContext(ctx, subject, data)
	if err != nil {
		return *response, fmt.Errorf("%s: failed to send request to node %s: %w", reflect.TypeOf(request), destNodeID, err)
	}

	// The handler will have wrapped the response in a Result[T] along with
	// any error that occurred, so we will decode it and pass the
	// inner response/error on to the caller.
	result := &Result[Response]{}
	err = json.Unmarshal(res.Data, result)
	if err != nil {
		return *response, fmt.Errorf("%s: failed to decode response from peer %s: %w", reflect.TypeOf(request), destNodeID, err)
	}

	return result.Rehydrate()
}

// Compile-time interface check:
var _ compute.Endpoint = (*ComputeProxy)(nil)
