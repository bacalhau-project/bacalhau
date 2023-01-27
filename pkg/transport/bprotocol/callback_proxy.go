package bprotocol

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/rs/zerolog/log"
)

type CallbackProxyParams struct {
	Host          host.Host
	LocalCallback compute.Callback
}

// CallbackProxy is a proxy for a compute.Callback that can be used to send compute callbacks to the requester node,
// such as when the execution is completed or when a failure occurs.
// The proxy can forward callbacks to a remote requester node, or locally if the node is the requester and a
// LocalCallback is provided.
type CallbackProxy struct {
	host          host.Host
	localCallback compute.Callback
}

func NewCallbackProxy(params CallbackProxyParams) *CallbackProxy {
	proxy := &CallbackProxy{
		host:          params.Host,
		localCallback: params.LocalCallback,
	}
	return proxy
}

func (p *CallbackProxy) RegisterLocalComputeCallback(callback compute.Callback) {
	p.localCallback = callback
}

func (p *CallbackProxy) OnRunComplete(ctx context.Context, result compute.RunResult) {
	proxyCallbackRequest(ctx, p, result.RoutingMetadata, OnRunComplete, result, func(ctx2 context.Context) {
		p.localCallback.OnRunComplete(ctx2, result)
	})
}

func (p *CallbackProxy) OnPublishComplete(ctx context.Context, result compute.PublishResult) {
	proxyCallbackRequest(ctx, p, result.RoutingMetadata, OnPublishComplete, result, func(ctx2 context.Context) {
		p.localCallback.OnPublishComplete(ctx2, result)
	})
}

func (p *CallbackProxy) OnCancelComplete(ctx context.Context, result compute.CancelResult) {
	proxyCallbackRequest(ctx, p, result.RoutingMetadata, OnCancelComplete, result, func(ctx2 context.Context) {
		p.localCallback.OnCancelComplete(ctx2, result)
	})
}

func (p *CallbackProxy) OnComputeFailure(ctx context.Context, result compute.ComputeError) {
	proxyCallbackRequest(ctx, p, result.RoutingMetadata, OnComputeFailure, result, func(ctx2 context.Context) {
		p.localCallback.OnComputeFailure(ctx2, result)
	})
}

func proxyCallbackRequest(
	ctx context.Context,
	p *CallbackProxy,
	resultInfo compute.RoutingMetadata,
	protocolID protocol.ID,
	request interface{},
	selfDialFunc func(ctx2 context.Context)) {
	if resultInfo.TargetPeerID == p.host.ID().String() {
		if p.localCallback == nil {
			log.Ctx(ctx).Error().Msgf("unable to dial to self, unless a local compute callback is provided")
		} else {
			// TODO: validate which context to user here, and whether running in a goroutine is ok
			ctx2 := logger.ContextWithNodeIDLogger(context.Background(), p.host.ID().String())
			go selfDialFunc(ctx2)
		}
	} else {
		// decode the destination peer ID string value
		targetPeerID := resultInfo.TargetPeerID
		peerID, err := peer.Decode(targetPeerID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("%s: failed to decode peer ID %s", reflect.TypeOf(request), targetPeerID)
			return
		}

		// deserialize the request object
		data, err := json.Marshal(request)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("%s: failed to marshal request", reflect.TypeOf(request))
			return
		}

		// opening a stream to the destination peer
		stream, err := p.host.NewStream(ctx, peerID, protocolID)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("%s: failed to open stream to peer %s", reflect.TypeOf(request), targetPeerID)
			return
		}
		defer stream.Close() //nolint:errcheck
		if scopingErr := stream.Scope().SetService(ComputeServiceName); scopingErr != nil {
			log.Ctx(ctx).Error().Err(scopingErr).Msg("error attaching stream to requester service")
			stream.Reset() //nolint:errcheck
			return
		}

		// write the request to the stream
		_, err = stream.Write(data)
		if err != nil {
			stream.Reset() //nolint:errcheck
			log.Ctx(ctx).Error().Err(err).Msgf("%s: failed to write request to peer %s", reflect.TypeOf(request), targetPeerID)
			return
		}
	}
}

// Compile-time interface check:
var _ compute.Callback = (*CallbackProxy)(nil)
