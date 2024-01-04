package node

import (
	"context"
	"fmt"
	"time"

	"github.com/imdario/mergo"
	"github.com/labstack/echo/v4"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/agent"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"

	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

const JobInfoTopic = "bacalhau-job-info"
const NodeInfoTopic = "bacalhau-node-info"

type FeatureConfig struct {
	Engines    []string
	Publishers []string
	Storages   []string
}

// Node configuration
type NodeConfig struct {
	IPFSClient                  ipfs.Client
	CleanupManager              *system.CleanupManager
	Host                        host.Host
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

	FsRepo *repo.FsRepo
}

// Lazy node dependency injector that generate instances of different
// components on demand and based on the configuration provided.
type NodeDependencyInjector struct {
	StorageProvidersFactory StorageProvidersFactory
	ExecutorsFactory        ExecutorsFactory
	PublishersFactory       PublishersFactory
}

func NewExecutorPluginNodeDependencyInjector() NodeDependencyInjector {
	return NodeDependencyInjector{
		StorageProvidersFactory: NewStandardStorageProvidersFactory(),
		ExecutorsFactory:        NewPluginExecutorFactory(),
		PublishersFactory:       NewStandardPublishersFactory(),
	}
}

func NewStandardNodeDependencyInjector() NodeDependencyInjector {
	return NodeDependencyInjector{
		StorageProvidersFactory: NewStandardStorageProvidersFactory(),
		ExecutorsFactory:        NewStandardExecutorsFactory(),
		PublishersFactory:       NewStandardPublishersFactory(),
	}
}

type Node struct {
	// Visible for testing
	APIServer      *publicapi.Server
	ComputeNode    *Compute
	RequesterNode  *Requester
	NodeInfoStore  routing.NodeInfoStore
	CleanupManager *system.CleanupManager
	IPFSClient     ipfs.Client
	Host           host.Host
}

func (n *Node) Start(ctx context.Context) error {
	return n.APIServer.ListenAndServe(ctx)
}

//nolint:funlen,gocyclo // Should be simplified when moving to FX
func NewNode(
	ctx context.Context,
	config NodeConfig) (*Node, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/node.NewNode")
	defer span.End()

	identify.ActivationThresh = 2

	config.DependencyInjector = mergeDependencyInjectors(config.DependencyInjector, NewStandardNodeDependencyInjector())
	err := mergo.Merge(&config.APIServerConfig, publicapi.DefaultConfig())
	if err != nil {
		return nil, err
	}
	// TODO: #830 Same as #829 in pkg/eventhandler/chained_handlers.go
	if system.GetEnvironment() == system.EnvironmentTest || system.GetEnvironment() == system.EnvironmentDev {
		config.APIServerConfig.LogLevel = "trace"
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

	// A single gossipSub instance that will be used by all topics
	gossipSubCtx, gossipSubCancel := context.WithCancel(ctx)
	gossipSub, err := newLibp2pPubSub(gossipSubCtx, config)
	defer func() {
		if err != nil {
			gossipSubCancel()
		}
	}()

	if err != nil {
		return nil, err
	}

	// PubSub to publish node info to the network
	nodeInfoPubSub, err := libp2p.NewPubSub[models.NodeInfo](libp2p.PubSubParams{
		Host:      config.Host,
		TopicName: NodeInfoTopic,
		PubSub:    gossipSub,
	})
	if err != nil {
		return nil, err
	}

	// node info publisher
	nodeInfoPublisherInterval := config.NodeInfoPublisherInterval
	if nodeInfoPublisherInterval.IsZero() {
		nodeInfoPublisherInterval = GetNodeInfoPublishConfig()
	}

	// node info store that is used for both discovering compute nodes, as to find addresses of other nodes for routing requests.
	nodeInfoStore := inmemory.NewNodeInfoStore(inmemory.NodeInfoStoreParams{
		TTL: config.NodeInfoStoreTTL,
	})
	routedHost := routedhost.Wrap(config.Host, nodeInfoStore)

	// register consumers of node info published over gossipSub
	nodeInfoSubscriber := pubsub.NewChainedSubscriber[models.NodeInfo](true)
	nodeInfoSubscriber.Add(pubsub.SubscriberFunc[models.NodeInfo](nodeInfoStore.Add))
	err = nodeInfoPubSub.Subscribe(ctx, nodeInfoSubscriber)
	if err != nil {
		return nil, err
	}

	// timeoutHandler doesn't implement http.Hijacker, so we need to skip it for websocket endpoints
	config.APIServerConfig.SkippedTimeoutPaths = append(config.APIServerConfig.SkippedTimeoutPaths, []string{
		"/api/v1/requester/websocket/events",
		"/api/v1/requester/logs",
	}...)

	// public http api server
	serverParams := publicapi.ServerParams{
		Router:  echo.New(),
		Address: config.HostAddress,
		Port:    config.APIPort,
		HostID:  config.Host.ID().String(),
		Config:  config.APIServerConfig,
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

	var requesterNode *Requester
	var computeNode *Compute

	var computeInfoProvider models.ComputeNodeInfoProvider
	var labelsProvider models.LabelsProvider = &ConfigLabelsProvider{staticLabels: config.Labels}

	// setup requester node
	if config.IsRequesterNode {
		requesterNode, err = NewRequesterNode(
			ctx,
			routedHost,
			apiServer,
			config.RequesterNodeConfig,
			storageProviders,
			nodeInfoStore,
			gossipSub,
			config.FsRepo,
		)
		if err != nil {
			return nil, err
		}
	}

	if config.IsComputeNode {
		storagePath := pkgconfig.GetStoragePath()

		// setup compute node
		computeNode, err = NewComputeNode(
			ctx,
			config.CleanupManager,
			routedHost,
			apiServer,
			config.ComputeConfig,
			storagePath,
			storageProviders,
			executors,
			publishers,
			config.FsRepo,
		)
		if err != nil {
			return nil, err
		}

		computeInfoProvider = computeNode.computeInfoProvider
		labelsProvider = models.MergeLabelsInOrder(
			computeNode.autoLabelsProvider,
			labelsProvider,
		)
	}

	// node info provider
	basicHost, ok := config.Host.(*basichost.BasicHost)
	if !ok {
		return nil, fmt.Errorf("host is not a basic host")
	}
	nodeInfoProvider := routing.NewNodeInfoProvider(routing.NodeInfoProviderParams{
		Host:                basicHost,
		IdentityService:     basicHost.IDService(),
		LabelsProvider:      labelsProvider,
		ComputeInfoProvider: computeInfoProvider,
		BacalhauVersion:     *version.Get(),
	})

	shared.NewEndpoint(shared.EndpointParams{
		Router:           apiServer.Router,
		NodeID:           config.Host.ID().String(),
		PeerStore:        config.Host.Peerstore(),
		NodeInfoProvider: nodeInfoProvider,
	})

	agent.NewEndpoint(agent.EndpointParams{
		Router:           apiServer.Router,
		NodeInfoProvider: nodeInfoProvider,
	})

	// NB(forrest): this must be done last to avoid eager publishing before nodes are constructed
	// TODO(forrest) [fixme] we should fix this to make it less racy in testing
	nodeInfoPublisher := routing.NewNodeInfoPublisher(routing.NodeInfoPublisherParams{
		PubSub:           nodeInfoPubSub,
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
		cleanupErr := nodeInfoPubSub.Close(ctx)
		util.LogDebugIfContextCancelled(ctx, cleanupErr, "node info pub sub")
		gossipSubCancel()

		cleanupErr = config.Host.Close()
		util.LogDebugIfContextCancelled(ctx, cleanupErr, "host")

		cleanupErr = apiServer.Shutdown(ctx)
		return cleanupErr
	})

	if requesterNode != nil && computeNode != nil {
		// To enable nodes self-dialing themselves as libp2p doesn't support it.
		computeNode.RegisterLocalComputeCallback(requesterNode.localCallback)
		requesterNode.RegisterLocalComputeEndpoint(computeNode.LocalEndpoint)
	}

	node := &Node{
		CleanupManager: config.CleanupManager,
		APIServer:      apiServer,
		IPFSClient:     config.IPFSClient,
		ComputeNode:    computeNode,
		RequesterNode:  requesterNode,
		NodeInfoStore:  nodeInfoStore,
		Host:           routedHost,
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

func newLibp2pPubSub(ctx context.Context, nodeConfig NodeConfig) (*libp2p_pubsub.PubSub, error) {
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
		nodeConfig.Host,
		libp2p_pubsub.WithPeerExchange(true),
		libp2p_pubsub.WithPeerGater(pgParams),
		libp2p_pubsub.WithEventTracer(tracer),
	)
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
	return injector
}
