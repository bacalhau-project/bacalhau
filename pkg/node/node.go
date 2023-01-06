package node

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	filecoinlotus "github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus"
	"github.com/filecoin-project/bacalhau/pkg/pubsub"
	"github.com/filecoin-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/simulator"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/imdario/mergo"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

const JobEventsTopic = "bacalhau-job-events"

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
	ComputeNode    Compute
	RequesterNode  Requester
	LocalDB        localdb.LocalDB
	CleanupManager *system.CleanupManager
	Executors      executor.ExecutorProvider
	IPFSClient     *ipfs.Client

	Host        host.Host
	metricsPort int
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

	// prepare event handlers
	tracerContextProvider := system.NewTracerContextProvider(config.Host.ID().String())
	config.CleanupManager.RegisterCallback(tracerContextProvider.Shutdown)

	localJobEventConsumer := eventhandler.NewChainedJobEventHandler(tracerContextProvider)

	var simulatorRequestHandler *simulator.RequestHandler
	if config.SimulatorNodeID == config.Host.ID().String() {
		log.Info().Msgf("Node %s is the simulator node. Setting proper event handlers", config.Host.ID().String())
		simulatorRequestHandler = simulator.NewRequestHandler()
	}

	// A single gossipSub instance that will be used by all topics
	gossipSubCtx, gossipSubCancel := context.WithCancel(ctx)
	gossipSub, err := newLibp2pPubSub(gossipSubCtx, config)
	if err != nil {
		gossipSubCancel()
		return nil, err
	}
	// PubSub to publish job events to the network
	libp2p2JobEventPubSub := libp2p.NewPubSub[pubsub.BufferingEnvelope](libp2p.PubSubParams{
		Host:        config.Host,
		TopicName:   JobEventsTopic,
		PubSub:      gossipSub,
		IgnoreLocal: true,
	})

	bufferedJobEventPubSub := pubsub.NewBufferingPubSub[model.JobEvent](pubsub.BufferingPubSubParams{
		DelegatePubSub: libp2p2JobEventPubSub,
		MaxBufferSize:  32 * 1024,       //nolint:gomnd
		MaxBufferAge:   5 * time.Second, // increase this once we move to an external job storage
	})

	// cleanup libp2p resources in the desired order
	config.CleanupManager.RegisterCallback(func() error {
		cleanupContext := context.Background()
		cleanupErr := bufferedJobEventPubSub.Close(cleanupContext)
		if cleanupErr != nil {
			log.Error().Err(cleanupErr).Msg("failed to close job event pubsub")
		}
		cleanupErr = libp2p2JobEventPubSub.Close(cleanupContext)
		if cleanupErr != nil {
			log.Error().Err(cleanupErr).Msg("failed to close libp2p job event pubsub")
		}
		gossipSubCancel()

		cleanupErr = config.Host.Close()
		if cleanupErr != nil {
			log.Error().Err(cleanupErr).Msg("failed to close host")
		}
		return cleanupErr
	})

	requesterNode, err := NewRequesterNode(
		ctx,
		config.CleanupManager,
		config.Host,
		config.RequesterNodeConfig,
		config.LocalDB,
		config.SimulatorNodeID,
		simulatorRequestHandler,
		verifiers,
		storageProviders,
		localJobEventConsumer,
	)
	if err != nil {
		return nil, err
	}

	// setup compute node
	computeNode := NewComputeNode(
		ctx,
		config.CleanupManager,
		config.Host,
		config.ComputeConfig,
		config.SimulatorNodeID,
		simulatorRequestHandler,
		executors,
		verifiers,
		publishers,
	)

	// To enable nodes self-dialing themselves as libp2p doesn't support it.
	computeNode.RegisterLocalComputeCallback(requesterNode.localCallback)
	requesterNode.RegisterLocalComputeEndpoint(computeNode.LocalEndpoint)

	apiServer := publicapi.NewServerWithConfig(
		ctx,
		config.HostAddress,
		config.APIPort,
		config.LocalDB,
		config.Host,
		requesterNode.Endpoint,
		computeNode.debugInfoProviders,
		publishers,
		storageProviders,
		config.APIServerConfig,
	)

	eventTracer, err := eventhandler.NewTracer()
	if err != nil {
		return nil, err
	}
	config.CleanupManager.RegisterCallback(eventTracer.Shutdown)

	// Register event handlers
	lifecycleEventHandler := system.NewJobLifecycleEventHandler(config.Host.ID().String())
	localDBEventHandler := localdb.NewLocalDBEventHandler(config.LocalDB)

	// order of event handlers is important as triggering some handlers might depend on the state of others.
	localJobEventConsumer.AddHandlers(
		// add tracing metadata to the context about the read event
		eventhandler.JobEventHandlerFunc(lifecycleEventHandler.HandleConsumedJobEvent),
		// ends the span for the job if received a terminal event
		tracerContextProvider,
		// record the event in a log
		eventTracer,
		// update the job state in the local DB
		localDBEventHandler,
		// dispatches events to listening websockets
		apiServer,
		// dispatches events to the network
		eventhandler.JobEventHandlerFunc(bufferedJobEventPubSub.Publish),
	)

	// register consumers of job events publishes over gossipSub
	networkJobEventConsumer := eventhandler.NewChainedJobEventHandler(system.NewNoopContextProvider())
	networkJobEventConsumer.AddHandlers(
		// update the job state in the local DB
		localDBEventHandler,
	)
	err = bufferedJobEventPubSub.Subscribe(ctx, pubsub.SubscriberFunc[model.JobEvent](networkJobEventConsumer.HandleJobEvent))
	if err != nil {
		return nil, err
	}

	node := &Node{
		CleanupManager: config.CleanupManager,
		APIServer:      apiServer,
		IPFSClient:     config.IPFSClient,
		LocalDB:        config.LocalDB,
		ComputeNode:    *computeNode,
		RequesterNode:  *requesterNode,
		Executors:      executors,
		Host:           config.Host,
		metricsPort:    config.MetricsPort,
	}

	return node, nil
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

func newJobEventPubSub(ctx context.Context, nodeConfig NodeConfig, sub *libp2p_pubsub.PubSub) (pubsub.PubSub[model.JobEvent], error) {
	libp2p2PubSub := libp2p.NewPubSub[pubsub.BufferingEnvelope](libp2p.PubSubParams{
		Host:        nodeConfig.Host,
		TopicName:   JobEventsTopic,
		PubSub:      sub,
		IgnoreLocal: true,
	})

	bufferingPubSub := pubsub.NewBufferingPubSub[model.JobEvent](pubsub.BufferingPubSubParams{
		DelegatePubSub: libp2p2PubSub,
		MaxBufferSize:  32 * 1024,       //nolint:gomnd
		MaxBufferAge:   5 * time.Second, // increase this once we move to an external job storage
	})

	return bufferingPubSub, nil
}
