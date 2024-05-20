package node

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/imdario/mergo"
	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p/core/host"
	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	libp2p_transport "github.com/bacalhau-project/bacalhau/pkg/libp2p/transport"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
	"github.com/bacalhau-project/bacalhau/pkg/node/manager"
	"github.com/bacalhau-project/bacalhau/pkg/node/metrics"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/agent"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/routing/kvstore"
	"github.com/bacalhau-project/bacalhau/pkg/routing/tracing"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

type FeatureConfig struct {
	Engines    []string
	Publishers []string
	Storages   []string
}

// Node configuration
type NodeConfig struct {
	NodeID                      string
	IPFSClient                  ipfs.Client
	CleanupManager              *system.CleanupManager
	HostAddress                 string
	APIPort                     uint16
	RequesterAutoCert           string
	RequesterAutoCertCache      string
	RequesterTLSCertificateFile string
	RequesterTLSKeyFile         string
	RequesterSelfSign           bool
	DisabledFeatures            FeatureConfig
	ComputeConfig               ComputeConfig
	RequesterNodeConfig         RequesterConfig
	APIServerConfig             publicapi.Config
	AuthConfig                  types.AuthConfig
	NodeType                    models.NodeType
	IsRequesterNode             bool
	IsComputeNode               bool
	Labels                      map[string]string
	NodeInfoPublisherInterval   routing.NodeInfoPublisherIntervalConfig
	DependencyInjector          NodeDependencyInjector
	AllowListedLocalPaths       []string
	NodeInfoStoreTTL            time.Duration

	NetworkConfig NetworkConfig
}

func (c *NodeConfig) Validate() error {
	// TODO: add more validations
	var mErr error
	mErr = errors.Join(mErr, c.NetworkConfig.Validate())
	return mErr
}

// Lazy node dependency injector that generate instances of different
// components on demand and based on the configuration provided.
type NodeDependencyInjector struct {
	StorageProvidersFactory StorageProvidersFactory
	ExecutorsFactory        ExecutorsFactory
	PublishersFactory       PublishersFactory
	AuthenticatorsFactory   AuthenticatorsFactory
}

func NewExecutorPluginNodeDependencyInjector() NodeDependencyInjector {
	return NodeDependencyInjector{
		StorageProvidersFactory: NewStandardStorageProvidersFactory(),
		ExecutorsFactory:        NewPluginExecutorFactory(),
		PublishersFactory:       NewStandardPublishersFactory(),
		AuthenticatorsFactory:   NewStandardAuthenticatorsFactory(),
	}
}

func NewStandardNodeDependencyInjector() NodeDependencyInjector {
	return NodeDependencyInjector{
		StorageProvidersFactory: NewStandardStorageProvidersFactory(),
		ExecutorsFactory:        NewStandardExecutorsFactory(),
		PublishersFactory:       NewStandardPublishersFactory(),
		AuthenticatorsFactory:   NewStandardAuthenticatorsFactory(),
	}
}

type Node struct {
	// Visible for testing
	ID             string
	APIServer      *publicapi.Server
	ComputeNode    *Compute
	RequesterNode  *Requester
	CleanupManager *system.CleanupManager
	IPFSClient     ipfs.Client
	Libp2pHost     host.Host // only set if using libp2p transport, nil otherwise
}

func (n *Node) Start(ctx context.Context) error {
	return n.APIServer.ListenAndServe(ctx)
}

