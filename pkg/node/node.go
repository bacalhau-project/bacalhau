package node

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	filecoinlotus "github.com/bacalhau-project/bacalhau/pkg/publisher/filecoin_lotus"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/simulator"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/bacalhau-project/bacalhau/pkg/version"
	"github.com/imdario/mergo"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/rs/zerolog/log"
)

const JobEventsTopic = "bacalhau-job-events"
const NodeInfoTopic = "bacalhau-node-info"
const DefaultNodeInfoPublisherInterval = 30 * time.Second

type FeatureConfig struct {
	Engines    []model.Engine
	Verifiers  []model.Verifier
	Publishers []model.Publisher
	Storages   []model.StorageSourceType
}

// Node configuration
type NodeConfig struct {
	IPFSClient                ipfs.Client
	CleanupManager            *system.CleanupManager
	JobStore                  jobstore.Store
	Host                      host.Host
	FilecoinUnsealedPath      string
	EstuaryAPIKey             string
	HostAddress               string
	APIPort                   uint16
	DisabledFeatures          FeatureConfig
	ComputeConfig             ComputeConfig
	RequesterNodeConfig       RequesterConfig
	APIServerConfig           publicapi.APIServerConfig
	LotusConfig               *filecoinlotus.PublisherConfig
	SimulatorNodeID           string
	IsRequesterNode           bool
	IsComputeNode             bool
	Labels                    map[string]string
	NodeInfoPublisherInterval time.Duration
	DependencyInjector        NodeDependencyInjector
}

// Lazy node dependency injector that generate instances of different
// components on demand and based on the configuration provided.
type NodeDependencyInjector struct {
	StorageProvidersFactory StorageProvidersFactory
	ExecutorsFactory        ExecutorsFactory
	VerifiersFactory        VerifiersFactory
	PublishersFactory       PublishersFactory
}

func NewStandardNodeDependencyInjector() NodeDependencyInjector {
	return NodeDependencyInjector{
		StorageProvidersFactory: NewStandardStorageProvidersFactory(),
		ExecutorsFactory:        NewStandardExecutorsFactory(),
		VerifiersFactory:        NewStandardVerifiersFactory(),
		PublishersFactory:       NewStandardPublishersFactory(),
	}
}

type Node struct {
	// Visible for testing
	APIServer      *publicapi.APIServer
	ComputeNode    *Compute
	RequesterNode  *Requester
	NodeInfoStore  routing.NodeInfoStore
	CleanupManager *system.CleanupManager
	IPFSClient     ipfs.Client
	Host           host.Host
}

func (n *Node) Start(ctx context.Context) error {
	return n.APIServer.ListenAndServe(ctx, n.CleanupManager)
}

