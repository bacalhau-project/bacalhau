package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/imdario/mergo"
	"github.com/labstack/echo/v4"
	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
	"github.com/bacalhau-project/bacalhau/pkg/node/metrics"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/agent"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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
	DependencyInjector          NodeDependencyInjector
	AllowListedLocalPaths       []string
	NetworkConfig               NetworkConfig
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

func NewExecutorPluginNodeDependencyInjector(cfg types.BacalhauConfig) NodeDependencyInjector {
	return NodeDependencyInjector{
		StorageProvidersFactory: NewStandardStorageProvidersFactory(cfg),
		ExecutorsFactory:        NewPluginExecutorFactory(cfg.Node.ExecutorPluginPath),
		PublishersFactory:       NewStandardPublishersFactory(cfg),
		AuthenticatorsFactory:   NewStandardAuthenticatorsFactory(cfg.User.KeyPath),
	}
}

func NewStandardNodeDependencyInjector(cfg types.BacalhauConfig) NodeDependencyInjector {
	return NodeDependencyInjector{
		StorageProvidersFactory: NewStandardStorageProvidersFactory(cfg),
		ExecutorsFactory:        NewStandardExecutorsFactory(cfg.Node.Compute.ManifestCache),
		PublishersFactory:       NewStandardPublishersFactory(cfg),
		AuthenticatorsFactory:   NewStandardAuthenticatorsFactory(cfg.User.KeyPath),
	}
}

type Node struct {
	// Visible for testing
	ID             string
	APIServer      *publicapi.Server
	ComputeNode    *Compute
	RequesterNode  *Requester
	CleanupManager *system.CleanupManager
}

func (n *Node) Start(ctx context.Context) error {
	return n.APIServer.ListenAndServe(ctx)
}

