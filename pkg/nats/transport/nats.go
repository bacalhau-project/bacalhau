package transport

import (
	"context"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_helper "github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/nats/proxy"
	nats_pubsub "github.com/bacalhau-project/bacalhau/pkg/nats/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	core_transport "github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

const NodeInfoSubjectPrefix = "node.info."

type NATSTransportConfig struct {
	NodeID            string
	Port              int
	AdvertisedAddress string
	Orchestrators     []string
	IsRequesterNode   bool
}

type NATSTransport struct {
	nodeID            string
	natsServer        *server.Server
	natsClient        *nats.Conn
	computeProxy      compute.Endpoint
	callbackProxy     compute.Callback
	nodeInfoPubSub    pubsub.PubSub[models.NodeInfo]
	nodeInfoDecorator models.NodeInfoDecorator
}

func NewNATSTransport(ctx context.Context,
	config NATSTransportConfig,
	nodeInfoStore routing.NodeInfoStore) (*NATSTransport, error) {
	log.Debug().Msgf("Creating NATS transport with config: %+v", config)
	var ns *server.Server
	var err error
	if config.IsRequesterNode {
		// create nats server with servers acting as its cluster peers
		serverOps := &server.Options{
			ServerName:      config.NodeID,
			Port:            config.Port,
			ClientAdvertise: config.AdvertisedAddress,
			RoutesStr:       strings.Join(config.Orchestrators, ","),
		}
		ns, err = nats_helper.NewServer(ctx, serverOps)
		if err != nil {
			return nil, err
		}

		config.Orchestrators = append(config.Orchestrators, ns.ClientURL())
	}

	// create nats client
	log.Debug().Msgf("Creating NATS client with servers: %s", strings.Join(config.Orchestrators, ","))
	nc, err := nats_helper.NewClient(ctx, config.NodeID, strings.Join(config.Orchestrators, ","))
	if err != nil {
		return nil, err
	}

	// PubSub to publish and consume node info messages
	nodeInfoPubSub, err := nats_pubsub.NewPubSub[models.NodeInfo](nats_pubsub.PubSubParams{
		Conn:                nc,
		Subject:             NodeInfoSubjectPrefix + config.NodeID,
		SubscriptionSubject: NodeInfoSubjectPrefix + "*",
	})
	if err != nil {
		return nil, err
	}

	if config.IsRequesterNode {
		// subscribe to nodeInfo subject and add nodeInfo to nodeInfoStore
		nodeInfoSubscriber := pubsub.NewChainedSubscriber[models.NodeInfo](true)
		nodeInfoSubscriber.Add(pubsub.SubscriberFunc[models.NodeInfo](nodeInfoStore.Add))
		err = nodeInfoPubSub.Subscribe(ctx, nodeInfoSubscriber)
		if err != nil {
			return nil, err
		}
	}

	// compute proxy
	computeProxy := proxy.NewComputeProxy(proxy.ComputeProxyParams{
		Conn: nc,
	})

	// Callback to send compute events (i.e. requester endpoint)
	computeCallback := proxy.NewCallbackProxy(proxy.CallbackProxyParams{
		Conn: nc,
	})

	return &NATSTransport{
		nodeID:            config.NodeID,
		natsServer:        ns,
		natsClient:        nc,
		computeProxy:      computeProxy,
		callbackProxy:     computeCallback,
		nodeInfoPubSub:    nodeInfoPubSub,
		nodeInfoDecorator: models.NoopNodeInfoDecorator{},
	}, nil
}

// RegisterComputeCallback registers a compute callback with the transport layer.
func (t *NATSTransport) RegisterComputeCallback(callback compute.Callback) error {
	_, err := proxy.NewCallbackHandler(proxy.CallbackHandlerParams{
		Name:     t.nodeID,
		Conn:     t.natsClient,
		Callback: callback,
	})
	return err
}

// RegisterComputeEndpoint registers a compute endpoint with the transport layer.
func (t *NATSTransport) RegisterComputeEndpoint(endpoint compute.Endpoint) error {
	_, err := proxy.NewComputeHandler(proxy.ComputeHandlerParams{
		Name:            t.nodeID,
		Conn:            t.natsClient,
		ComputeEndpoint: endpoint,
	})
	return err
}

// ComputeProxy returns the compute proxy.
func (t *NATSTransport) ComputeProxy() compute.Endpoint {
	return t.computeProxy
}

// CallbackProxy returns the callback proxy.
func (t *NATSTransport) CallbackProxy() compute.Callback {
	return t.callbackProxy
}

// NodeInfoPubSub returns the node info pubsub.
func (t *NATSTransport) NodeInfoPubSub() pubsub.PubSub[models.NodeInfo] {
	return t.nodeInfoPubSub
}

// NodeInfoDecorator returns the node info decorator.
func (t *NATSTransport) NodeInfoDecorator() models.NodeInfoDecorator {
	return t.nodeInfoDecorator
}

// Close closes the transport layer.
func (t *NATSTransport) Close(ctx context.Context) error {
	if t.natsServer != nil {
		t.natsServer.Shutdown()
	}
	if t.natsClient != nil {
		t.natsClient.Close()
	}
	return nil
}

// compile-time interface check
var _ core_transport.TransportLayer = (*NATSTransport)(nil)
