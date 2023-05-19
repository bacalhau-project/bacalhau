package simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/rs/zerolog/log"
)

type ComputeProxyParams struct {
	SimulatorNodeID string
	Host            host.Host
	LocalEndpoint   compute.Endpoint // optional in case this host is also a compute node and to allow local calls

}

// ComputeProxy is a proxy to a compute node endpoint that will forward requests to remote compute nodes, or
// to a local compute node if the target peer ID is the same as the local host, and a LocalEndpoint implementation
// is provided.
type ComputeProxy struct {
	simulatorNodeID string
	host            host.Host
	localEndpoint   compute.Endpoint
}

func NewComputeProxy(params ComputeProxyParams) *ComputeProxy {
	proxy := &ComputeProxy{
		simulatorNodeID: params.SimulatorNodeID,
		host:            params.Host,
		localEndpoint:   params.LocalEndpoint,
	}
	log.Info().Msgf("ComputeProxy created with simulator node %s", params.SimulatorNodeID)
	return proxy
}

func (p *ComputeProxy) RegisterLocalComputeEndpoint(endpoint compute.Endpoint) {
	p.localEndpoint = endpoint
}

func (p *ComputeProxy) AskForBid(ctx context.Context, request compute.AskForBidRequest) (compute.AskForBidResponse, error) {
	if p.simulatorNodeID == p.host.ID().String() {
		if p.localEndpoint == nil {
			return compute.AskForBidResponse{}, fmt.Errorf("unable to dial to self, unless a local compute endpoint is provided")
		}
		return p.localEndpoint.AskForBid(ctx, request)
	}
	return proxyRequest[compute.AskForBidRequest, compute.AskForBidResponse](
		ctx, p.host, p.simulatorNodeID, bprotocol.AskForBidProtocolID, request)
}

func (p *ComputeProxy) BidAccepted(ctx context.Context, request compute.BidAcceptedRequest) (compute.BidAcceptedResponse, error) {
	if p.simulatorNodeID == p.host.ID().String() {
		if p.localEndpoint == nil {
			return compute.BidAcceptedResponse{}, fmt.Errorf("unable to dial to self, unless a local compute endpoint is provided")
		}
		return p.localEndpoint.BidAccepted(ctx, request)
	}
	return proxyRequest[compute.BidAcceptedRequest, compute.BidAcceptedResponse](
		ctx, p.host, p.simulatorNodeID, bprotocol.BidAcceptedProtocolID, request)
}

func (p *ComputeProxy) BidRejected(ctx context.Context, request compute.BidRejectedRequest) (compute.BidRejectedResponse, error) {
	if p.simulatorNodeID == p.host.ID().String() {
		if p.localEndpoint == nil {
			return compute.BidRejectedResponse{}, fmt.Errorf("unable to dial to self, unless a local compute endpoint is provided")
		}
		return p.localEndpoint.BidRejected(ctx, request)
	}
	return proxyRequest[compute.BidRejectedRequest, compute.BidRejectedResponse](
		ctx, p.host, p.simulatorNodeID, bprotocol.BidRejectedProtocolID, request)
}

func (p *ComputeProxy) ResultAccepted(ctx context.Context, request compute.ResultAcceptedRequest) (compute.ResultAcceptedResponse, error) {
	if p.simulatorNodeID == p.host.ID().String() {
		if p.localEndpoint == nil {
			return compute.ResultAcceptedResponse{}, fmt.Errorf("unable to dial to self, unless a local compute endpoint is provided")
		}
		return p.localEndpoint.ResultAccepted(ctx, request)
	}
	return proxyRequest[compute.ResultAcceptedRequest, compute.ResultAcceptedResponse](
		ctx, p.host, p.simulatorNodeID, bprotocol.ResultAcceptedProtocolID, request)
}

func (p *ComputeProxy) ResultRejected(ctx context.Context, request compute.ResultRejectedRequest) (compute.ResultRejectedResponse, error) {
	if p.simulatorNodeID == p.host.ID().String() {
		if p.localEndpoint == nil {
			return compute.ResultRejectedResponse{}, fmt.Errorf("unable to dial to self, unless a local compute endpoint is provided")
		}
		return p.localEndpoint.ResultRejected(ctx, request)
	}
	return proxyRequest[compute.ResultRejectedRequest, compute.ResultRejectedResponse](
		ctx, p.host, p.simulatorNodeID, bprotocol.ResultRejectedProtocolID, request)
}

func (p *ComputeProxy) CancelExecution(
	ctx context.Context, request compute.CancelExecutionRequest) (compute.CancelExecutionResponse, error) {
	if p.simulatorNodeID == p.host.ID().String() {
		if p.localEndpoint == nil {
			return compute.CancelExecutionResponse{}, fmt.Errorf("unable to dial to self, unless a local compute endpoint is provided")
		}
		return p.localEndpoint.CancelExecution(ctx, request)
	}
	return proxyRequest[compute.CancelExecutionRequest, compute.CancelExecutionResponse](
		ctx, p.host, p.simulatorNodeID, bprotocol.CancelProtocolID, request)
}

func (p *ComputeProxy) ExecutionLogs(
	ctx context.Context, request compute.ExecutionLogsRequest) (compute.ExecutionLogsResponse, error) {
	if p.simulatorNodeID == p.host.ID().String() {
		if p.localEndpoint == nil {
			return compute.ExecutionLogsResponse{}, fmt.Errorf("unable to dial to self, unless a local compute endpoint is provided")
		}
		return p.localEndpoint.ExecutionLogs(ctx, request)
	}
	return proxyRequest[compute.ExecutionLogsRequest, compute.ExecutionLogsResponse](
		ctx, p.host, p.simulatorNodeID, bprotocol.CancelProtocolID, request)
}

func proxyRequest[Request any, Response any](
	ctx context.Context,
	h host.Host,
	destPeerID string,
	protocolID protocol.ID,
	request Request) (Response, error) {
	log.Ctx(ctx).Info().Msgf("Forwarding request %+v to %s", request, destPeerID)
	// response object
	response := new(Response)

	// decode the destination peer ID string value
	peerID, err := peer.Decode(destPeerID)
	if err != nil {
		return *response, fmt.Errorf("%s: failed to decode peer ID %s: %w", reflect.TypeOf(request), destPeerID, err)
	}

	// deserialize the request object
	data, err := json.Marshal(request)
	if err != nil {
		return *response, fmt.Errorf("%s: failed to marshal request: %w", reflect.TypeOf(request), err)
	}

	// opening a stream to the destination peer
	stream, err := h.NewStream(ctx, peerID, protocolID)
	if err != nil {
		return *response, fmt.Errorf("%s: failed to open stream to peer %s: %w", reflect.TypeOf(request), destPeerID, err)
	}
	defer stream.Close() //nolint:errcheck

	// write the request to the stream
	_, err = stream.Write(data)
	if err != nil {
		_ = stream.Reset()
		return *response, fmt.Errorf("%s: failed to write request to peer %s: %w", reflect.TypeOf(request), destPeerID, err)
	}

	// The handler will have wrapped the response in a Result[T] along with
	// any error that occurred, so we will decode it and pass the
	// inner response/error on to the caller.
	result := &bprotocol.Result[Response]{}
	err = json.NewDecoder(stream).Decode(result)
	if err != nil {
		_ = stream.Reset()
		return *response, fmt.Errorf("%s: failed to decode response from peer %s: %w", reflect.TypeOf(request), destPeerID, err)
	}

	return result.Rehydrate()
}

// Compile-time interface check:
var _ compute.Endpoint = (*ComputeProxy)(nil)
