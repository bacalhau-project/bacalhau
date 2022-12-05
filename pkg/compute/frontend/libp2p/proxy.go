package libp2p

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/compute/frontend"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type ProxyParams struct {
	Host host.Host
}

type Proxy struct {
	host host.Host
}

func NewProxy(params ProxyParams) *Proxy {
	proxy := &Proxy{
		host: params.Host,
	}
	return proxy
}

func (p Proxy) AskForBid(ctx context.Context, request frontend.AskForBidRequest) (frontend.AskForBidResponse, error) {
	return proxyRequest[frontend.AskForBidRequest, frontend.AskForBidResponse](
		ctx, p.host, request.DestPeerID, AskForBidProtocolID, request)
}

func (p Proxy) BidAccepted(ctx context.Context, request frontend.BidAcceptedRequest) (frontend.BidAcceptedResult, error) {
	return proxyRequest[frontend.BidAcceptedRequest, frontend.BidAcceptedResult](
		ctx, p.host, request.DestPeerID, BidAcceptedProtocolID, request)
}

func (p Proxy) BidRejected(ctx context.Context, request frontend.BidRejectedRequest) (frontend.BidRejectedResult, error) {
	return proxyRequest[frontend.BidRejectedRequest, frontend.BidRejectedResult](
		ctx, p.host, request.DestPeerID, BidRejectedProtocolID, request)
}

func (p Proxy) ResultAccepted(ctx context.Context, request frontend.ResultAcceptedRequest) (frontend.ResultAcceptedResult, error) {
	return proxyRequest[frontend.ResultAcceptedRequest, frontend.ResultAcceptedResult](
		ctx, p.host, request.DestPeerID, ResultAcceptedProtocolID, request)
}

func (p Proxy) ResultRejected(ctx context.Context, request frontend.ResultRejectedRequest) (frontend.ResultRejectedResult, error) {
	return proxyRequest[frontend.ResultRejectedRequest, frontend.ResultRejectedResult](
		ctx, p.host, request.DestPeerID, ResultRejectedProtocolID, request)
}

func (p Proxy) CancelJob(ctx context.Context, request frontend.CancelJobRequest) (frontend.CancelJobResult, error) {
	return proxyRequest[frontend.CancelJobRequest, frontend.CancelJobResult](
		ctx, p.host, request.DestPeerID, CancelProtocolID, request)
}

func proxyRequest[Request any, Response any](
	ctx context.Context,
	h host.Host,
	destPeerID string,
	protocolID protocol.ID,
	request Request) (Response, error) {
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

	// write the request to the stream
	_, err = stream.Write(data)
	if err != nil {
		return *response, fmt.Errorf("%s: failed to write request to peer %s: %w", reflect.TypeOf(request), destPeerID, err)
	}

	// Now we read the response that was sent from the dest peer
	err = json.NewDecoder(stream).Decode(response)
	if err != nil {
		return *response, fmt.Errorf("%s: failed to decode response from peer %s: %w", reflect.TypeOf(request), destPeerID, err)
	}

	return *response, nil
}

// Compile-time interface check:
var _ frontend.Service = (*Proxy)(nil)