//nolint:funlen,gocyclo // Should be simplified when moving to FX
func NewNode(
	ctx context.Context,
	config NodeConfig) (*Node, error) {
	var err error
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	config.DependencyInjector = mergeDependencyInjectors(config.DependencyInjector, NewStandardNodeDependencyInjector())
	err = mergo.Merge(&config.APIServerConfig, publicapi.DefaultConfig())
	if err != nil {
		return nil, err
	}
	// TODO: #830 Same as #829 in pkg/eventhandler/chained_handlers.go
	if system.GetEnvironment() == system.EnvironmentTest || system.GetEnvironment() == system.EnvironmentDev {
		config.APIServerConfig.LogLevel = "trace"
	}

	err = config.Validate()
	if err != nil {
		return nil, fmt.Errorf("error validating node config. %w", err)
	}

	storageProviders, err := config.DependencyInjector.StorageProvidersFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	authzPolicy, err := policy.FromPathOrDefault(config.AuthConfig.AccessPolicyPath, authz.AlwaysAllowPolicy)
	if err != nil {
		return nil, err
	}

	signingKey, err := pkgconfig.GetClientPublicKey()
	if err != nil {
		return nil, err
	}

	serverVersion := version.Get()
	// public http api server
	serverParams := publicapi.ServerParams{
		Router:     echo.New(),
		Address:    config.HostAddress,
		Port:       config.APIPort,
		HostID:     config.NodeID,
		Config:     config.APIServerConfig,
		Authorizer: authz.NewPolicyAuthorizer(authzPolicy, signingKey, config.NodeID),
		Headers: map[string]string{
			apimodels.HTTPHeaderBacalhauGitVersion: serverVersion.GitVersion,
			apimodels.HTTPHeaderBacalhauGitCommit:  serverVersion.GitCommit,
			apimodels.HTTPHeaderBacalhauBuildDate:  serverVersion.BuildDate.UTC().String(),
			apimodels.HTTPHeaderBacalhauBuildOS:    serverVersion.GOOS,
			apimodels.HTTPHeaderBacalhauArch:       serverVersion.GOARCH,
		},
	}

	// Only allow autocert for requester nodes
	if config.IsRequesterNode {
		serverParams.AutoCertDomain = config.RequesterAutoCert
		serverParams.AutoCertCache = config.RequesterAutoCertCache
		serverParams.TLSCertificateFile = config.RequesterTLSCertificateFile
		serverParams.TLSKeyFile = config.RequesterTLSKeyFile
	}

	apiServer, err := publicapi.NewAPIServer(serverParams)
	if err != nil {
		return nil, err
	}
	// node info store that is used for both discovering compute nodes, as to find addresses of other nodes for routing requests.

	var natsConfig *nats_transport.NATSTransportConfig
	var transportLayer transport.TransportLayer
	var tracingInfoStore routing.NodeInfoStore
	var heartbeatSvr *heartbeat.HeartbeatServer

	if config.NetworkConfig.Type == models.NetworkTypeNATS {
		natsConfig = &nats_transport.NATSTransportConfig{
			NodeID:                   config.NodeID,
			Port:                     config.NetworkConfig.Port,
			AdvertisedAddress:        config.NetworkConfig.AdvertisedAddress,
			AuthSecret:               config.NetworkConfig.AuthSecret,
			Orchestrators:            config.NetworkConfig.Orchestrators,
			StoreDir:                 config.NetworkConfig.StoreDir,
			ClusterName:              config.NetworkConfig.ClusterName,
			ClusterPort:              config.NetworkConfig.ClusterPort,
			ClusterPeers:             config.NetworkConfig.ClusterPeers,
			ClusterAdvertisedAddress: config.NetworkConfig.ClusterAdvertisedAddress,
			IsRequesterNode:          config.IsRequesterNode,
		}

		natsTransportLayer, err := nats_transport.NewNATSTransport(ctx, natsConfig)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to create NATS transport layer")
		}
		transportLayer = natsTransportLayer

		if config.IsRequesterNode {
			// KV Node Store requires connection info from the NATS server so that it is able
			// to create its own connection and then subscribe to the node info topic.
			natsClient, err := nats_transport.CreateClient(ctx, natsTransportLayer.Config)
			if err != nil {
				return nil, pkgerrors.Wrap(err, "failed to create NATS client for node info store")
			}
			nodeInfoStore, err := kvstore.NewNodeStore(ctx, kvstore.NodeStoreParams{
				BucketName: kvstore.DefaultBucketName,
				Client:     natsClient.Client,
			})
			if err != nil {
				return nil, pkgerrors.Wrap(err, "failed to create node info store using NATS transport connection info")
			}
			tracingInfoStore = tracing.NewNodeStore(nodeInfoStore)

			heartbeatParams := heartbeat.HeartbeatServerParams{
				Client:                natsClient.Client,
				Topic:                 config.RequesterNodeConfig.ControlPlaneSettings.HeartbeatTopic,
				CheckFrequency:        config.RequesterNodeConfig.ControlPlaneSettings.HeartbeatCheckFrequency.AsTimeDuration(),
				NodeDisconnectedAfter: config.RequesterNodeConfig.ControlPlaneSettings.NodeDisconnectedAfter.AsTimeDuration(),
			}
			heartbeatSvr, err = heartbeat.NewServer(heartbeatParams)
			if err != nil {
				return nil, pkgerrors.Wrap(err, "failed to create heartbeat server using NATS transport connection info")
			}

			// Once the KV store has been created, it can be offered to the transport layer to be used as a consumer
			// of node info.
			if err := transportLayer.RegisterNodeInfoConsumer(ctx, tracingInfoStore); err != nil {
				return nil, pkgerrors.Wrap(err, "failed to register node info consumer with nats transport")
			}
		}
	} else {
		tracingInfoStore = tracing.NewNodeStore(
			inmemory.NewNodeStore(inmemory.NodeStoreParams{
				TTL: config.NodeInfoStoreTTL,
			}))

		libp2pConfig := libp2p_transport.Libp2pTransportConfig{
			Host:           config.NetworkConfig.Libp2pHost,
			Peers:          config.NetworkConfig.ClusterPeers,
			ReconnectDelay: config.NetworkConfig.ReconnectDelay,
			CleanupManager: config.CleanupManager,
		}
		transportLayer, err = libp2p_transport.NewLibp2pTransport(ctx, libp2pConfig, tracingInfoStore)
		if err = transportLayer.RegisterNodeInfoConsumer(ctx, tracingInfoStore); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to register node info consumer with libp2p transport")
		}
	}
	if err != nil {
		return nil, err
	}

	var debugInfoProviders []model.DebugInfoProvider
	debugInfoProviders = append(debugInfoProviders, transportLayer.DebugInfoProviders()...)

	var requesterNode *Requester
	var computeNode *Compute
	var labelsProvider models.LabelsProvider

	// setup requester node
	if config.IsRequesterNode {
		authenticators, err := config.DependencyInjector.AuthenticatorsFactory.Get(ctx, config)
		if err != nil {
			return nil, err
		}

		metrics.NodeInfo.Add(ctx, 1,
			attribute.StringSlice("node_authenticators", authenticators.Keys(ctx)),
		)

		// Create a new node manager to keep track of compute nodes connecting
		// to the network. Provide it with a mechanism to lookup (and enhance)
		// node info, and a reference to the heartbeat server if running NATS.
		nodeManager := manager.NewNodeManager(manager.NodeManagerParams{
			NodeInfo:             tracingInfoStore,
			Heartbeats:           heartbeatSvr,
			DefaultApprovalState: config.RequesterNodeConfig.DefaultApprovalState,
		})

		// Start the nodemanager, ensuring it doesn't block the main thread and
		// that any errors are logged. If we are unable to start the manager
		// then we should not start the node.
		if err := nodeManager.Start(ctx); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to start node manager")
		}

		// NodeManager node wraps the node manager and implements the routing.NodeInfoStore
		// interface so that it can return nodes and add the most recent resource information
		// to the node info returned.  When the libp2p transport is no longer necessary, we
		// can remove the parameter from the NewRequesterNode call and use the nodeManager
		// instead.
		legacyInfoStore := tracingInfoStore
		if config.NetworkConfig.Type == models.NetworkTypeNATS {
			legacyInfoStore = nodeManager
		}

		requesterNode, err = NewRequesterNode(
			ctx,
			config.NodeID,
			apiServer,
			config.RequesterNodeConfig,
			storageProviders,
			authenticators,
			legacyInfoStore,
			transportLayer.ComputeProxy(),
			nodeManager,
		)
		if err != nil {
			return nil, err
		}

		err = transportLayer.RegisterComputeCallback(requesterNode.localCallback)
		if err != nil {
			return nil, err
		}

		// TODO: We only currently want a management endpoint for register/update
		// when using NATS
		if config.NetworkConfig.Type == models.NetworkTypeNATS {
			err = transportLayer.RegisterManagementEndpoint(nodeManager)
			if err != nil {
				return nil, err
			}
		}

		labelsProvider = models.MergeLabelsInOrder(
			&ConfigLabelsProvider{staticLabels: config.Labels},
			&RuntimeLabelsProvider{},
		)
		debugInfoProviders = append(debugInfoProviders, requesterNode.debugInfoProviders...)
	}

	if config.IsComputeNode {
		storagePath := pkgconfig.GetStoragePath()

		publishers, err := config.DependencyInjector.PublishersFactory.Get(ctx, config)
		if err != nil {
			return nil, err
		}

		executors, err := config.DependencyInjector.ExecutorsFactory.Get(ctx, config)
		if err != nil {
			return nil, err
		}

		metrics.NodeInfo.Add(ctx, 1,
			attribute.StringSlice("node_publishers", publishers.Keys(ctx)),
			attribute.StringSlice("node_engines", executors.Keys(ctx)),
		)

		var hbClient *heartbeat.HeartbeatClient

		// We want to provide a heartbeat client to the compute node if we are using NATS.
		// We can only create a heartbeat client if we have a NATS client, and we can
		// only do that if the configuration is available. Whilst we support libp2p this
		// is not always the case.
		if natsConfig != nil {
			natsClient, err := nats_transport.CreateClient(ctx, natsConfig)
			if err != nil {
				return nil, pkgerrors.Wrap(err, "failed to create NATS client for node info store")
			}

			hbClient, err = heartbeat.NewClient(
				natsClient.Client,
				config.NodeID,
				config.ComputeConfig.ControlPlaneSettings.HeartbeatTopic,
			)
			if err != nil {
				return nil, pkgerrors.Wrap(err, "failed to create heartbeat client")
			}
		}

		// setup compute node
		computeNode, err = NewComputeNode(
			ctx,
			config.NodeID,
			config.CleanupManager,
			apiServer,
			config.ComputeConfig,
			storagePath,
			storageProviders,
			executors,
			publishers,
			transportLayer.CallbackProxy(),
			transportLayer.ManagementProxy(),
			config.Labels,
			hbClient,
		)
		if err != nil {
			return nil, err
		}

		err = transportLayer.RegisterComputeEndpoint(computeNode.LocalEndpoint)
		if err != nil {
			return nil, err
		}

		labelsProvider = computeNode.labelsProvider
		debugInfoProviders = append(debugInfoProviders, computeNode.debugInfoProviders...)
	}

	// Create a node info provider for LibP2P, and specify the default node approval state
	// of Approved to avoid confusion as approval state is not used for this transport type.
	nodeInfoProvider := routing.NewNodeStateProvider(routing.NodeStateProviderParams{
		NodeID:              config.NodeID,
		LabelsProvider:      labelsProvider,
		BacalhauVersion:     *version.Get(),
		DefaultNodeApproval: models.NodeMembership.APPROVED,
	})
	nodeInfoProvider.RegisterNodeInfoDecorator(transportLayer.NodeInfoDecorator())
	if computeNode != nil {
		nodeInfoProvider.RegisterNodeInfoDecorator(computeNode.nodeInfoDecorator)
	}

	shared.NewEndpoint(shared.EndpointParams{
		Router:            apiServer.Router,
		NodeID:            config.NodeID,
		NodeStateProvider: nodeInfoProvider,
	})

	agent.NewEndpoint(agent.EndpointParams{
		Router:             apiServer.Router,
		NodeStateProvider:  nodeInfoProvider,
		DebugInfoProviders: debugInfoProviders,
	})

	var nodeInfoPublisher *routing.NodeInfoPublisher
	if config.NetworkConfig.Type != models.NetworkTypeNATS {
		// We do not want to keep publishing node information if we are
		// using NATS. We will initially call the management endpoint
		// and then send less static information separately.
		nodeInfoPublisherInterval := config.NodeInfoPublisherInterval
		if nodeInfoPublisherInterval.IsZero() {
			nodeInfoPublisherInterval = GetNodeInfoPublishConfig()
		}

		// NB(forrest): this must be done last to avoid eager publishing before nodes are constructed
		// TODO(forrest) [fixme] we should fix this to make it less racy in testing
		nodeInfoPublisher = routing.NewNodeInfoPublisher(routing.NodeInfoPublisherParams{
			PubSub:            transportLayer.NodeInfoPubSub(),
			NodeStateProvider: nodeInfoProvider,
			IntervalConfig:    nodeInfoPublisherInterval,
		})
	} else {
		// We want to register the current requester node to the node store
		if config.IsRequesterNode {
			nodeState := nodeInfoProvider.GetNodeState(ctx)
			// TODO what is the liveness here? We are adding ourselves so I assume connected?
			nodeState.Membership = models.NodeMembership.APPROVED
			if err := tracingInfoStore.Add(ctx, nodeState); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to add requester node to the node store")
				return nil, fmt.Errorf("registering node to the node store: %w", err)
			}
		}
	}

	// Start periodic software update checks.
	updateCheckCtx, stopUpdateChecks := context.WithCancel(ctx)
	version.RunUpdateChecker(
		updateCheckCtx,
		// TODO(forrest) [correctness]: this code is literally the server, why are we returning nil???!!!
		func(ctx context.Context) (*models.BuildVersionInfo, error) { return nil, nil },
		version.LogUpdateResponse,
	)
	config.CleanupManager.RegisterCallback(func() error {
		stopUpdateChecks()
		return nil
	})

	// Cleanup libp2p resources in the desired order
	config.CleanupManager.RegisterCallbackWithContext(func(ctx context.Context) error {
		if computeNode != nil {
			computeNode.Cleanup(ctx)
		}
		if requesterNode != nil {
			requesterNode.cleanup(ctx)
		}

		if nodeInfoPublisher != nil {
			nodeInfoPublisher.Stop(ctx)
		}

		var err error
		if transportLayer != nil {
			err = errors.Join(err, transportLayer.Close(ctx))
		}

		if apiServer != nil {
			err = errors.Join(err, apiServer.Shutdown(ctx))
		}

		cancel()
		return err
	})

	metrics.NodeInfo.Add(ctx, 1,
		attribute.String("node_id", config.NodeID),
		attribute.String("node_network_transport", config.NetworkConfig.Type),
		attribute.Bool("node_is_compute", config.IsComputeNode),
		attribute.Bool("node_is_requester", config.IsRequesterNode),
		attribute.StringSlice("node_storages", storageProviders.Keys(ctx)),
	)
	node := &Node{
		ID:             config.NodeID,
		CleanupManager: config.CleanupManager,
		APIServer:      apiServer,
		IPFSClient:     config.IPFSClient,
		ComputeNode:    computeNode,
		RequesterNode:  requesterNode,
		Libp2pHost:     config.NetworkConfig.Libp2pHost,
	}

	return node, nil
}

// IsRequesterNode returns true if the node is a requester node
func (n *Node) IsRequesterNode() bool {
	return n.RequesterNode != nil
}

// IsComputeNode returns true if the node is a compute node
func (n *Node) IsComputeNode() bool {
	return n.ComputeNode != nil
}

func mergeDependencyInjectors(injector NodeDependencyInjector, defaultInjector NodeDependencyInjector) NodeDependencyInjector {
	if injector.StorageProvidersFactory == nil {
		injector.StorageProvidersFactory = defaultInjector.StorageProvidersFactory
	}
	if injector.ExecutorsFactory == nil {
		injector.ExecutorsFactory = defaultInjector.ExecutorsFactory
	}
	if injector.PublishersFactory == nil {
		injector.PublishersFactory = defaultInjector.PublishersFactory
	}
	if injector.AuthenticatorsFactory == nil {
		injector.AuthenticatorsFactory = defaultInjector.AuthenticatorsFactory
	}
	return injector
}
