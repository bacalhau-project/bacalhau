package node

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/backend"
	"github.com/filecoin-project/bacalhau/pkg/compute/bidstrategy"
	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/compute/capacity/disk"
	"github.com/filecoin-project/bacalhau/pkg/compute/frontend"
	"github.com/filecoin-project/bacalhau/pkg/compute/pubsub"
	"github.com/filecoin-project/bacalhau/pkg/compute/sensors"
	"github.com/filecoin-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	filecoinlotus "github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus"
	"github.com/filecoin-project/bacalhau/pkg/verifier"

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
	ComputeConfig        ComputeConfig
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
	ComputeNode    frontend.Service
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
		config.CleanupManager,
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

	// setup compute node
	computeService, computeListener, debugInfoProviders := generateComputeNode(
		ctx,
		config,
		executors,
		verifiers,
		publishers,
		jobEventPublisher,
	)

	apiServer := publicapi.NewServer(
		ctx,
		config.HostAddress,
		config.APIPort,
		config.LocalDB,
		config.Transport,
		requesterNode,
		debugInfoProviders,
		publishers,
		storageProviders,
	)

	eventTracer, err := eventhandler.NewTracer()
	if err != nil {
		return nil, err
	}
	config.CleanupManager.RegisterCallback(eventTracer.Shutdown)

	// Register event handlers
	lifecycleEventHandler := system.NewJobLifecycleEventHandler(config.HostID)
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
		// handles bid and result proposals
		requesterNode,
		// handles job execution
		computeListener,
		// dispatches events to listening websockets
		apiServer,
	)
	jobEventPublisher.AddHandlers(
		// publish events to the network
		eventhandler.JobEventHandlerFunc(config.Transport.Publish),
		// record the event in a log
		eventTracer,
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
		ComputeNode:    computeService,
		RequesterNode:  requesterNode,
		Executors:      executors,
		HostID:         config.HostID,
		metricsPort:    config.MetricsPort,
	}

	return node, nil
}

