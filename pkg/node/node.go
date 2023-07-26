package node

import (
	"context"
	"fmt"
	"time"

	"github.com/imdario/mergo"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/requester/pubsub/jobinfo"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

const JobInfoTopic = "bacalhau-job-info"
const NodeInfoTopic = "bacalhau-node-info"

type FeatureConfig struct {
	Engines    []model.Engine
	Publishers []model.Publisher
	Storages   []model.StorageSourceType
}

// Node configuration
type NodeConfig struct {
	IPFSClient                ipfs.Client
	CleanupManager            *system.CleanupManager
	JobStore                  jobstore.Store
	Host                      host.Host
	EstuaryAPIKey             string
	HostAddress               string
	APIPort                   uint16
	DisabledFeatures          FeatureConfig
	ComputeConfig             ComputeConfig
	RequesterNodeConfig       RequesterConfig
	APIServerConfig           publicapi.APIServerConfig
	IsRequesterNode           bool
	IsComputeNode             bool
	Labels                    map[string]string
	NodeInfoPublisherInterval routing.NodeInfoPublisherIntervalConfig
	DependencyInjector        NodeDependencyInjector
	AllowListedLocalPaths     []string
}

// Lazy node dependency injector that generate instances of different
// components on demand and based on the configuration provided.
type NodeDependencyInjector struct {
	StorageProvidersFactory StorageProvidersFactory
	ExecutorsFactory        ExecutorsFactory
	PublishersFactory       PublishersFactory
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

	publishers, err := config.DependencyInjector.PublishersFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	executors, err := config.DependencyInjector.ExecutorsFactory.Get(ctx, config, storageProviders)
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
	if nodeInfoPublisherInterval.IsZero() {
		nodeInfoPublisherInterval = GetNodeInfoPublishConfig()
	}

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

	// PubSub to publish job events to the network
	jobInfoPubSub, err := libp2p.NewPubSub[jobinfo.Envelope](libp2p.PubSubParams{
		Host:        config.Host,
		TopicName:   JobInfoTopic,
		PubSub:      gossipSub,
		IgnoreLocal: true,
	})
	if err != nil {
		return nil, err
	}
	jobInfoPublisher := jobinfo.NewPublisher(jobinfo.PublisherParams{
		JobStore: config.JobStore,
		PubSub:   jobInfoPubSub,
	})
	err = jobInfoPubSub.Subscribe(ctx, pubsub.NewNoopSubscriber[jobinfo.Envelope]())
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
			storageProviders,
			jobInfoPublisher,
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
			storageProviders,
			executors,
			publishers,
		)
		if err != nil {
			return nil, err
		}
		nodeInfoProvider.RegisterComputeInfoProvider(computeNode.computeInfoProvider)
	}

	// NB(forrest): this must be done last to avoid eager publishing before nodes are constructed
	// TODO(forrest) [fixme] we should fix this to make it less racy in testing
	nodeInfoPublisher := routing.NewNodeInfoPublisher(routing.NodeInfoPublisherParams{
		PubSub:           nodeInfoPubSub,
		NodeInfoProvider: nodeInfoProvider,
		IntervalConfig:   nodeInfoPublisherInterval,
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
		cleanupErr = jobInfoPubSub.Close(ctx)
		util.LogDebugIfContextCancelled(ctx, cleanupErr, "job info pub sub")
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
	if injector.PublishersFactory == nil {
		injector.PublishersFactory = defaultInjector.PublishersFactory
	}
	return injector
}
