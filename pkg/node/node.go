package node

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	filecoinlotus "github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus"
	"github.com/filecoin-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/simulator"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/imdario/mergo"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

const JobEventsTopic = "bacalhau-job-events"
const NodeInfoTopic = "bacalhau-node-info"

// Node configuration
type NodeConfig struct {
	IPFSClient           *ipfs.Client
	CleanupManager       *system.CleanupManager
	LocalDB              localdb.LocalDB
	Host                 host.Host
	FilecoinUnsealedPath string
	EstuaryAPIKey        string
	HostAddress          string
	APIPort              int
	MetricsPort          int
	ComputeConfig        ComputeConfig
	RequesterNodeConfig  RequesterConfig
	APIServerConfig      publicapi.APIServerConfig
	LotusConfig          *filecoinlotus.PublisherConfig
	SimulatorNodeID      string
	IsRequesterNode      bool
	IsComputeNode        bool
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
	CleanupManager *system.CleanupManager
	IPFSClient     *ipfs.Client
	Host           host.Host
	metricsPort    int
}

func (n *Node) Start(ctx context.Context) error {
	go func(ctx context.Context) {
		if err := n.APIServer.ListenAndServe(ctx, n.CleanupManager); err != nil {
			log.Ctx(ctx).Error().Msgf("Api server can't run. Cannot serve client requests!: %v", err)
		}
	}(ctx)

	go func(ctx context.Context) {
		if err := system.ListenAndServeMetrics(ctx, n.CleanupManager, n.metricsPort); err != nil {
			log.Ctx(ctx).Error().Msgf("Cannot serve metrics: %v", err)
		}
	}(ctx)

	return nil
}

func NewStandardNode(
	ctx context.Context,
	config NodeConfig) (*Node, error) {
	return NewNode(ctx, config, NewStandardNodeDependencyInjector())
}

//nolint:funlen
func NewNode(
	ctx context.Context,
	config NodeConfig,
	injector NodeDependencyInjector) (*Node, error) {
	err := mergo.Merge(&config.APIServerConfig, publicapi.DefaultAPIServerConfig)
	if err != nil {
		return nil, err
	}

	storageProviders, err := injector.StorageProvidersFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	executors, err := injector.ExecutorsFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	verifiers, err := injector.VerifiersFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	publishers, err := injector.PublishersFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	var simulatorRequestHandler *simulator.RequestHandler
	if config.SimulatorNodeID == config.Host.ID().String() {
		log.Info().Msgf("Node %s is the simulator node. Setting proper event handlers", config.Host.ID().String())
		simulatorRequestHandler = simulator.NewRequestHandler()
	}

	// public http api server
	apiServer, err := publicapi.NewAPIServer(publicapi.APIServerParams{
		Address: config.HostAddress,
		Port:    config.APIPort,
		Host:    config.Host,
		Config:  config.APIServerConfig,
	})
	if err != nil {
		return nil, err
	}

	// A single gossipSub instance that will be used by all topics
	gossipSubCtx, gossipSubCancel := context.WithCancel(ctx)
	gossipSub, err := newLibp2pPubSub(gossipSubCtx, config)
	if err != nil {
		gossipSubCancel()
		return nil, err
	}

	// PubSub to publish node info to the network
	nodeInfoPubSub, err := libp2p.NewPubSub[model.NodeInfo](libp2p.PubSubParams{
		Host:      config.Host,
		TopicName: NodeInfoTopic,
		PubSub:    gossipSub,
	})
	if err != nil {
		gossipSubCancel()
		return nil, err
	}

	var requesterNode *Requester
	var computeNode *Compute

	// setup requester node
	if config.IsRequesterNode {
		requesterNode, err = NewRequesterNode(
			ctx,
			config.CleanupManager,
			config.Host,
			apiServer,
			config.RequesterNodeConfig,
			config.LocalDB,
			config.SimulatorNodeID,
			simulatorRequestHandler,
			verifiers,
			storageProviders,
			nodeInfoPubSub,
			gossipSub,
		)
		if err != nil {
			gossipSubCancel()
			return nil, err
		}
	}

	if config.IsComputeNode {
		// setup compute node
		computeNode, err = NewComputeNode(
			ctx,
			config.CleanupManager,
			config.Host,
			apiServer,
			config.ComputeConfig,
			config.SimulatorNodeID,
			simulatorRequestHandler,
			executors,
			verifiers,
			publishers,
			nodeInfoPubSub,
		)
		if err != nil {
			gossipSubCancel()
			return nil, err
		}
	}

	// cleanup libp2p resources in the desired order
	config.CleanupManager.RegisterCallback(func() error {
		cleanupCtx := context.Background()
		if computeNode != nil {
			computeNode.cleanup(cleanupCtx)
		}
		if requesterNode != nil {
			requesterNode.cleanup(cleanupCtx)
		}
		cleanupErr := nodeInfoPubSub.Close(cleanupCtx)
		if cleanupErr != nil {
			log.Error().Err(cleanupErr).Msg("failed to close libp2p node info pubsub")
		}
		gossipSubCancel()

		cleanupErr = config.Host.Close()
		if cleanupErr != nil {
			log.Error().Err(cleanupErr).Msg("failed to close host")
		}
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
		Host:           config.Host,
		metricsPort:    config.MetricsPort,
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
