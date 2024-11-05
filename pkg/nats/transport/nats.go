package transport

import (
	"context"
	"errors"
	"strings"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_helper "github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/nats/proxy"
	nats_pubsub "github.com/bacalhau-project/bacalhau/pkg/nats/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

const NodeInfoSubjectPrefix = "node.info."

// reservedChars are the characters that are not allowed in node IDs as nodes
// subscribe to subjects with their node IDs, and these are wildcards
// in NATS subjects that could cause a node to subscribe to unintended subjects.
const reservedChars = ".*>"

type NATSTransportConfig struct {
	NodeID            string
	Host              string
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
	mErr := errors.Join(
		validate.NotBlank(c.NodeID, "missing node ID"),
		validate.NoSpaces(c.NodeID, "node ID cannot contain spaces"),
		validate.NoNullChars(c.NodeID, "node ID cannot contain null characters"),
		validate.ContainsNoneOf(c.NodeID, reservedChars,
			"node ID cannot contain any of the following characters: %s", reservedChars),
	)

	if c.IsRequesterNode {
		mErr = errors.Join(mErr, validate.IsGreaterThanZero(c.Port, "port %d must be greater than zero", c.Port))

		// if cluster config is set, validate it
		if c.ClusterName != "" || c.ClusterPort != 0 || c.ClusterAdvertisedAddress != "" || len(c.ClusterPeers) > 0 {
			mErr = errors.Join(mErr,
				validate.IsGreaterThanZero(c.ClusterPort, "cluster port %d must be greater than zero", c.ClusterPort))
		}
	} else {
		mErr = errors.Join(mErr, validate.IsNotEmpty(c.Orchestrators, "missing orchestrators"))
	}
	if mErr != nil {
		return nats_helper.NewConfigurationError("invalid transport config:\n%s", mErr)
	}
	return nil
}

type NATSTransport struct {
	Config            *NATSTransportConfig
	nodeID            string
	natsServer        *nats_helper.ServerManager
	natsClient        *nats_helper.ClientManager
	computeProxy      compute.Endpoint
	logstreamProxy    logstream.Server
	callbackProxy     compute.Callback
	nodeInfoPubSub    pubsub.PubSub[models.NodeState]
	nodeInfoDecorator models.NodeInfoDecorator
	managementProxy   compute.ManagementEndpoint
}

//nolint:funlen
func NewNATSTransport(ctx context.Context,
	config *NATSTransportConfig) (*NATSTransport, error) {
	log.Debug().Msgf("Creating NATS transport with config: %+v", config)
	if err := config.Validate(); err != nil {
		return nil, bacerrors.Wrap(err, "invalid cluster config").WithCode(bacerrors.ValidationError)
	}

	var sm *nats_helper.ServerManager
	if config.IsRequesterNode {
		var err error

		// create nats server with servers acting as its cluster peers
		serverOpts := &server.Options{
			ServerName:             config.NodeID,
			Host:                   config.Host,
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
	nodeInfoPubSub, err := nats_pubsub.NewPubSub[models.NodeState](nats_pubsub.PubSubParams{
		Conn:                nc.Client,
		Subject:             NodeInfoSubjectPrefix + config.NodeID,
		SubscriptionSubject: NodeInfoSubjectPrefix + "*",
	})
	if err != nil {
		return nil, err
	}

	// logstream compute proxy
	logStreamProxy, err := proxy.NewLogStreamProxy(proxy.LogStreamProxyParams{
		Conn: nc.Client,
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
		logstreamProxy:    logStreamProxy,
		computeProxy:      computeProxy,
		callbackProxy:     computeCallback,
		nodeInfoPubSub:    nodeInfoPubSub,
		nodeInfoDecorator: models.NoopNodeInfoDecorator{},
		managementProxy:   managementProxy,
	}, nil
}

// CreateClient creates a new NATS client.
func (t *NATSTransport) CreateClient(ctx context.Context) (*nats.Conn, error) {
	clientManager, err := CreateClient(ctx, t.Config)
	if err != nil {
		return nil, err
	}
	return clientManager.Client, nil
}

// Client returns the existing NATS client.
func (t *NATSTransport) Client() *nats.Conn {
	return t.natsClient.Client
}

func CreateClient(ctx context.Context, config *NATSTransportConfig) (*nats_helper.ClientManager, error) {
	// create nats client
	log.Debug().Msgf("Creating NATS client with servers: %s", strings.Join(config.Orchestrators, ","))
	clientOptions := []nats.Option{
		nats.Name(config.NodeID),
		nats.MaxReconnects(-1),
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
	nodeInfoSubscriber := pubsub.NewChainedSubscriber[models.NodeState](true)
	nodeInfoSubscriber.Add(pubsub.SubscriberFunc[models.NodeState](infostore.Add))
	return t.nodeInfoPubSub.Subscribe(ctx, nodeInfoSubscriber)
}

// RegisterLogstreamServer registers a compute logstream server with the transport layer.
func (t *NATSTransport) RegisterLogstreamServer(ctx context.Context, logstreamServer logstream.Server) error {
	if logstreamServer == nil {
		return errors.New("logstreamServer cannot be nil")
	}
	_, err := proxy.NewLogStreamHandler(ctx, proxy.LogStreamHandlerParams{
		Name:            t.nodeID,
		Conn:            t.natsClient.Client,
		LogstreamServer: logstreamServer,
	})
	return err
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
func (t *NATSTransport) RegisterComputeEndpoint(ctx context.Context, endpoint compute.Endpoint) error {
	_, err := proxy.NewComputeHandler(ctx, proxy.ComputeHandlerParams{
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

// LogstreamServer returns the compute logstream server.
func (t *NATSTransport) LogstreamServer() logstream.Server {
	return t.logstreamProxy
}

// ComputeProxy returns the compute proxy.
func (t *NATSTransport) ComputeProxy() compute.Endpoint {
	return t.computeProxy
}

// CallbackProxy returns the callback proxy.
func (t *NATSTransport) CallbackProxy() compute.Callback {
	return t.callbackProxy
}

// ManagementProxy returns the previously created registration proxy.
func (t *NATSTransport) ManagementProxy() compute.ManagementEndpoint {
	return t.managementProxy
}

// NodeInfoPubSub returns the node info pubsub.
func (t *NATSTransport) NodeInfoPubSub() pubsub.PubSub[models.NodeState] {
	return t.nodeInfoPubSub
}

// NodeInfoDecorator returns the node info decorator.
func (t *NATSTransport) NodeInfoDecorator() models.NodeInfoDecorator {
	return t.nodeInfoDecorator
}

// DebugInfoProviders returns the debug info of the NATS transport layer
func (t *NATSTransport) DebugInfoProviders() []models.DebugInfoProvider {
	var debugInfoProviders []models.DebugInfoProvider
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
	if t.natsClient != nil {
		t.natsClient.Stop()
	}
	if t.natsServer != nil {
		log.Ctx(ctx).Debug().Msgf("Shutting down server %s", t.natsServer.Server.Name())
		t.natsServer.Stop()
	}
	return nil
}
