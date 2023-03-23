package bprotocol

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rs/zerolog/log"
)

type CallbackHandlerParams struct {
	Host     host.Host
	Callback compute.Callback
}

// CallbackHandler is a handler for callback events that registers for incoming libp2p requests to Bacalhau callback
// protocol, and delegates the handling of the request to the provided callback.
type CallbackHandler struct {
	host     host.Host
	callback compute.Callback
}

func NewCallbackHandler(params CallbackHandlerParams) *CallbackHandler {
	handler := &CallbackHandler{
		host:     params.Host,
		callback: params.Callback,
	}

	handler.host.SetStreamHandler(OnRunComplete, handler.onRunSuccess)
	handler.host.SetStreamHandler(OnPublishComplete, handler.onPublishSuccess)
	handler.host.SetStreamHandler(OnCancelComplete, handler.onCancelSuccess)
	handler.host.SetStreamHandler(OnComputeFailure, handler.onComputeFailure)
	return handler
}

func (h *CallbackHandler) onRunSuccess(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleCallbackStream[compute.RunResult](ctx, stream, h.callback.OnRunComplete)
}
func (h *CallbackHandler) onPublishSuccess(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleCallbackStream[compute.PublishResult](ctx, stream, h.callback.OnPublishComplete)
}

func (h *CallbackHandler) onCancelSuccess(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleCallbackStream[compute.CancelResult](ctx, stream, h.callback.OnCancelComplete)
}

func (h *CallbackHandler) onComputeFailure(stream network.Stream) {
	ctx := logger.ContextWithNodeIDLogger(context.Background(), h.host.ID().String())
	handleCallbackStream[compute.ComputeError](ctx, stream, h.callback.OnComputeFailure)
}

//nolint:errcheck
func handleCallbackStream[Request any](
	ctx context.Context,
	stream network.Stream,
	f func(ctx context.Context, r Request)) {
	ctx = logger.ContextWithNodeIDLogger(ctx, stream.Conn().LocalPeer().String())
	if err := stream.Scope().SetService(CallbackServiceName); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("error attaching stream to requester service")
		_ = stream.Reset()
		return
	}

	request := new(Request)
	err := json.NewDecoder(stream).Decode(request)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error decoding %s: %s", reflect.TypeOf(request), err)
		_ = stream.Reset()
		return
	}
	defer stream.Close() //nolint:errcheck

	// TODO: validate which context to use here, and whether running in a goroutine is ok
	newCtx := logger.ContextWithNodeIDLogger(context.Background(), stream.Conn().LocalPeer().String())
	go f(newCtx, *request)
}
