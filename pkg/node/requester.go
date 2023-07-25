package node

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/evaluation"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/planner"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/scheduler"
	"github.com/bacalhau-project/bacalhau/pkg/requester/pubsub/jobinfo"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/discovery"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/ranking"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	requester_publicapi "github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
)

type Requester struct {
	// Visible for testing
	Endpoint           requester.Endpoint
	JobStore           jobstore.Store
	NodeDiscoverer     orchestrator.NodeDiscoverer
	computeProxy       *bprotocol.ComputeProxy
	localCallback      compute.Callback
	requesterAPIServer *requester_publicapi.RequesterAPIServer
	cleanupFunc        func(ctx context.Context)
}

//nolint:funlen
func NewRequesterNode(
	ctx context.Context,
	cleanupManager *system.CleanupManager,
	host host.Host,
	apiServer *publicapi.APIServer,
	config RequesterConfig,
	jobStore jobstore.Store,
	storageProviders storage.StorageProvider,
	jobInfoPublisher *jobinfo.Publisher,
	nodeInfoStore routing.NodeInfoStore,
) (*Requester, error) {
	// prepare event handlers
	tracerContextProvider := eventhandler.NewTracerContextProvider(host.ID().String())
	localJobEventConsumer := eventhandler.NewChainedJobEventHandler(tracerContextProvider)

	// compute proxy
	computeProxy := bprotocol.NewComputeProxy(bprotocol.ComputeProxyParams{
		Host: host,
	})

	eventEmitter := orchestrator.NewEventEmitter(orchestrator.EventEmitterParams{
		EventConsumer: localJobEventConsumer,
	})

	// compute node discoverer
	nodeDiscoveryChain := discovery.NewChain(true)
	nodeDiscoveryChain.Add(
		discovery.NewStoreNodeDiscoverer(discovery.StoreNodeDiscovererParams{
			Store: nodeInfoStore,
		}),
	)

	// compute node ranker
	nodeRankerChain := ranking.NewChain()
	nodeRankerChain.Add(
		// rankers that act as filters and give a -1 score to nodes that do not match the filter
		ranking.NewEnginesNodeRanker(),
		ranking.NewPublishersNodeRanker(),
		ranking.NewStoragesNodeRanker(),
		ranking.NewLabelsNodeRanker(),
		ranking.NewMaxUsageNodeRanker(),
		ranking.NewMinVersionNodeRanker(ranking.MinVersionNodeRankerParams{MinVersion: config.MinBacalhauVersion}),
		ranking.NewPreviousExecutionsNodeRanker(ranking.PreviousExecutionsNodeRankerParams{JobStore: jobStore}),
		// arbitrary rankers
		ranking.NewRandomNodeRanker(ranking.RandomNodeRankerParams{
			RandomnessRange: config.NodeRankRandomnessRange,
		}),
	)

	// evaluation broker
	evalBroker, err := evaluation.NewInMemoryBroker(evaluation.InMemoryBrokerParams{
		VisibilityTimeout:    config.EvalBrokerVisibilityTimeout,
		InitialRetryDelay:    config.EvalBrokerInitialRetryDelay,
		SubsequentRetryDelay: config.EvalBrokerSubsequentRetryDelay,
		MaxReceiveCount:      config.EvalBrokerMaxRetryCount,
	})
	if err != nil {
		return nil, err
	}
	evalBroker.SetEnabled(true)

	// planners that execute the proposed plan by the scheduler
	// order of the planners is important as they are executed in order
	planners := planner.NewChain(
		// planner that persist the desired state as defined by the scheduler
		planner.NewStateUpdater(jobStore),

		// planner that forwards the desired state to the compute nodes,
		// and updates the observed state if the compute node accepts the desired state
		planner.NewComputeForwarder(planner.ComputeForwarderParams{
			ID:             host.ID().String(),
			ComputeService: computeProxy,
			JobStore:       jobStore,
		}),

		// planner that publishes events on job completion or failure
		planner.NewEventEmitter(planner.EventEmitterParams{
			ID:           host.ID().String(),
			EventEmitter: eventEmitter,
		}),

		// logs job completion or failure
		planner.NewLoggingPlanner(),
	)

	retryStrategy := config.RetryStrategy
	if retryStrategy == nil {
		// retry strategy
		retryStrategyChain := retry.NewChain()
		retryStrategyChain.Add(
			retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: true}),
		)
		retryStrategy = retryStrategyChain
	}

	// scheduler provider
	schedulerProvider := orchestrator.NewMappedSchedulerProvider(map[string]orchestrator.Scheduler{
		model.JobTypeBatch: scheduler.NewBatchJobScheduler(scheduler.BatchJobSchedulerParams{
			JobStore:       jobStore,
			Planner:        planners,
			NodeDiscoverer: nodeDiscoveryChain,
			NodeRanker:     nodeRankerChain,
			RetryStrategy:  retryStrategy,
		}),
	})

	workers := make([]*orchestrator.Worker, 0, config.WorkerCount)
	for i := 1; i <= config.WorkerCount; i++ {
		log.Debug().Msgf("Starting worker %d", i)
		// worker config the polls from the broker
		worker := orchestrator.NewWorker(orchestrator.WorkerParams{
			SchedulerProvider:     schedulerProvider,
			EvaluationBroker:      evalBroker,
			DequeueTimeout:        config.WorkerEvalDequeueTimeout,
			DequeueFailureBackoff: backoff.NewExponential(config.WorkerEvalDequeueBaseBackoff, config.WorkerEvalDequeueMaxBackoff),
		})
		workers = append(workers, worker)
		worker.Start(ctx)
	}

	publicKey := host.Peerstore().PubKey(host.ID())
	marshaledPublicKey, err := crypto.MarshalPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	endpoint := requester.NewBaseEndpoint(&requester.BaseEndpointParams{
		ID:                         host.ID().String(),
		PublicKey:                  marshaledPublicKey,
		EvaluationBroker:           evalBroker,
		EventEmitter:               eventEmitter,
		ComputeEndpoint:            computeProxy,
		Store:                      jobStore,
		StorageProviders:           storageProviders,
		MinJobExecutionTimeout:     config.MinJobExecutionTimeout,
		DefaultJobExecutionTimeout: config.DefaultJobExecutionTimeout,
	})

	housekeeping := requester.NewHousekeeping(requester.HousekeepingParams{
		Endpoint: endpoint,
		JobStore: jobStore,
		NodeID:   host.ID().String(),
		Interval: config.HousekeepingBackgroundTaskInterval,
	})

	// register a handler for the bacalhau protocol handler that will forward requests to the scheduler
	bprotocol.NewCallbackHandler(bprotocol.CallbackHandlerParams{
		Host:     host,
		Callback: endpoint,
	})

	// register debug info providers for the /debug endpoint
	debugInfoProviders := []model.DebugInfoProvider{
		discovery.NewDebugInfoProvider(nodeDiscoveryChain),
	}

	// register requester public http apis
	requesterAPIServer := requester_publicapi.NewRequesterAPIServer(requester_publicapi.RequesterAPIServerParams{
		APIServer:          apiServer,
		Requester:          endpoint,
		DebugInfoProviders: debugInfoProviders,
		JobStore:           jobStore,
		NodeDiscoverer:     nodeDiscoveryChain,
	})
	err = requesterAPIServer.RegisterAllHandlers()
	if err != nil {
		return nil, err
	}

	// Register event handlers
	lifecycleEventHandler := system.NewJobLifecycleEventHandler(host.ID().String())
	eventTracer, err := eventhandler.NewTracer()
	if err != nil {
		return nil, err
	}

	// order of event handlers is important as triggering some handlers might depend on the state of others.
	localJobEventConsumer.AddHandlers(
		// add tracing metadata to the context about the read event
		eventhandler.JobEventHandlerFunc(lifecycleEventHandler.HandleConsumedJobEvent),
		// ends the span for the job if received a terminal event
		tracerContextProvider,
		// record the event in a log
		eventTracer,
		// dispatches events to listening websockets
		requesterAPIServer,
		// publish job events to the network
		jobInfoPublisher,
	)

	// A single cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		// stop the housekeeping background task
		housekeeping.Stop()
		for _, worker := range workers {
			worker.Stop()
		}
		evalBroker.SetEnabled(false)

		cleanupErr := tracerContextProvider.Shutdown()
		if cleanupErr != nil {
			log.Ctx(ctx).Error().Err(cleanupErr).Msg("failed to shutdown tracer context provider")
		}
		cleanupErr = eventTracer.Shutdown()
		if cleanupErr != nil {
			log.Ctx(ctx).Error().Err(cleanupErr).Msg("failed to shutdown event tracer")
		}
	}

	return &Requester{
		Endpoint:           endpoint,
		localCallback:      endpoint,
		NodeDiscoverer:     nodeDiscoveryChain,
		JobStore:           jobStore,
		computeProxy:       computeProxy,
		cleanupFunc:        cleanupFunc,
		requesterAPIServer: requesterAPIServer,
	}, nil
}

func (r *Requester) RegisterLocalComputeEndpoint(endpoint compute.Endpoint) {
	r.computeProxy.RegisterLocalComputeEndpoint(endpoint)
}

func (r *Requester) cleanup(ctx context.Context) {
	r.cleanupFunc(ctx)
}
