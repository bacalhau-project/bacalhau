package node

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	libp2p_transport "github.com/bacalhau-project/bacalhau/pkg/libp2p/transport"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/agent"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/bacalhau-project/bacalhau/pkg/version"
	"github.com/hashicorp/go-multierror"
	"github.com/imdario/mergo"
	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p/core/host"
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
	DisabledFeatures            FeatureConfig
	ComputeConfig               ComputeConfig
	RequesterNodeConfig         RequesterConfig
	APIServerConfig             publicapi.Config
	IsRequesterNode             bool
	IsComputeNode               bool
	Labels                      map[string]string
	NodeInfoPublisherInterval   routing.NodeInfoPublisherIntervalConfig
	DependencyInjector          NodeDependencyInjector
	AllowListedLocalPaths       []string
	NodeInfoStoreTTL            time.Duration

	FsRepo        *repo.FsRepo
	NetworkConfig NetworkConfig
}

func (c *NodeConfig) Validate() error {
	// TODO: add more validations
	var mErr *multierror.Error
	mErr = multierror.Append(mErr, c.NetworkConfig.Validate())
	return mErr.ErrorOrNil()
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
	NodeInfoStore  routing.NodeInfoStore
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

	publishers, err := config.DependencyInjector.PublishersFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	executors, err := config.DependencyInjector.ExecutorsFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	authenticators, err := config.DependencyInjector.AuthenticatorsFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	// timeoutHandler doesn't implement http.Hijacker, so we need to skip it for websocket endpoints
	config.APIServerConfig.SkippedTimeoutPaths = append(config.APIServerConfig.SkippedTimeoutPaths, []string{
		"/api/v1/requester/websocket/events",
		"/api/v1/requester/logs",
	}...)

	serverVersion := version.Get()
	// public http api server
	serverParams := publicapi.ServerParams{
		Router:     echo.New(),
		Address:    config.HostAddress,
		Port:       config.APIPort,
		HostID:     config.NodeID,
		Config:     config.APIServerConfig,
		Authorizer: authz.AlwaysAllow,
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
	nodeInfoStore := inmemory.NewNodeInfoStore(inmemory.NodeInfoStoreParams{
		TTL: config.NodeInfoStoreTTL,
	})

	var transportLayer transport.TransportLayer

	if config.NetworkConfig.Type == models.NetworkTypeNATS {
		natsConfig := nats_transport.NATSTransportConfig{
			NodeID:                   config.NodeID,
			Port:                     config.NetworkConfig.Port,
			AdvertisedAddress:        config.NetworkConfig.AdvertisedAddress,
			Orchestrators:            config.NetworkConfig.Orchestrators,
			ClusterName:              config.NetworkConfig.ClusterName,
			ClusterPort:              config.NetworkConfig.ClusterPort,
			ClusterPeers:             config.NetworkConfig.ClusterPeers,
			ClusterAdvertisedAddress: config.NetworkConfig.ClusterAdvertisedAddress,
			IsRequesterNode:          config.IsRequesterNode,
		}
		transportLayer, err = nats_transport.NewNATSTransport(ctx, natsConfig, nodeInfoStore)
	} else {
		libp2pConfig := libp2p_transport.Libp2pTransportConfig{
			Host:           config.NetworkConfig.Libp2pHost,
			Peers:          config.NetworkConfig.ClusterPeers,
			ReconnectDelay: config.NetworkConfig.ReconnectDelay,
			CleanupManager: config.CleanupManager,
		}
		transportLayer, err = libp2p_transport.NewLibp2pTransport(ctx, libp2pConfig, nodeInfoStore)
	}
	if err != nil {
		return nil, err
	}

	var debugInfoProviders []model.DebugInfoProvider
	debugInfoProviders = append(debugInfoProviders, transportLayer.DebugInfoProviders()...)

	var requesterNode *Requester
	var computeNode *Compute
	var labelsProvider models.LabelsProvider = &ConfigLabelsProvider{staticLabels: config.Labels}

	// setup requester node
	if config.IsRequesterNode {
		requesterNode, err = NewRequesterNode(
			ctx,
			config.NodeID,
			apiServer,
			config.RequesterNodeConfig,
			storageProviders,
			authenticators,
			nodeInfoStore,
			config.FsRepo,
			transportLayer.ComputeProxy(),
		)
		if err != nil {
			return nil, err
		}
		err = transportLayer.RegisterComputeCallback(requesterNode.localCallback)
		if err != nil {
			return nil, err
		}
		debugInfoProviders = append(debugInfoProviders, requesterNode.debugInfoProviders...)
	}

	if config.IsComputeNode {
		storagePath := pkgconfig.GetStoragePath()

		// setup compute node
		computeNode, err = NewComputeNode(
			ctx,
			config.NodeID,
			config.CleanupManager,
			config.NetworkConfig.Libp2pHost,
			apiServer,
			config.ComputeConfig,
			storagePath,
			storageProviders,
			executors,
			publishers,
			config.FsRepo,
			transportLayer.CallbackProxy(),
		)
		if err != nil {
			return nil, err
		}

		err = transportLayer.RegisterComputeEndpoint(computeNode.LocalEndpoint)
		if err != nil {
			return nil, err
		}

		labelsProvider = models.MergeLabelsInOrder(
			computeNode.autoLabelsProvider,
			labelsProvider,
		)
		debugInfoProviders = append(debugInfoProviders, computeNode.debugInfoProviders...)
	}

	nodeInfoProvider := routing.NewNodeInfoProvider(routing.NodeInfoProviderParams{
		NodeID:          config.NodeID,
		LabelsProvider:  labelsProvider,
		BacalhauVersion: *version.Get(),
	})
	nodeInfoProvider.RegisterNodeInfoDecorator(transportLayer.NodeInfoDecorator())
	if computeNode != nil {
		nodeInfoProvider.RegisterNodeInfoDecorator(computeNode.nodeInfoDecorator)
	}

	shared.NewEndpoint(shared.EndpointParams{
		Router:           apiServer.Router,
		NodeID:           config.NodeID,
		NodeInfoProvider: nodeInfoProvider,
	})

	agent.NewEndpoint(agent.EndpointParams{
		Router:             apiServer.Router,
		NodeInfoProvider:   nodeInfoProvider,
		DebugInfoProviders: debugInfoProviders,
	})

	// node info publisher
	nodeInfoPublisherInterval := config.NodeInfoPublisherInterval
	if nodeInfoPublisherInterval.IsZero() {
		nodeInfoPublisherInterval = GetNodeInfoPublishConfig()
	}

	// NB(forrest): this must be done last to avoid eager publishing before nodes are constructed
	// TODO(forrest) [fixme] we should fix this to make it less racy in testing
	nodeInfoPublisher := routing.NewNodeInfoPublisher(routing.NodeInfoPublisherParams{
		PubSub:           transportLayer.NodeInfoPubSub(),
		NodeInfoProvider: nodeInfoProvider,
		IntervalConfig:   nodeInfoPublisherInterval,
	})

	// Start periodic software update checks.
	updateCheckCtx, stopUpdateChecks := context.WithCancel(ctx)
	version.RunUpdateChecker(
		updateCheckCtx,
		func(ctx context.Context) (*models.BuildVersionInfo, error) { return nil, nil },
		version.LogUpdateResponse,
	)
	config.CleanupManager.RegisterCallback(func() error {
		stopUpdateChecks()
		return nil
	})

	// cleanup libp2p resources in the desired order
	config.CleanupManager.RegisterCallbackWithContext(func(ctx context.Context) error {
		if computeNode != nil {
			computeNode.cleanup(ctx)
		}
		if requesterNode != nil {
			requesterNode.cleanup(ctx)
		}
		nodeInfoPublisher.Stop(ctx)

		var errors *multierror.Error
		errors = multierror.Append(errors, transportLayer.Close(ctx))
		errors = multierror.Append(errors, apiServer.Shutdown(ctx))
		cancel()
		return errors.ErrorOrNil()
	})

	node := &Node{
		ID:             config.NodeID,
		CleanupManager: config.CleanupManager,
		APIServer:      apiServer,
		IPFSClient:     config.IPFSClient,
		ComputeNode:    computeNode,
		RequesterNode:  requesterNode,
		NodeInfoStore:  nodeInfoStore,
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
