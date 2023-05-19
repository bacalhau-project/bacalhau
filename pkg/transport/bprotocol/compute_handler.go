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

type handlerWithResponse[Request, Response any] func(context.Context, Request) (Response, error)

func NewComputeHandler(params ComputeHandlerParams) *ComputeHandler {
	handler := &ComputeHandler{
		host:            params.Host,
		computeEndpoint: params.ComputeEndpoint,
	}

	host := handler.host
	host.SetStreamHandler(AskForBidProtocolID, handleWith(host, handler.computeEndpoint.AskForBid))
	host.SetStreamHandler(BidAcceptedProtocolID, handleWith(host, handler.computeEndpoint.BidAccepted))
	host.SetStreamHandler(BidRejectedProtocolID, handleWith(host, handler.computeEndpoint.BidRejected))
	host.SetStreamHandler(ResultAcceptedProtocolID, handleWith(host, handler.computeEndpoint.ResultAccepted))
	host.SetStreamHandler(ResultRejectedProtocolID, handleWith(host, handler.computeEndpoint.ResultRejected))
	host.SetStreamHandler(CancelProtocolID, handleWith(host, handler.computeEndpoint.CancelExecution))
	host.SetStreamHandler(ExecutionLogsID, handleWith(host, handler.computeEndpoint.ExecutionLogs))
	log.Debug().Msgf("ComputeHandler started on host %s", handler.host.ID().String())
	return handler
}

func handleWith[Request, Response any](host host.Host, f handlerWithResponse[Request, Response]) func(network.Stream) {
	return func(stream network.Stream) {
		ctx := logger.ContextWithNodeIDLogger(context.Background(), host.ID().String())
		handleStream(ctx, stream, f)
	}
}

func handleStream[Request, Response any](ctx context.Context, stream network.Stream, f handlerWithResponse[Request, Response]) {
	if err := stream.Scope().SetService(ComputeServiceName); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("error attaching stream to compute service")
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

	response, err := f(ctx, *request)

	// We will wrap up the response/error in a bprotocol Result type which
	// can be decoded by the proxy itself.
	result := Result[Response]{
		Response: response,
	}

	// We can log the error here, but we should not bail as we want the error to be sent
	// back to the caller.
	if err != nil {
		result.Error = err.Error()
		log.Ctx(ctx).Debug().Err(err).Msgf("error delegating %s", reflect.TypeOf(request))
	}

	err = json.NewEncoder(stream).Encode(result)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("error encoding %s: %s", reflect.TypeOf(response), err)
		_ = stream.Reset()
		return
	}
}