//nolint:funlen,gocyclo // Should be simplified when moving to FX
func NewNode(
	ctx context.Context,
	bacalhauConfig v2.Bacalhau,
	config NodeConfig,
	fsr *repo.FsRepo,
) (*Node, error) {
	var err error
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	authProviders, err := SetupAuthenticators("TODO", bacalhauConfig)
	if err != nil {
		return nil, fmt.Errorf("creating auth provider: %w", err)
	}
	apiServer, err := SetupAPIServer("TODO", bacalhauConfig)
	if err != nil {
		return nil, fmt.Errorf("creating api server: %w", err)
	}
	transport, err := SetupTransport(bacalhauConfig)

	/*
		if err = prepareConfig(&config, bacalhauConfig); err != nil {
			return nil, err
		}

		apiServer, err := createAPIServer(config, bacalhauConfig)
		if err != nil {
			return nil, err
		}

		transportLayer, err := createTransport(ctx, config)
		if err != nil {
			return nil, err
		}

	*/

	var debugInfoProviders []models.DebugInfoProvider
	debugInfoProviders = append(debugInfoProviders, transport.DebugInfoProviders()...)

	var requesterNode *Requester
	var computeNode *Compute
	var labelsProvider models.LabelsProvider

	// setup requester node
	if config.IsRequesterNode {
		requesterNode, err = NewRequesterNode(
			ctx,
			config.NodeID,
			apiServer,
			config,
			bacalhauConfig.Metrics,
			config.RequesterNodeConfig,
			transport,
			transport.ComputeProxy(),
		)
		if err != nil {
			return nil, err
		}

		labelsProvider = models.MergeLabelsInOrder(
			&ConfigLabelsProvider{staticLabels: config.Labels},
			&RuntimeLabelsProvider{},
		)
		debugInfoProviders = append(debugInfoProviders, requesterNode.debugInfoProviders...)
	}

	if config.IsComputeNode {
		storagePath := bacalhauConfig.Node.ComputeStoragePath

		// Setup dependencies
		publishers, err := config.DependencyInjector.PublishersFactory.Get(ctx, config)
		if err != nil {
			return nil, err
		}

		executors, err := config.DependencyInjector.ExecutorsFactory.Get(ctx, config)
		if err != nil {
			return nil, err
		}

		storages, err := config.DependencyInjector.StorageProvidersFactory.Get(ctx, config)
		if err != nil {
			return nil, err
		}

		// TODO calling `Keys` on the publishers takes ~10 seconds per call
		// https://github.com/bacalhau-project/bacalhau/issues/4153
		metrics.NodeInfo.Add(ctx, 1,
			attribute.StringSlice("node_publishers", publishers.Keys(ctx)),
			attribute.StringSlice("node_storages", storages.Keys(ctx)),
			attribute.StringSlice("node_engines", executors.Keys(ctx)),
		)

		// heartbeat client
		heartbeatClient, err := heartbeat.NewClient(
			transport.Client(), config.NodeID, config.ComputeConfig.ControlPlaneSettings.HeartbeatTopic,
		)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to create heartbeat client")
		}

		repoPath, err := fsr.Path()
		if err != nil {
			return nil, err
		}
		// setup compute node
		computeNode, err = NewComputeNode(
			ctx,
			config.NodeID,
			apiServer,
			config.ComputeConfig,
			storagePath,
			repoPath,
			storages,
			executors,
			publishers,
			transport.CallbackProxy(),
			transport.ManagementProxy(),
			config.Labels,
			heartbeatClient,
		)
		if err != nil {
			return nil, err
		}

		err = transport.RegisterComputeEndpoint(ctx, computeNode.LocalEndpoint)
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
	nodeInfoProvider.RegisterNodeInfoDecorator(transport.NodeInfoDecorator())
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

	// We want to register the current requester node to the node store
	// TODO (walid): revisit self node registration of requester node
	if config.IsRequesterNode {
		nodeState := nodeInfoProvider.GetNodeState(ctx)
		// TODO what is the liveness here? We are adding ourselves so I assume connected?
		nodeState.Membership = models.NodeMembership.APPROVED
		if err := requesterNode.NodeInfoStore.Add(ctx, nodeState); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to add requester node to the node store")
			return nil, fmt.Errorf("registering node to the node store: %w", err)
		}
	}

	// Start periodic software update checks.
	updateCheckCtx, stopUpdateChecks := context.WithCancel(ctx)
	version.RunUpdateChecker(
		updateCheckCtx,
		bacalhauConfig,
		func(ctx context.Context) (*models.BuildVersionInfo, error) { return nil, nil },
		version.LogUpdateResponse,
	)

	// Cleanup libp2p resources in the desired order
	config.CleanupManager.RegisterCallbackWithContext(func(ctx context.Context) error {
		if computeNode != nil {
			computeNode.Cleanup(ctx)
		}
		if requesterNode != nil {
			requesterNode.cleanup(ctx)
		}

		stopUpdateChecks()

		var err error
		if transportLayer != nil {
			err = errors.Join(err, transport.Close(ctx))
		}

		if apiServer != nil {
			err = errors.Join(err, apiServer.Shutdown(ctx))
		}

		cancel()
		return err
	})

	metrics.NodeInfo.Add(ctx, 1,
		attribute.String("node_id", config.NodeID),
		attribute.Bool("node_is_compute", config.IsComputeNode),
		attribute.Bool("node_is_requester", config.IsRequesterNode),
	)
	node := &Node{
		ID:             config.NodeID,
		CleanupManager: config.CleanupManager,
		APIServer:      apiServer,
		ComputeNode:    computeNode,
		RequesterNode:  requesterNode,
	}

	return node, nil
}

func prepareConfig(config *NodeConfig, bacalhauConfig types.BacalhauConfig) error {
	config.DependencyInjector =
		mergeDependencyInjectors(config.DependencyInjector, NewStandardNodeDependencyInjector(bacalhauConfig))
	err := mergo.Merge(&config.APIServerConfig, publicapi.DefaultConfig())
	if err != nil {
		return err
	}
	// TODO: #830 Same as #829 in pkg/eventhandler/chained_handlers.go
	if system.GetEnvironment() == system.EnvironmentTest || system.GetEnvironment() == system.EnvironmentDev {
		config.APIServerConfig.LogLevel = "trace"
	}

	err = config.Validate()
	if err != nil {
		return fmt.Errorf("error validating node config. %w", err)
	}
	return nil
}

func createAPIServer(config NodeConfig, bacalhauConfig types.BacalhauConfig) (*publicapi.Server, error) {
	authzPolicy, err := policy.FromPathOrDefault(config.AuthConfig.AccessPolicyPath, authz.AlwaysAllowPolicy)
	if err != nil {
		return nil, err
	}

	signingKey, err := loadUserIDKey(bacalhauConfig.User.KeyPath)
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
		Authorizer: authz.NewPolicyAuthorizer(authzPolicy, &signingKey.PublicKey, config.NodeID),
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
	return apiServer, nil
}

func createTransport(ctx context.Context, config NodeConfig) (*nats_transport.NATSTransport, error) {
	transportLayer, err := nats_transport.NewNATSTransport(ctx, &nats_transport.NATSTransportConfig{
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
	})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create NATS transport layer")
	}
	return transportLayer, nil
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
