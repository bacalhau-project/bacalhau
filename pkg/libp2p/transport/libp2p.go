package transport

import (
	"context"
	"errors"
	"fmt"
	"time"

	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/multiformats/go-multiaddr"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	libp2p_host "github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	core_transport "github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
)

const NodeInfoTopic = "bacalhau-node-info"

type Libp2pTransportConfig struct {
	Host           host.Host
	Peers          []string
	ReconnectDelay time.Duration
	CleanupManager *system.CleanupManager
}

func (c *Libp2pTransportConfig) Validate() error {
	return errors.Join(
		validate.IsNotNil(c.Host, "libp2p host cannot be nil"),
		validate.IsNotNil(c.CleanupManager, "cleanupManager cannot be nil"),
	)
}

type Libp2pTransport struct {
	Host              host.Host
	computeProxy      *bprotocol.ComputeProxy
	callbackProxy     *bprotocol.CallbackProxy
	nodeInfoPubSub    pubsub.PubSub[models.NodeState]
	nodeInfoDecorator models.NodeInfoDecorator
}

func NewLibp2pTransport(ctx context.Context,
	config Libp2pTransportConfig,
	nodeInfoStore routing.NodeInfoStore) (*Libp2pTransport, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("error validating libp2p transport config. %w", err)
	}

	// Monkey patch the identify protocol to allow discovering advertised addresses of networks of 3 or more nodes, instead of 5.
	// Setting the value to 2 means two other nodes must see the same addr for a node to discover its observed addr, which enables a network
	// of at least 3 nodes.
	identify.ActivationThresh = 2

	libp2pHost := config.Host

	// A single gossipSub instance that will be used by all topics
	gossipSub, err := newLibp2pPubSub(ctx, libp2pHost)
	if err != nil {
		return nil, err
	}

	// PubSub to publish node info to the network
	nodeInfoPubSub, err := libp2p.NewPubSub[models.NodeState](libp2p.PubSubParams{
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

	// compute proxy
	computeProxy := bprotocol.NewComputeProxy(bprotocol.ComputeProxyParams{
		Host: libp2pHost,
	})

	// Callback to send compute events (i.e. requester endpoint)
	computeCallback := bprotocol.NewCallbackProxy(bprotocol.CallbackProxyParams{
		Host: libp2pHost,
	})

	var libp2pPeer []multiaddr.Multiaddr
	for _, addr := range config.Peers {
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		libp2pPeer = append(libp2pPeer, maddr)
	}

	err = libp2p_host.ConnectToPeersContinuouslyWithRetryDuration(
		ctx, config.CleanupManager, libp2pHost, libp2pPeer, config.ReconnectDelay)
	if err != nil {
		return nil, err
	}

	return &Libp2pTransport{
		Host:              libp2pHost,
		computeProxy:      computeProxy,
		callbackProxy:     computeCallback,
		nodeInfoPubSub:    nodeInfoPubSub,
		nodeInfoDecorator: peerInfoDecorator,
	}, nil
}

func (t *Libp2pTransport) RegisterNodeInfoConsumer(ctx context.Context, nodeInfoStore routing.NodeInfoStore) error {
	// register consumers of node info published over gossipSub
	nodeInfoSubscriber := pubsub.NewChainedSubscriber[models.NodeState](true)
	nodeInfoSubscriber.Add(pubsub.SubscriberFunc[models.NodeState](nodeInfoStore.Add))

	return t.nodeInfoPubSub.Subscribe(ctx, nodeInfoSubscriber)
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

// RegisterManagementEndpoint is not implemented for libp2p transport. Compute
// nodes using this transport can call this with no effect.
func (t *Libp2pTransport) RegisterManagementEndpoint(endpoint compute.ManagementEndpoint) error {
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

// RegistrationProxy is not supported for the Libp2p transport and returns nil.
func (t *Libp2pTransport) ManagementProxy() compute.ManagementEndpoint {
	return nil
}

// NodeInfoPubSub returns the node info pubsub.
func (t *Libp2pTransport) NodeInfoPubSub() pubsub.PubSub[models.NodeState] {
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
	return errors.Join(
		t.nodeInfoPubSub.Close(ctx),
		t.Host.Close(),
	)
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
