package libp2p

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/compute/frontend"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rs/zerolog/log"
)

type HandlerParams struct {
	Host     host.Host
	Frontend frontend.Service
}

type Handler struct {
	host     host.Host
	frontend frontend.Service
}

func NewHandler(params HandlerParams) *Handler {
	handler := &Handler{
		host:     params.Host,
		frontend: params.Frontend,
	}

	handler.host.SetStreamHandler(AskForBidProtocolID, handler.onAskForBid)
	handler.host.SetStreamHandler(BidAcceptedProtocolID, handler.onBidAccepted)
	handler.host.SetStreamHandler(BidRejectedProtocolID, handler.onBidRejected)
	handler.host.SetStreamHandler(ResultAcceptedProtocolID, handler.onResultAccepted)
	handler.host.SetStreamHandler(ResultRejectedProtocolID, handler.onResultRejected)
	handler.host.SetStreamHandler(CancelProtocolID, handler.onCancelJob)
	return handler
}

func (h *Handler) onAskForBid(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[frontend.AskForBidRequest, frontend.AskForBidResponse](ctx, stream, h.frontend.AskForBid)
}

func (h *Handler) onBidAccepted(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[frontend.BidAcceptedRequest, frontend.BidAcceptedResult](ctx, stream, h.frontend.BidAccepted)
}

func (h *Handler) onBidRejected(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[frontend.BidRejectedRequest, frontend.BidRejectedResult](ctx, stream, h.frontend.BidRejected)
}

func (h *Handler) onResultAccepted(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[frontend.ResultAcceptedRequest, frontend.ResultAcceptedResult](ctx, stream, h.frontend.ResultAccepted)
}

func (h *Handler) onResultRejected(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[frontend.ResultRejectedRequest, frontend.ResultRejectedResult](ctx, stream, h.frontend.ResultRejected)
}

func (h *Handler) onCancelJob(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleStream[frontend.CancelJobRequest, frontend.CancelJobResult](ctx, stream, h.frontend.CancelJob)
}

//nolint:errcheck
func handleStream[Request any, Response any](
	ctx context.Context,
	stream network.Stream,
	f func(ctx context.Context, r Request) (Response, error)) {
	if err := stream.Scope().SetService(ServiceName); err != nil {
		log.Ctx(ctx).Debug().Msgf("error attaching stream to compute service: %s", err)
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
	stream.Close()
}
