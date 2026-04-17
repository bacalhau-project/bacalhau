package proxy

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

type CallbackProxyParams struct {
	Conn *nats.Conn
}

// CallbackProxy is a proxy for a compute.Callback that can be used to send compute callbacks to the requester node,
// such as when the execution is completed or when a failure occurs.
// The proxy can forward callbacks to a remote requester node, or locally if the node is the requester and a
// LocalCallback is provided.
type CallbackProxy struct {
	conn *nats.Conn
}

func NewCallbackProxy(params CallbackProxyParams) *CallbackProxy {
	proxy := &CallbackProxy{
		conn: params.Conn,
	}
	return proxy
}

func (p *CallbackProxy) OnBidComplete(ctx context.Context, result legacy.BidResult) {
	proxyCallbackRequest(ctx, p.conn, result.TargetPeerID, OnBidComplete, result)
}

func (p *CallbackProxy) OnRunComplete(ctx context.Context, result legacy.RunResult) {
	proxyCallbackRequest(ctx, p.conn, result.TargetPeerID, OnRunComplete, result)
}

func (p *CallbackProxy) OnComputeFailure(ctx context.Context, result legacy.ComputeError) {
	proxyCallbackRequest(ctx, p.conn, result.TargetPeerID, OnComputeFailure, result)
}

func proxyCallbackRequest(
	ctx context.Context,
	conn *nats.Conn,
	destNodeID string,
	method string,
	request interface{}) {
	// deserialize the request object
	data, err := json.Marshal(request)
	if err != nil {
		log.Ctx(ctx).Error().Err(errors.WithStack(err)).Msgf("%s: failed to marshal request", reflect.TypeOf(request))
		return
	}

	subject := callbackPublishSubject(destNodeID, method)
	log.Ctx(ctx).Trace().Msgf("Sending request %+v to subject %s", request, subject)

	// We use Publish instead of Request as Orchestrator callbacks do not return a response, for now.
	err = conn.Publish(subject, data)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("%s: failed to send callback to node %s", reflect.TypeOf(request), destNodeID)
		return
	}
}

// Compile-time interface check:
var _ compute.Callback = (*CallbackProxy)(nil)
