package node

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node/manager"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/evaluation"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/planner"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/scheduler"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/selector"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	auth_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/auth"
	orchestrator_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/orchestrator"
	requester_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/requester"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/translation"
	"github.com/bacalhau-project/bacalhau/pkg/util"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/discovery"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/ranking"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type Requester struct {
	// Visible for testing
	Endpoint   requester.Endpoint
	EndpointV2 *orchestrator.BaseEndpoint
	JobStore   jobstore.Store
	// We need a reference to the node info store until libp2p is removed
	NodeInfoStore      routing.NodeInfoStore
	NodeDiscoverer     orchestrator.NodeDiscoverer
	nodeManager        *manager.NodeManager
	localCallback      compute.Callback
	cleanupFunc        func(ctx context.Context)
	debugInfoProviders []model.DebugInfoProvider
}

//nolint:funlen
func NewRequesterNode(
	ctx context.Context,
	nodeID string,
	apiServer *publicapi.Server,
	requesterConfig RequesterConfig,
	storageProvider storage.StorageProvider,
	authnProvider authn.Provider,
	nodeInfoStore routing.NodeInfoStore, // for libp2p store only, once removed remove this in favour of nodeManager
	computeProxy compute.Endpoint,
	nodeManager *manager.NodeManager,
) (*Requester, error) {
	// prepare event handlers
	tracerContextProvider := eventhandler.NewTracerContextProvider(nodeID)
	localJobEventConsumer := eventhandler.NewChainedJobEventHandler(tracerContextProvider)

	eventEmitter := orchestrator.NewEventEmitter(orchestrator.EventEmitterParams{
		EventConsumer: localJobEventConsumer,
	})

	jobStore := requesterConfig.JobStore

	// TODO(forrest) [simplify]: given the current state of the code this interface obfuscates what is happening here,
	// there isn't any "node discovery" happening here, we are simply listing a node store.
	// The todo here is to simply pass a node store where it's needed instead of this chain wrapping a discoverer wrapping
	// a store...
	// compute node discoverer
	log.Ctx(ctx).
		Info().
		Msgf("Nodes joining the cluster will be assigned approval state: %s", requesterConfig.DefaultApprovalState.String())

	// compute node ranker
	nodeRankerChain := ranking.NewChain()
	nodeRankerChain.Add(
		// rankers that act as filters and give a -1 score to nodes that do not match the filter
		ranking.NewEnginesNodeRanker(),
		ranking.NewPublishersNodeRanker(),
		ranking.NewStoragesNodeRanker(),
		ranking.NewLabelsNodeRanker(),
		ranking.NewMaxUsageNodeRanker(),
		ranking.NewMinVersionNodeRanker(ranking.MinVersionNodeRankerParams{MinVersion: requesterConfig.MinBacalhauVersion}),
		ranking.NewPreviousExecutionsNodeRanker(ranking.PreviousExecutionsNodeRankerParams{JobStore: jobStore}),
		// arbitrary rankers
		ranking.NewRandomNodeRanker(ranking.RandomNodeRankerParams{
			RandomnessRange: requesterConfig.NodeRankRandomnessRange,
		}),
	)

	// evaluation broker
	evalBroker, err := evaluation.NewInMemoryBroker(evaluation.InMemoryBrokerParams{
		VisibilityTimeout:    requesterConfig.EvalBrokerVisibilityTimeout,
		InitialRetryDelay:    requesterConfig.EvalBrokerInitialRetryDelay,
		SubsequentRetryDelay: requesterConfig.EvalBrokerSubsequentRetryDelay,
		MaxReceiveCount:      requesterConfig.EvalBrokerMaxRetryCount,
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
			ID:             nodeID,
			ComputeService: computeProxy,
			JobStore:       jobStore,
		}),

		// planner that publishes events on job completion or failure
		planner.NewEventEmitter(planner.EventEmitterParams{
			ID:           nodeID,
			EventEmitter: eventEmitter,
		}),

		// logs job completion or failure
		planner.NewLoggingPlanner(),
	)

	retryStrategy := requesterConfig.RetryStrategy
	if retryStrategy == nil {
		// retry strategy
		retryStrategyChain := retry.NewChain()
		retryStrategyChain.Add(
			retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: true}),
		)
		retryStrategy = retryStrategyChain
	}

	// TODO(forrest): [refactor] the selector constraints ought to be a parameter to the node selector.
	// node selector
	nodeSelector := selector.NewNodeSelector(selector.NodeSelectorParams{
		NodeDiscoverer: nodeInfoStore,
		NodeRanker:     nodeRankerChain,
	})
	// selector constraints: require nodes be online and approved to schedule
	selectorConstraints := orchestrator.NodeSelectionConstraints{
		RequireConnected: true,
		RequireApproval:  true,
	}

	// scheduler provider
	batchServiceJobScheduler := scheduler.NewBatchServiceJobScheduler(
		jobStore,
		planners,
		nodeSelector,
		retryStrategy,
		selectorConstraints,
	)
	schedulerProvider := orchestrator.NewMappedSchedulerProvider(map[string]orchestrator.Scheduler{
		models.JobTypeBatch:   batchServiceJobScheduler,
		models.JobTypeService: batchServiceJobScheduler,
		models.JobTypeOps: scheduler.NewOpsJobScheduler(
			jobStore,
			planners,
			nodeSelector,
			selectorConstraints,
		),
		models.JobTypeDaemon: scheduler.NewDaemonJobScheduler(
			jobStore,
			planners,
			nodeSelector,
			selectorConstraints,
		),
	})

	workers := make([]*orchestrator.Worker, 0, requesterConfig.WorkerCount)
	for i := 1; i <= requesterConfig.WorkerCount; i++ {
		log.Debug().Msgf("Starting worker %d", i)
		// worker config the polls from the broker
		worker := orchestrator.NewWorker(orchestrator.WorkerParams{
			SchedulerProvider:     schedulerProvider,
			EvaluationBroker:      evalBroker,
			DequeueTimeout:        requesterConfig.WorkerEvalDequeueTimeout,
			DequeueFailureBackoff: backoff.NewExponential(requesterConfig.WorkerEvalDequeueBaseBackoff, requesterConfig.WorkerEvalDequeueMaxBackoff),
		})
		workers = append(workers, worker)
		worker.Start(ctx)
	}

	// result transformers that are applied to the result before it is returned to the user
	resultTransformers := transformer.ChainedTransformer[*models.SpecConfig]{}

	if !requesterConfig.S3PreSignedURLDisabled {
		// S3 result signer
		s3Config, err := s3helper.DefaultAWSConfig()
		if err != nil {
			return nil, err
		}
		resultSigner := s3helper.NewResultSigner(s3helper.ResultSignerParams{
			ClientProvider: s3helper.NewClientProvider(s3helper.ClientProviderParams{
				AWSConfig: s3Config,
			}),
			Expiration: requesterConfig.S3PreSignedURLExpiration,
		})
		resultTransformers = append(resultTransformers, resultSigner)
	}

	endpoint := requester.NewBaseEndpoint(&requester.BaseEndpointParams{
		ID:                         nodeID,
		EvaluationBroker:           evalBroker,
		EventEmitter:               eventEmitter,
		ComputeEndpoint:            computeProxy,
		Store:                      jobStore,
		StorageProviders:           storageProvider,
		DefaultJobExecutionTimeout: requesterConfig.JobDefaults.ExecutionTimeout,
		DefaultPublisher:           requesterConfig.DefaultPublisher,
	})

	var translationProvider translation.TranslatorProvider
	if requesterConfig.TranslationEnabled {
		translationProvider = translation.NewStandardTranslatorsProvider()
	}

	jobTransformers := transformer.ChainedTransformer[*models.Job]{
		transformer.JobFn(transformer.IDGenerator),
		transformer.NameOptional(),
		transformer.DefaultsApplier(requesterConfig.JobDefaults),
		transformer.RequesterInfo(nodeID),
		transformer.NewInlineStoragePinner(storageProvider),
	}

	if requesterConfig.DefaultPublisher != "" {
		// parse the publisher to generate a models.SpecConfig and add it to each job
		// which is without a publisher
		config, err := job.ParsePublisherString(requesterConfig.DefaultPublisher)
		if err == nil {
			jobTransformers = append(jobTransformers, transformer.DefaultPublisher(config))
		}
	}

	endpointV2 := orchestrator.NewBaseEndpoint(&orchestrator.BaseEndpointParams{
		ID:                nodeID,
		EvaluationBroker:  evalBroker,
		Store:             jobStore,
		EventEmitter:      eventEmitter,
		ComputeProxy:      computeProxy,
		JobTransformer:    jobTransformers,
		TaskTranslator:    translationProvider,
		ResultTransformer: resultTransformers,
	})

	housekeeping := requester.NewHousekeeping(requester.HousekeepingParams{
		Endpoint: endpoint,
		JobStore: jobStore,
		NodeID:   nodeID,
		Interval: requesterConfig.HousekeepingBackgroundTaskInterval,
	})

	// register debug info providers for the /debug endpoint
	debugInfoProviders := []model.DebugInfoProvider{
		discovery.NewDebugInfoProvider(nodeInfoStore),
	}

	// register requester public http apis
	requesterAPIServer := requester_endpoint.NewEndpoint(requester_endpoint.EndpointParams{
		Router:             apiServer.Router,
		Requester:          endpoint,
		DebugInfoProviders: debugInfoProviders,
		JobStore:           jobStore,
		NodeDiscoverer:     nodeInfoStore,
	})

	orchestrator_endpoint.NewEndpoint(orchestrator_endpoint.EndpointParams{
		Router:       apiServer.Router,
		Orchestrator: endpointV2,
		JobStore:     jobStore,
		NodeManager:  nodeManager,
	})

	auth_endpoint.BindEndpoint(ctx, apiServer.Router, authnProvider)

	// Register event handlers
	lifecycleEventHandler := system.NewJobLifecycleEventHandler(nodeID)
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
	)

	// A single Cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		// stop the housekeeping background task
		housekeeping.Stop()
		for _, worker := range workers {
			worker.Stop()
		}
		evalBroker.SetEnabled(false)

		cleanupErr := tracerContextProvider.Shutdown()
		if cleanupErr != nil {
			util.LogDebugIfContextCancelled(ctx, cleanupErr, "failed to shutdown tracer context provider")
		}
		cleanupErr = eventTracer.Shutdown()
		if cleanupErr != nil {
			util.LogDebugIfContextCancelled(ctx, cleanupErr, "failed to shutdown event tracer")
		}

		// Close the jobstore after the evaluation broker is disabled
		cleanupErr = jobStore.Close(ctx)
		if cleanupErr != nil {
			util.LogDebugIfContextCancelled(ctx, cleanupErr, "failed to cleanly shutdown jobstore")
		}
	}

	return &Requester{
		Endpoint:           endpoint,
		localCallback:      endpoint,
		EndpointV2:         endpointV2,
		NodeDiscoverer:     nodeInfoStore,
		NodeInfoStore:      nodeInfoStore,
		JobStore:           jobStore,
		nodeManager:        nodeManager,
		cleanupFunc:        cleanupFunc,
		debugInfoProviders: debugInfoProviders,
	}, nil
}

func (r *Requester) cleanup(ctx context.Context) {
	r.cleanupFunc(ctx)
}
