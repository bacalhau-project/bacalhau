package node

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	filecoinlotus "github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus"
	"github.com/filecoin-project/bacalhau/pkg/simulator"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/imdario/mergo"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

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

	jobEventConsumer := eventhandler.NewChainedJobEventHandler(tracerContextProvider)

	var simulatorRequestHandler *simulator.RequestHandler
	if config.SimulatorNodeID == config.Host.ID().String() {
		log.Info().Msgf("Node %s is the simulator node. Setting proper event handlers", config.Host.ID().String())
		simulatorRequestHandler = simulator.NewRequestHandler()
	}

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
		jobEventConsumer,
	)
	if err != nil {
		return nil, err
	}

	// setup compute node
	computeNode := NewComputeNode(
		ctx,
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
	jobEventConsumer.AddHandlers(
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
	)

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
