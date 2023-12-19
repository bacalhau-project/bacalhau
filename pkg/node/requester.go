package node

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/evaluation"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/planner"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/scheduler"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/selector"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	orchestrator_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/orchestrator"
	requester_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/requester"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/requester/pubsub/jobinfo"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/translation"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/discovery"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/ranking"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
)

type Requester struct {
	// Visible for testing
	Endpoint       requester.Endpoint
	JobStore       jobstore.Store
	NodeDiscoverer orchestrator.NodeDiscoverer
	computeProxy   *bprotocol.ComputeProxy
	localCallback  compute.Callback
	cleanupFunc    func(ctx context.Context)
}

//nolint:funlen
func NewRequesterNode(
	ctx context.Context,
	host host.Host,
	apiServer *publicapi.Server,
	requesterConfig RequesterConfig,
	storageProviders storage.StorageProvider,
	nodeInfoStore routing.NodeInfoStore,
	gossipSub *libp2p_pubsub.PubSub,
	fsRepo *repo.FsRepo,
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

	jobStore, err := fsRepo.InitJobStore(ctx, host.ID().String())
	if err != nil {
		return nil, err
	}

	// PubSub to publish job events to the network
	jobInfoPubSub, err := libp2p.NewPubSub[jobinfo.Envelope](libp2p.PubSubParams{
		Host:        host,
		TopicName:   JobInfoTopic,
		PubSub:      gossipSub,
		IgnoreLocal: true,
	})
	if err != nil {
		return nil, err
	}
	jobInfoPublisher := jobinfo.NewPublisher(jobinfo.PublisherParams{
		JobStore: jobStore,
		PubSub:   jobInfoPubSub,
	})
	err = jobInfoPubSub.Subscribe(ctx, pubsub.NewNoopSubscriber[jobinfo.Envelope]())
	if err != nil {
		return nil, err
	}

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
		ranking.NewMinVersionNodeRanker(ranking.MinVersionNodeRankerParams{MinVersion: requesterConfig.MinBacalhauVersion}),
		ranking.NewPreviousExecutionsNodeRanker(ranking.PreviousExecutionsNodeRankerParams{JobStore: jobStore}),
		// arbitrary rankers
		ranking.NewRandomNodeRanker(ranking.RandomNodeRankerParams{
			RandomnessRange: requesterConfig.NodeRankRandomnessRange,
		}),
	)

	// node selector
	nodeSelector := selector.NewNodeSelector(selector.NodeSelectorParams{
		NodeDiscoverer: nodeDiscoveryChain,
		NodeRanker:     nodeRankerChain,
	})

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

	retryStrategy := requesterConfig.RetryStrategy
	if retryStrategy == nil {
		// retry strategy
		retryStrategyChain := retry.NewChain()
		retryStrategyChain.Add(
			retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: true}),
		)
		retryStrategy = retryStrategyChain
	}

	// scheduler provider
	batchServiceJobScheduler := scheduler.NewBatchServiceJobScheduler(scheduler.BatchServiceJobSchedulerParams{
		JobStore:      jobStore,
		Planner:       planners,
		NodeSelector:  nodeSelector,
		RetryStrategy: retryStrategy,
	})
	schedulerProvider := orchestrator.NewMappedSchedulerProvider(map[string]orchestrator.Scheduler{
		models.JobTypeBatch:   batchServiceJobScheduler,
		models.JobTypeService: batchServiceJobScheduler,
		models.JobTypeOps: scheduler.NewOpsJobScheduler(scheduler.OpsJobSchedulerParams{
			JobStore:     jobStore,
			Planner:      planners,
			NodeSelector: nodeSelector,
		}),
		models.JobTypeDaemon: scheduler.NewDaemonJobScheduler(scheduler.DaemonJobSchedulerParams{
			JobStore:     jobStore,
			Planner:      planners,
			NodeSelector: nodeSelector,
		}),
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

	publicKey := host.Peerstore().PubKey(host.ID())
	marshaledPublicKey, err := crypto.MarshalPublicKey(publicKey)
	if err != nil {
		return nil, err
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
		ID:                         host.ID().String(),
		PublicKey:                  marshaledPublicKey,
		EvaluationBroker:           evalBroker,
		EventEmitter:               eventEmitter,
		ComputeEndpoint:            computeProxy,
		Store:                      jobStore,
		StorageProviders:           storageProviders,
		DefaultJobExecutionTimeout: requesterConfig.JobDefaults.ExecutionTimeout,
	})

	var translationProvider translation.TranslatorProvider
	if requesterConfig.TranslationEnabled {
		translationProvider = translation.NewStandardTranslatorsProvider()
	}

	endpointV2 := orchestrator.NewBaseEndpoint(&orchestrator.BaseEndpointParams{
		ID:               host.ID().String(),
		EvaluationBroker: evalBroker,
		Store:            jobStore,
		EventEmitter:     eventEmitter,
		ComputeProxy:     computeProxy,
		JobTransformer: transformer.ChainedTransformer[*models.Job]{
			transformer.JobFn(transformer.IDGenerator),
			transformer.NameOptional(),
			transformer.DefaultsApplier(requesterConfig.JobDefaults),
			transformer.RequesterInfo(host.ID().String(), marshaledPublicKey),
			transformer.NewInlineStoragePinner(storageProviders),
		},
		TaskTranslator:    translationProvider,
		ResultTransformer: resultTransformers,
	})

	housekeeping := requester.NewHousekeeping(requester.HousekeepingParams{
		Endpoint: endpoint,
		JobStore: jobStore,
		NodeID:   host.ID().String(),
		Interval: requesterConfig.HousekeepingBackgroundTaskInterval,
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
	requesterAPIServer := requester_endpoint.NewEndpoint(requester_endpoint.EndpointParams{
		Router:             apiServer.Router,
		Requester:          endpoint,
		DebugInfoProviders: debugInfoProviders,
		JobStore:           jobStore,
		NodeDiscoverer:     nodeDiscoveryChain,
	})

	orchestrator_endpoint.NewEndpoint(orchestrator_endpoint.EndpointParams{
		Router:       apiServer.Router,
		Orchestrator: endpointV2,
		JobStore:     jobStore,
		NodeStore:    nodeInfoStore,
	})

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

		cleanupErr := jobInfoPubSub.Close(ctx)
		if cleanupErr != nil {
			util.LogDebugIfContextCancelled(ctx, cleanupErr, "failed to shutdown job info pubsub")
		}

		cleanupErr = tracerContextProvider.Shutdown()
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
		Endpoint:       endpoint,
		localCallback:  endpoint,
		NodeDiscoverer: nodeDiscoveryChain,
		JobStore:       jobStore,
		computeProxy:   computeProxy,
		cleanupFunc:    cleanupFunc,
	}, nil
}

func (r *Requester) RegisterLocalComputeEndpoint(endpoint compute.Endpoint) {
	r.computeProxy.RegisterLocalComputeEndpoint(endpoint)
}

func (r *Requester) cleanup(ctx context.Context) {
	r.cleanupFunc(ctx)
}
