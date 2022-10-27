package node

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	filecoinlotus "github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus"

	computenode "github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/rs/zerolog/log"
)

// Node configuration
type NodeConfig struct {
	IPFSClient           *ipfs.Client
	CleanupManager       *system.CleanupManager
	LocalDB              localdb.LocalDB
	Transport            transport.Transport
	FilecoinUnsealedPath string
	EstuaryAPIKey        string
	HostAddress          string
	HostID               string
	APIPort              int
	MetricsPort          int
	IsBadActor           bool
	ComputeNodeConfig    computenode.ComputeNodeConfig
	RequesterNodeConfig  requesternode.RequesterNodeConfig
	LotusConfig          *filecoinlotus.PublisherConfig
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
	ComputeNode    *computenode.ComputeNode
	RequesterNode  *requesternode.RequesterNode
	LocalDB        localdb.LocalDB
	Transport      transport.Transport
	CleanupManager *system.CleanupManager
	Executors      executor.ExecutorProvider
	IPFSClient     *ipfs.Client

	HostID      string
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
	if config.HostID == "" {
		config.HostID = config.Transport.HostID()
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
	tracerContextProvider := system.NewTracerContextProvider(config.HostID)
	config.CleanupManager.RegisterCallback(tracerContextProvider.Shutdown)

	localEventConsumer := eventhandler.NewChainedLocalEventHandler(system.NewNoopContextProvider())
	jobEventConsumer := eventhandler.NewChainedJobEventHandler(tracerContextProvider)
	jobEventPublisher := eventhandler.NewChainedJobEventHandler(tracerContextProvider)

	requesterNode, err := requesternode.NewRequesterNode(
		ctx,
		config.HostID,
		config.LocalDB,
		localEventConsumer,
		jobEventPublisher,
		verifiers,
		storageProviders,
		config.RequesterNodeConfig,
	)
	if err != nil {
		return nil, err
	}
	computeNode, err := computenode.NewComputeNode(
		ctx,
		config.CleanupManager,
		config.HostID,
		config.LocalDB,
		localEventConsumer,
		jobEventPublisher,
		executors,
		verifiers,
		publishers,
		config.ComputeNodeConfig,
	)
	if err != nil {
		return nil, err
	}

	apiServer := publicapi.NewServer(
		ctx,
		config.HostAddress,
		config.APIPort,
		config.LocalDB,
		config.Transport,
		requesterNode,
		publishers,
		storageProviders,
	)

	// Register event handlers
	lifecycleEventHandler := system.NewJobLifecycleEventHandler(config.HostID)
	localDBEventHandler := localdb.NewLocalDBEventHandler(config.LocalDB)

	// order of event handlers is important as triggering some handlers might depend on the state of others.
	jobEventConsumer.AddHandlers(
		// add tracing metadata to the context about the read event
		eventhandler.JobEventHandlerFunc(lifecycleEventHandler.HandleConsumedJobEvent),
		// ends the span for the job if received a terminal event
		tracerContextProvider,
		// update the job state in the local DB
		localDBEventHandler,
		// handles bid and result proposals
		requesterNode,
		// handles job execution
		computeNode,
	)
	jobEventPublisher.AddHandlers(
		// publish events to the network
		eventhandler.JobEventHandlerFunc(config.Transport.Publish),
		// add tracing metadata to the context about the published event
		eventhandler.JobEventHandlerFunc(lifecycleEventHandler.HandlePublishedJobEvent),
	)
	localEventConsumer.AddHandlers(
		// update the job node state in the local DB
		localDBEventHandler,
	)

	// subscribe the job event handler to the transport
	config.Transport.Subscribe(ctx, jobEventConsumer.HandleJobEvent)

	node := &Node{
		CleanupManager: config.CleanupManager,
		APIServer:      apiServer,
		IPFSClient:     config.IPFSClient,
		LocalDB:        config.LocalDB,
		Transport:      config.Transport,
		ComputeNode:    computeNode,
		RequesterNode:  requesterNode,
		Executors:      executors,
		HostID:         config.HostID,
		metricsPort:    config.MetricsPort,
	}

	return node, nil
}
