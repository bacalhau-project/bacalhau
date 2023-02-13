package bprotocol

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rs/zerolog/log"
)

type ComputeHandlerParams struct {
	Host            host.Host
	ComputeEndpoint compute.Endpoint
}

// ComputeHandler is a handler for compute requests that registers for incoming libp2p requests to Bacalhau compute
// protocol, and delegates the requests to the compute endpoint.
type ComputeHandler struct {
	host            host.Host
	computeEndpoint compute.Endpoint
}

func NewComputeHandler(params ComputeHandlerParams) *ComputeHandler {
	handler := &ComputeHandler{
		host:            params.Host,
		computeEndpoint: params.ComputeEndpoint,
	}

	handler.host.SetStreamHandler(AskForBidProtocolID, handler.onAskForBid)
	handler.host.SetStreamHandler(BidAcceptedProtocolID, handler.onBidAccepted)
	handler.host.SetStreamHandler(BidRejectedProtocolID, handler.onBidRejected)
	handler.host.SetStreamHandler(ResultAcceptedProtocolID, handler.onResultAccepted)
	handler.host.SetStreamHandler(ResultRejectedProtocolID, handler.onResultRejected)
	handler.host.SetStreamHandler(CancelProtocolID, handler.onCancelJob)
	log.Debug().Msgf("ComputeHandler started on host %s", handler.host.ID().String())
	return handler
}

func (h *ComputeHandler) onAskForBid(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[compute.AskForBidRequest, compute.AskForBidResponse](ctx, stream, h.computeEndpoint.AskForBid)
}

func (h *ComputeHandler) onBidAccepted(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[compute.BidAcceptedRequest, compute.BidAcceptedResponse](ctx, stream, h.computeEndpoint.BidAccepted)
}

func (h *ComputeHandler) onBidRejected(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[compute.BidRejectedRequest, compute.BidRejectedResponse](ctx, stream, h.computeEndpoint.BidRejected)
}

func (h *ComputeHandler) onResultAccepted(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[compute.ResultAcceptedRequest, compute.ResultAcceptedResponse](ctx, stream, h.computeEndpoint.ResultAccepted)
}

func (h *ComputeHandler) onResultRejected(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[compute.ResultRejectedRequest, compute.ResultRejectedResponse](ctx, stream, h.computeEndpoint.ResultRejected)
}

func (h *ComputeHandler) onCancelJob(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[compute.CancelExecutionRequest, compute.CancelExecutionResponse](ctx, stream, h.computeEndpoint.CancelExecution)
}

//nolint:errcheck
func handleStream[Request any, Response any](
	ctx context.Context,
	stream network.Stream,
	f func(ctx context.Context, r Request) (Response, error)) {
	if err := stream.Scope().SetService(ComputeServiceName); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("error attaching stream to compute service")
		stream.Reset()
		return
	}

	request := new(Request)
	err := json.NewDecoder(stream).Decode(request)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error decoding %s: %s", reflect.TypeOf(request), err)
		stream.Reset()
		return
	}
	defer stream.Close() //nolint:errcheck

	response, err := f(ctx, *request)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error delegating %s: %s", reflect.TypeOf(request), err)
		stream.Reset()
		return
	}

	err = json.NewEncoder(stream).Encode(response)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error encoding %s: %s", reflect.TypeOf(response), err)
		stream.Reset()
		return
	}
}