//nolint:funlen
func generateComputeNode(
	ctx context.Context,
	nodeConfig NodeConfig,
	executors executor.ExecutorProvider,
	verifiers verifier.VerifierProvider,
	publishers publisher.PublisherProvider,
	jobEventPublisher eventhandler.JobEventHandler) (frontend.Service, *pubsub.FrontendEventProxy, []model.DebugInfoProvider) {
	debugInfoProviders := []model.DebugInfoProvider{}
	executionStore := inmemory.NewStore()

	// backend
	capacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: nodeConfig.ComputeConfig.TotalResourceLimits,
	})
	debugInfoProviders = append(debugInfoProviders, sensors.NewCapacityInfoProvider(sensors.CapacityInfoProviderParams{
		CapacityTracker: capacityTracker,
	}))

	backendCallback := backend.NewChainedCallback(backend.ChainedCallbackParams{
		Callbacks: []backend.Callback{
			backend.NewStateUpdateCallback(backend.StateUpdateCallbackParams{
				ExecutionStore: executionStore,
			}),
			pubsub.NewBackendCallback(pubsub.BackendCallbackParams{
				NodeID:            nodeConfig.HostID,
				ExecutionStore:    executionStore,
				JobEventPublisher: jobEventPublisher,
			}),
		},
	})

	baseRunner := backend.NewBaseService(backend.BaseServiceParams{
		ID:         nodeConfig.HostID,
		Callback:   backendCallback,
		Store:      executionStore,
		Executors:  executors,
		Verifiers:  verifiers,
		Publishers: publishers,
	})

	bufferRunner := backend.NewServiceBuffer(backend.ServiceBufferParams{
		DelegateService:            baseRunner,
		Callback:                   backendCallback,
		RunningCapacityTracker:     capacityTracker,
		DefaultJobExecutionTimeout: nodeConfig.ComputeConfig.DefaultJobExecutionTimeout,
		BackoffDuration:            50 * time.Millisecond,
	})
	runningInfoProvider := sensors.NewRunningInfoProvider(sensors.RunningInfoProviderParams{
		BackendBuffer: bufferRunner,
	})
	debugInfoProviders = append(debugInfoProviders, runningInfoProvider)
	loggingSensor := sensors.NewLoggingSensor(sensors.LoggingSensorParams{
		InfoProvider: runningInfoProvider,
		Interval:     nodeConfig.ComputeConfig.LogRunningExecutionsInterval,
	})
	go loggingSensor.Start(ctx)

	// frontend
	capacityCalculator := capacity.NewChainedUsageCalculator(capacity.ChainedUsageCalculatorParams{
		Calculators: []capacity.UsageCalculator{
			capacity.NewDefaultsUsageCalculator(capacity.DefaultsUsageCalculatorParams{
				Defaults: nodeConfig.ComputeConfig.DefaultJobResourceLimits,
			}),
			disk.NewDiskUsageCalculator(disk.DiskUsageCalculatorParams{
				Executors: executors,
			}),
		},
	})

	biddingStrategy := bidstrategy.NewChainedBidStrategy(
		bidstrategy.NewMaxCapacityStrategy(bidstrategy.MaxCapacityStrategyParams{
			MaxJobRequirements: nodeConfig.ComputeConfig.JobResourceLimits,
		}),
		bidstrategy.NewAvailableCapacityStrategy(bidstrategy.AvailableCapacityStrategyParams{
			CapacityTracker: capacityTracker,
			CommitFactor:    nodeConfig.ComputeConfig.OverCommitResourcesFactor,
		}),
		// TODO XXX: don't hardcode networkSize, calculate this dynamically from
		//  libp2p instead somehow. https://github.com/filecoin-project/bacalhau/issues/512
		bidstrategy.NewDistanceDelayStrategy(bidstrategy.DistanceDelayStrategyParams{
			NetworkSize: 1,
		}),
		bidstrategy.NewEnginesInstalledStrategy(bidstrategy.EnginesInstalledStrategyParams{
			Executors: executors,
			Verifiers: verifiers,
		}),
		bidstrategy.NewExternalCommandStrategy(bidstrategy.ExternalCommandStrategyParams{
			Command: nodeConfig.ComputeConfig.JobSelectionPolicy.ProbeExec,
		}),
		bidstrategy.NewExternalHTTPStrategy(bidstrategy.ExternalHTTPStrategyParams{
			URL: nodeConfig.ComputeConfig.JobSelectionPolicy.ProbeHTTP,
		}),
		bidstrategy.NewInputLocalityStrategy(bidstrategy.InputLocalityStrategyParams{
			Locality:  nodeConfig.ComputeConfig.JobSelectionPolicy.Locality,
			Executors: executors,
		}),
		bidstrategy.NewStatelessJobStrategy(bidstrategy.StatelessJobStrategyParams{
			RejectStatelessJobs: nodeConfig.ComputeConfig.JobSelectionPolicy.RejectStatelessJobs,
		}),
		bidstrategy.NewTimeoutStrategy(bidstrategy.TimeoutStrategyParams{
			MaxJobExecutionTimeout: nodeConfig.ComputeConfig.MaxJobExecutionTimeout,
			MinJobExecutionTimeout: nodeConfig.ComputeConfig.MinJobExecutionTimeout,
		}),
	)

	frontendNode := frontend.NewBaseService(frontend.BaseServiceParams{
		ID:              nodeConfig.HostID,
		ExecutionStore:  executionStore,
		UsageCalculator: capacityCalculator,
		BidStrategy:     biddingStrategy,
		Backend:         bufferRunner,
	})

	frontendProxy := pubsub.NewFrontendEventProxy(pubsub.FrontendEventProxyParams{
		NodeID:            nodeConfig.HostID,
		Frontend:          frontendNode,
		JobStore:          nodeConfig.LocalDB,
		ExecutionStore:    executionStore,
		JobEventPublisher: jobEventPublisher,
	})

	return frontendNode, frontendProxy, debugInfoProviders
}