//nolint:funlen,gocyclo // Should be simplified when moving to FX
func NewNode(
	ctx context.Context,
	config NodeConfig) (*Node, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/node.NewNode")
	defer span.End()

	identify.ActivationThresh = 2

	config.DependencyInjector = mergeDependencyInjectors(config.DependencyInjector, NewStandardNodeDependencyInjector())
	err := mergo.Merge(&config.APIServerConfig, publicapi.DefaultAPIServerConfig)
	if err != nil {
		return nil, err
	}

	storageProviders, err := config.DependencyInjector.StorageProvidersFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	executors, err := config.DependencyInjector.ExecutorsFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	verifiers, err := config.DependencyInjector.VerifiersFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	publishers, err := config.DependencyInjector.PublishersFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	var simulatorRequestHandler *simulator.RequestHandler
	if config.SimulatorNodeID == config.Host.ID().String() {
		log.Ctx(ctx).Info().Msgf("Node %s is the simulator node. Setting proper event handlers", config.Host.ID().String())
		simulatorRequestHandler = simulator.NewRequestHandler()
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
	nodeInfoPubSub, err := libp2p.NewPubSub[model.NodeInfo](libp2p.PubSubParams{
		Host:      config.Host,
		TopicName: NodeInfoTopic,
		PubSub:    gossipSub,
	})
	if err != nil {
		return nil, err
	}

	// node info provider
	basicHost, ok := config.Host.(*basichost.BasicHost)
	if !ok {
		return nil, fmt.Errorf("host is not a basic host")
	}
	nodeInfoProvider := routing.NewNodeInfoProvider(routing.NodeInfoProviderParams{
		Host:            basicHost,
		IdentityService: basicHost.IDService(),
		Labels:          config.Labels,
		BacalhauVersion: *version.Get(),
	})

	// node info publisher
	nodeInfoPublisherInterval := config.NodeInfoPublisherInterval
	if nodeInfoPublisherInterval == 0 {
		nodeInfoPublisherInterval = DefaultNodeInfoPublisherInterval
	}
	nodeInfoPublisher := routing.NewNodeInfoPublisher(routing.NodeInfoPublisherParams{
		PubSub:           nodeInfoPubSub,
		NodeInfoProvider: nodeInfoProvider,
		Interval:         nodeInfoPublisherInterval,
	})

	// node info store that is used for both discovering compute nodes, as to find addresses of other nodes for routing requests.
	nodeInfoStore := inmemory.NewNodeInfoStore(inmemory.NodeInfoStoreParams{
		TTL: 10 * time.Minute,
	})
	routedHost := routedhost.Wrap(config.Host, nodeInfoStore)

	// register consumers of node info published over gossipSub
	nodeInfoSubscriber := pubsub.NewChainedSubscriber[model.NodeInfo](true)
	nodeInfoSubscriber.Add(pubsub.SubscriberFunc[model.NodeInfo](nodeInfoStore.Add))
	err = nodeInfoPubSub.Subscribe(ctx, nodeInfoSubscriber)
	if err != nil {
		return nil, err
	}

	// public http api server
	apiServer, err := publicapi.NewAPIServer(publicapi.APIServerParams{
		Address:          config.HostAddress,
		Port:             config.APIPort,
		Host:             config.Host,
		Config:           config.APIServerConfig,
		NodeInfoProvider: nodeInfoProvider,
	})
	if err != nil {
		return nil, err
	}

	var requesterNode *Requester
	var computeNode *Compute

	// setup requester node
	if config.IsRequesterNode {
		requesterNode, err = NewRequesterNode(
			ctx,
			config.CleanupManager,
			routedHost,
			apiServer,
			config.RequesterNodeConfig,
			config.JobStore,
			config.SimulatorNodeID,
			simulatorRequestHandler,
			verifiers,
			storageProviders,
			gossipSub,
			nodeInfoStore,
		)
		if err != nil {
			return nil, err
		}
	}

	if config.IsComputeNode {
		// setup compute node
		computeNode, err = NewComputeNode(
			ctx,
			config.CleanupManager,
			routedHost,
			apiServer,
			config.ComputeConfig,
			config.SimulatorNodeID,
			simulatorRequestHandler,
			storageProviders,
			executors,
			verifiers,
			publishers,
		)
		if err != nil {
			return nil, err
		}
		nodeInfoProvider.RegisterComputeInfoProvider(computeNode.computeInfoProvider)
	}

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
		return cleanupErr
	})

	if requesterNode != nil && computeNode != nil {
		// To enable nodes self-dialing themselves as libp2p doesn't support it.
		computeNode.RegisterLocalComputeCallback(requesterNode.localCallback)
		requesterNode.RegisterLocalComputeEndpoint(computeNode.LocalEndpoint)
	}

	// Eagerly publish node info to the network. Do this in a goroutine so that
	// slow plugins don't slow down the node from booting.
	go func() {
		err = nodeInfoPublisher.Publish(ctx)
		log.Ctx(ctx).WithLevel(logger.ErrOrDebug(err)).Err(err).Msg("Eagerly published node info")
	}()

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
	tracer, err := libp2p_pubsub.NewJSONTracer(config.GetLibp2pTracerPath())
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
	if injector.VerifiersFactory == nil {
		injector.VerifiersFactory = defaultInjector.VerifiersFactory
	}
	if injector.PublishersFactory == nil {
		injector.PublishersFactory = defaultInjector.PublishersFactory
	}
	return injector
}
