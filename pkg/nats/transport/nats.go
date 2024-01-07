package transport

import (
	"context"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_helper "github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/nats/proxy"
	nats_pubsub "github.com/bacalhau-project/bacalhau/pkg/nats/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	core_transport "github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/rs/zerolog/log"
)

const NodeInfoSubjectPrefix = "node.info."

type NATSTransportConfig struct {
	NodeID            string
	Port              int
	AdvertisedAddress string
	Orchestrators     []string
	IsRequesterNode   bool

	// Cluster config for requester nodes to connect with each other
	ClusterName              string
	ClusterPort              int
	ClusterAdvertisedAddress string
	ClusterPeers             []string
}

type NATSTransport struct {
	nodeID            string
	natsServer        *nats_helper.ServerManager
	natsClient        *nats_helper.ClientManager
	computeProxy      compute.Endpoint
	callbackProxy     compute.Callback
	nodeInfoPubSub    pubsub.PubSub[models.NodeInfo]
	nodeInfoDecorator models.NodeInfoDecorator
}

func NewNATSTransport(ctx context.Context,
	config NATSTransportConfig,
	nodeInfoStore routing.NodeInfoStore) (*NATSTransport, error) {
	log.Debug().Msgf("Creating NATS transport with config: %+v", config)
	var sm *nats_helper.ServerManager
	var err error
	if config.IsRequesterNode {
		// create nats server with servers acting as its cluster peers
		routes, err := nats_helper.RoutesFromSlice(config.ClusterPeers)
		if err != nil {
			return nil, err
		}
		serverOps := &server.Options{
			ServerName:      config.NodeID,
			Port:            config.Port,
			ClientAdvertise: config.AdvertisedAddress,
			Routes:          routes,
			Debug:           true, // will only be used if log level is debug
			Cluster: server.ClusterOpts{
				Name:      config.ClusterName,
				Port:      config.ClusterPort,
				Advertise: config.ClusterAdvertisedAddress,
			},
		}
		log.Debug().Msgf("Creating NATS server with options: %+v", serverOps)
		sm, err = nats_helper.NewServerManager(ctx, serverOps)
		if err != nil {
			return nil, err
		}

		config.Orchestrators = append(config.Orchestrators, sm.Server.ClientURL())
	}

	// create nats client
	log.Debug().Msgf("Creating NATS client with servers: %s", strings.Join(config.Orchestrators, ","))
	nc, err := nats_helper.NewClientManager(ctx, nats_helper.ClientManagerParams{
		Name:    config.NodeID,
		Servers: strings.Join(config.Orchestrators, ","),
	})
	if err != nil {
		return nil, err
	}

	// PubSub to publish and consume node info messages
	nodeInfoPubSub, err := nats_pubsub.NewPubSub[models.NodeInfo](nats_pubsub.PubSubParams{
		Conn:                nc.Client,
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
		Conn: nc.Client,
	})

	// Callback to send compute events (i.e. requester endpoint)
	computeCallback := proxy.NewCallbackProxy(proxy.CallbackProxyParams{
		Conn: nc.Client,
	})

	return &NATSTransport{
		nodeID:            config.NodeID,
		natsServer:        sm,
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
		Conn:     t.natsClient.Client,
		Callback: callback,
	})
	return err
}

// RegisterComputeEndpoint registers a compute endpoint with the transport layer.
func (t *NATSTransport) RegisterComputeEndpoint(endpoint compute.Endpoint) error {
	_, err := proxy.NewComputeHandler(proxy.ComputeHandlerParams{
		Name:            t.nodeID,
		Conn:            t.natsClient.Client,
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

// DebugInfoProviders returns the debug info of the NATS transport layer
func (t *NATSTransport) DebugInfoProviders() []model.DebugInfoProvider {
	var debugInfoProviders []model.DebugInfoProvider
	if t.natsServer != nil {
		debugInfoProviders = append(debugInfoProviders, t.natsServer)
	}
	if t.natsClient != nil {
		debugInfoProviders = append(debugInfoProviders, t.natsClient)
	}
	return debugInfoProviders
}

// Close closes the transport layer.
func (t *NATSTransport) Close(ctx context.Context) error {
	if t.natsServer != nil {
		log.Ctx(ctx).Debug().Msgf("Shutting down server %s", t.natsServer.Server.Name())
		t.natsServer.Stop()
	}
	if t.natsClient != nil {
		t.natsClient.Stop()
	}
	return nil
}

// compile-time interface check
var _ core_transport.TransportLayer = (*NATSTransport)(nil)
