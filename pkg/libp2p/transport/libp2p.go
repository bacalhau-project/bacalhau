package transport

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	libp2p_host "github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	core_transport "github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
	"github.com/hashicorp/go-multierror"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

const NodeInfoTopic = "bacalhau-node-info"

type Libp2pTransportConfig struct {
	Host host.Host
}

type Libp2pTransport struct {
	Host              host.Host
	computeProxy      *bprotocol.ComputeProxy
	callbackProxy     *bprotocol.CallbackProxy
	nodeInfoPubSub    pubsub.PubSub[models.NodeInfo]
	nodeInfoDecorator models.NodeInfoDecorator
}

func NewLibp2pTransport(ctx context.Context,
	config Libp2pTransportConfig,
	nodeInfoStore routing.NodeInfoStore) (*Libp2pTransport, error) {
	// Monkey patch the identify protocol to allow discovering advertised addresses of networks of 3 or more nodes, instead of 5.
	identify.ActivationThresh = 2

	libp2pHost := config.Host

	// A single gossipSub instance that will be used by all topics
	gossipSub, err := newLibp2pPubSub(ctx, libp2pHost)
	if err != nil {
		return nil, err
	}

	// PubSub to publish node info to the network
	nodeInfoPubSub, err := libp2p.NewPubSub[models.NodeInfo](libp2p.PubSubParams{
		Host:      libp2pHost,
		TopicName: NodeInfoTopic,
		PubSub:    gossipSub,
	})
	if err != nil {
		return nil, err
	}

	// node info provider
	basicHost, ok := libp2pHost.(*basichost.BasicHost)
	if !ok {
		return nil, fmt.Errorf("host is not a basic host")
	}

	peerInfoDecorator := libp2p_host.NewPeerInfoDecorator(libp2p_host.PeerInfoDecoratorParams{
		Host:            basicHost,
		IdentityService: basicHost.IDService(),
	})

	libp2pHost = routedhost.Wrap(libp2pHost, nodeInfoStore)

	// register consumers of node info published over gossipSub
	nodeInfoSubscriber := pubsub.NewChainedSubscriber[models.NodeInfo](true)
	nodeInfoSubscriber.Add(pubsub.SubscriberFunc[models.NodeInfo](nodeInfoStore.Add))
	err = nodeInfoPubSub.Subscribe(ctx, nodeInfoSubscriber)
	if err != nil {
		return nil, err
	}

	// compute proxy
	computeProxy := bprotocol.NewComputeProxy(bprotocol.ComputeProxyParams{
		Host: libp2pHost,
	})

	// Callback to send compute events (i.e. requester endpoint)
	computeCallback := bprotocol.NewCallbackProxy(bprotocol.CallbackProxyParams{
		Host: libp2pHost,
	})

	return &Libp2pTransport{
		Host:              libp2pHost,
		computeProxy:      computeProxy,
		callbackProxy:     computeCallback,
		nodeInfoPubSub:    nodeInfoPubSub,
		nodeInfoDecorator: peerInfoDecorator,
	}, nil
}

// RegisterComputeCallback registers a compute callback with the transport layer.
func (t *Libp2pTransport) RegisterComputeCallback(callback compute.Callback) error {
	bprotocol.NewCallbackHandler(bprotocol.CallbackHandlerParams{
		Host:     t.Host,
		Callback: callback,
	})
	// To enable nodes self-dialing themselves as libp2p doesn't support it.
	t.callbackProxy.RegisterLocalComputeCallback(callback)

	return nil
}

// RegisterComputeEndpoint registers a compute endpoint with the transport layer.
func (t *Libp2pTransport) RegisterComputeEndpoint(endpoint compute.Endpoint) error {
	bprotocol.NewComputeHandler(bprotocol.ComputeHandlerParams{
		Host:            t.Host,
		ComputeEndpoint: endpoint,
	})
	// To enable nodes self-dialing themselves as libp2p doesn't support it.
	t.computeProxy.RegisterLocalComputeEndpoint(endpoint)

	return nil
}

// ComputeProxy returns the compute proxy.
func (t *Libp2pTransport) ComputeProxy() compute.Endpoint {
	return t.computeProxy
}

// CallbackProxy returns the callback proxy.
func (t *Libp2pTransport) CallbackProxy() compute.Callback {
	return t.callbackProxy
}

// NodeInfoPubSub returns the node info pubsub.
func (t *Libp2pTransport) NodeInfoPubSub() pubsub.PubSub[models.NodeInfo] {
	return t.nodeInfoPubSub
}

// NodeInfoDecorator returns the node info decorator.
func (t *Libp2pTransport) NodeInfoDecorator() models.NodeInfoDecorator {
	return t.nodeInfoDecorator
}

// DebugInfoProviders returns the debug info.
func (t *Libp2pTransport) DebugInfoProviders() []model.DebugInfoProvider {
	return []model.DebugInfoProvider{}
}

// Close closes the transport layer.
func (t *Libp2pTransport) Close(ctx context.Context) error {
	var errors *multierror.Error
	errors = multierror.Append(errors, t.nodeInfoPubSub.Close(ctx))
	errors = multierror.Append(errors, t.Host.Close())
	return errors.ErrorOrNil()
}

func newLibp2pPubSub(ctx context.Context, host host.Host) (*libp2p_pubsub.PubSub, error) {
	tracer, err := libp2p_pubsub.NewJSONTracer(pkgconfig.GetLibp2pTracerPath())
	if err != nil {
		return nil, err
	}

	pgParams := libp2p_pubsub.NewPeerGaterParams(
		0.33, //nolint:gomnd
		libp2p_pubsub.ScoreParameterDecay(2*time.Minute),  //nolint:gomnd
		libp2p_pubsub.ScoreParameterDecay(10*time.Minute), //nolint:gomnd
	)

	return libp2p_pubsub.NewGossipSub(
		ctx,
		host,
		libp2p_pubsub.WithPeerExchange(true),
		libp2p_pubsub.WithPeerGater(pgParams),
		libp2p_pubsub.WithEventTracer(tracer),
	)
}

// compile-time interface check
var _ core_transport.TransportLayer = (*Libp2pTransport)(nil)
