package transport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_helper "github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/nats/proxy"
	nats_pubsub "github.com/bacalhau-project/bacalhau/pkg/nats/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	core_transport "github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/hashicorp/go-multierror"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

const NodeInfoSubjectPrefix = "node.info."

// reservedChars are the characters that are not allowed in node IDs as nodes
// subscribe to subjects with their node IDs, and these are wildcards
// in NATS subjects that could cause a node to subscribe to unintended subjects.
const reservedChars = ".*>"

type NATSTransportConfig struct {
	NodeID            string
	Port              int
	AdvertisedAddress string
	Orchestrators     []string
	IsRequesterNode   bool

	// StoreDir is the directory where the NATS server will store its data
	StoreDir string

	// AuthSecret is a secret string that clients must use to connect. NATS servers
	// must supply this config, while clients can also supply it as the user part
	// of their Orchestrator URL.
	AuthSecret string

	// Cluster config for requester nodes to connect with each other
	ClusterName              string
	ClusterPort              int
	ClusterAdvertisedAddress string
	ClusterPeers             []string
}

func (c *NATSTransportConfig) Validate() error {
	var mErr *multierror.Error
	if validate.IsBlank(c.NodeID) {
		mErr = multierror.Append(mErr, errors.New("missing node ID"))
	} else if validate.ContainsSpaces(c.NodeID) {
		mErr = multierror.Append(mErr, errors.New("node ID contains a space"))
	} else if validate.ContainsNull(c.NodeID) {
		mErr = multierror.Append(mErr, errors.New("node ID contains a null character"))
	} else if strings.ContainsAny(c.NodeID, reservedChars) {
		mErr = multierror.Append(mErr, fmt.Errorf("node ID '%s' contains one or more reserved characters: %s", c.NodeID, reservedChars))
	}

	if c.IsRequesterNode {
		mErr = multierror.Append(mErr, validate.IsGreaterThanZero(c.Port, "port %d must be greater than zero", c.Port))

		// if cluster config is set, validate it
		if c.ClusterName != "" || c.ClusterPort != 0 || c.ClusterAdvertisedAddress != "" || len(c.ClusterPeers) > 0 {
			mErr = multierror.Append(mErr,
				validate.IsGreaterThanZero(c.ClusterPort, "cluster port %d must be greater than zero", c.ClusterPort))
		}
	} else {
		if validate.IsEmpty(c.Orchestrators) {
			mErr = multierror.Append(mErr, errors.New("missing orchestrators"))
		}
	}
	return mErr.ErrorOrNil()
}

type NATSTransport struct {
	Config            NATSTransportConfig
	nodeID            string
	natsServer        *nats_helper.ServerManager
	natsClient        *nats_helper.ClientManager
	computeProxy      compute.Endpoint
	callbackProxy     compute.Callback
	nodeInfoPubSub    pubsub.PubSub[models.NodeInfo]
	nodeInfoDecorator models.NodeInfoDecorator
	managementProxy   compute.ManagementEndpoint
}

//nolint:funlen
func NewNATSTransport(ctx context.Context,
	config NATSTransportConfig) (*NATSTransport, error) {
	log.Debug().Msgf("Creating NATS transport with config: %+v", config)
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("error validating nats transport config. %w", err)
	}

	var sm *nats_helper.ServerManager
	if config.IsRequesterNode {
		var err error

		// create nats server with servers acting as its cluster peers
		serverOpts := &server.Options{
			ServerName:             config.NodeID,
			Port:                   config.Port,
			ClientAdvertise:        config.AdvertisedAddress,
			Authorization:          config.AuthSecret,
			Debug:                  true, // will only be used if log level is debug
			JetStream:              true,
			DisableJetStreamBanner: true,
			StoreDir:               config.StoreDir,
		}

		// Only set cluster options if cluster peers are provided. Jetstream doesn't
		// like the setting to be present with no values, or with values that are
		// a local address (e.g. it can't RAFT to itself).
		routes, err := nats_helper.RoutesFromSlice(config.ClusterPeers, false)
		if err != nil {
			return nil, err
		}

		if len(config.ClusterPeers) > 0 {
			serverOpts.Routes = routes

			serverOpts.Cluster = server.ClusterOpts{
				Name:      config.ClusterName,
				Port:      config.ClusterPort,
				Advertise: config.ClusterAdvertisedAddress,
			}
		}

		log.Debug().Msgf("Creating NATS server with options: %+v", serverOpts)
		sm, err = nats_helper.NewServerManager(ctx, nats_helper.ServerManagerParams{
			Options: serverOpts,
		})
		if err != nil {
			return nil, err
		}

		config.Orchestrators = append(config.Orchestrators, sm.Server.ClientURL())
	}

	nc, err := CreateClient(ctx, config)
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

	// compute proxy
	computeProxy, err := proxy.NewComputeProxy(proxy.ComputeProxyParams{
		Conn: nc.Client,
	})
	if err != nil {
		return nil, err
	}

	// Callback to send compute events (i.e. requester endpoint)
	computeCallback := proxy.NewCallbackProxy(proxy.CallbackProxyParams{
		Conn: nc.Client,
	})

	// A proxy to register and unregister compute nodes with the requester
	managementProxy := proxy.NewManagementProxy(proxy.ManagementProxyParams{
		Conn: nc.Client,
	})

	return &NATSTransport{
		nodeID:            config.NodeID,
		natsServer:        sm,
		natsClient:        nc,
		Config:            config,
		computeProxy:      computeProxy,
		callbackProxy:     computeCallback,
		nodeInfoPubSub:    nodeInfoPubSub,
		nodeInfoDecorator: models.NoopNodeInfoDecorator{},
		managementProxy:   managementProxy,
	}, nil
}

func CreateClient(ctx context.Context, config NATSTransportConfig) (*nats_helper.ClientManager, error) {
	// create nats client
	log.Debug().Msgf("Creating NATS client with servers: %s", strings.Join(config.Orchestrators, ","))
	clientOptions := []nats.Option{
		nats.Name(config.NodeID),
	}
	if config.AuthSecret != "" {
		clientOptions = append(clientOptions, nats.Token(config.AuthSecret))
	}
	return nats_helper.NewClientManager(ctx,
		strings.Join(config.Orchestrators, ","),
		clientOptions...,
	)
}

func (t *NATSTransport) RegisterNodeInfoConsumer(ctx context.Context, infostore routing.NodeInfoStore) error {
	// subscribe to nodeInfo subject and add nodeInfo to nodeInfoStore
	nodeInfoSubscriber := pubsub.NewChainedSubscriber[models.NodeInfo](true)
	nodeInfoSubscriber.Add(pubsub.SubscriberFunc[models.NodeInfo](infostore.Add))
	return t.nodeInfoPubSub.Subscribe(ctx, nodeInfoSubscriber)
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

// RegisterManagementEndpoint registers a requester endpoint with the transport layer
func (t *NATSTransport) RegisterManagementEndpoint(endpoint compute.ManagementEndpoint) error {
	_, err := proxy.NewManagementHandler(proxy.ManagementHandlerParams{
		Conn:               t.natsClient.Client,
		ManagementEndpoint: endpoint,
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

// RegistrationProxy returns the previoously created registration proxy.
func (t *NATSTransport) ManagementProxy() compute.ManagementEndpoint {
	return t.managementProxy
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
