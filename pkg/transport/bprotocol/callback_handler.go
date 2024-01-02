package bprotocol

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
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

type callbackHandler[Request any] func(context.Context, Request)

func NewCallbackHandler(params CallbackHandlerParams) *CallbackHandler {
	handler := &CallbackHandler{
		host:     params.Host,
		callback: params.Callback,
	}

	host := handler.host
	host.SetStreamHandler(OnBidComplete, handleCallback(host, handler.callback.OnBidComplete))
	host.SetStreamHandler(OnRunComplete, handleCallback(host, handler.callback.OnRunComplete))
	host.SetStreamHandler(OnCancelComplete, handleCallback(host, handler.callback.OnCancelComplete))
	host.SetStreamHandler(OnComputeFailure, handleCallback(host, handler.callback.OnComputeFailure))
	return handler
}

func handleCallback[Request any](host host.Host, f callbackHandler[Request]) func(network.Stream) {
	return func(stream network.Stream) {
		ctx := logger.ContextWithNodeIDLogger(context.Background(), host.ID().String())
		handleCallbackStream(ctx, stream, f)
	}
}

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
	defer closer.CloseWithLogOnError("stream", stream)

	// TODO: validate which context to use here, and whether running in a goroutine is ok
	newCtx := logger.ContextWithNodeIDLogger(context.Background(), stream.Conn().LocalPeer().String())
	go f(newCtx, *request)
}
