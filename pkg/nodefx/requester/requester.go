package requester

import (
	"context"

	"github.com/labstack/echo/v4"
	pkgerrors "github.com/pkg/errors"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/node/manager"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/evaluation"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/planner"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/scheduler"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/discovery"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/ranking"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/selector"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
	orchestrator_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/orchestrator"
	requester_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/requester"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/kvstore"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/translation"
)

// Module contains all dependencies required for a requester node.
/* TODO: this can be further modularized into the parts that make up a requester node, such as:
- Control Plane(?)
	- Scheduler
	- Orchestrator
	- Planner
- API
- Node Manager
*/
var Module = fx.Module("requester",
	fx.Provide(NewRequesterNode),
	fx.Provide(JobStore),
	fx.Provide(NodeStore),
	fx.Provide(NodeManager),
	fx.Provide(TracerContextProvider),
	fx.Provide(JobEventHandler),
	fx.Provide(EventEmitter),
	fx.Provide(EventTracer),
	fx.Provide(NodeDiscoverer),
	fx.Provide(NodeRanker),
	fx.Provide(NodeSelector),
	fx.Provide(EvaluationBroker),
	fx.Provide(Planner),
	fx.Provide(RetryStrategy),
	fx.Provide(SchedulerProvider),
	fx.Provide(
		fx.Annotate(
			EndpointV1,
			fx.As(new(requester.Endpoint)),
			fx.As(new(compute.Callback)),
		),
	),
	fx.Provide(EndpointV2),
	fx.Provide(Housekeeping),
	fx.Provide(RequesterAPI),
	fx.Provide(
		fx.Annotate(
			RequesterDebugInfoProviders,
			fx.ResultTags(`name:"requester_debug_providers"`),
		),
	),

	fx.Invoke(OrchestratorAPI),
	fx.Invoke(RegisterEventConsumerHandlers),
	fx.Invoke(RegisterTransportComputeCallback),
	fx.Invoke(RegisterTransportNodeManager),
	fx.Invoke(PopulateNodeManagerStore),
)

type RequesterNode struct {
	Endpoint        requester.Endpoint
	ComputeCallback compute.Callback
	EndpointV2      *orchestrator.BaseEndpoint
	JobStore        jobstore.Store
	NodeDiscoverer  orchestrator.NodeDiscoverer
	NodeManager     *manager.NodeManager
	Scheduler       orchestrator.SchedulerProvider
}

type RequesterParams struct {
	fx.In
	// Visible for testing
	Endpoint        requester.Endpoint
	ComputeCallback compute.Callback
	EndpointV2      *orchestrator.BaseEndpoint
	JobStore        jobstore.Store
	// We need a reference to the node info store until libp2p is removed
	NodeDiscoverer orchestrator.NodeDiscoverer
	NodeManager    *manager.NodeManager
	Scheduler      orchestrator.SchedulerProvider
}

func NewRequesterNode(p RequesterParams) *RequesterNode {
	return &RequesterNode{
		Endpoint:        p.Endpoint,
		ComputeCallback: p.ComputeCallback,
		EndpointV2:      p.EndpointV2,
		JobStore:        p.JobStore,
		NodeDiscoverer:  p.NodeDiscoverer,
		NodeManager:     p.NodeManager,
		Scheduler:       p.Scheduler,
	}
}

func TracerContextProvider(lc fx.Lifecycle, cfg node.RequesterConfig) (*eventhandler.TracerContextProvider, error) {
	provider := eventhandler.NewTracerContextProvider(cfg.NodeID)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return provider.Shutdown()
		},
	})
	return provider, nil
}

func JobEventHandler(contextProvider *eventhandler.TracerContextProvider) (*eventhandler.ChainedJobEventHandler, error) {
	return eventhandler.NewChainedJobEventHandler(contextProvider), nil
}

func EventEmitter(consumer *eventhandler.ChainedJobEventHandler) (orchestrator.EventEmitter, error) {
	return orchestrator.NewEventEmitter(orchestrator.EventEmitterParams{
		EventConsumer: consumer,
	}), nil
}

func NodeStore(transport *nats_transport.NATSTransport) (*kvstore.NodeStore, error) {
	ctx := context.TODO()
	nodeInfoStore, err := kvstore.NewNodeStore(ctx, kvstore.NodeStoreParams{
		BucketName: kvstore.DefaultBucketName,
		Client:     transport.Client().Client,
	})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create node info store using NATS transport connection info")
	}
	// TODO use a decorator
	// tracingInfoStore := tracing.NewNodeStore(nodeInfoStore)

	// TODO might need lifycycle or invoke
	if err := transport.RegisterNodeInfoConsumer(ctx, nodeInfoStore); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to register node info consumer with nats transport")
	}

	return nodeInfoStore, nil
}

func NodeManager(store *kvstore.NodeStore) (*manager.NodeManager, error) {
	nodeManager := manager.NewNodeManager(manager.NodeManagerParams{NodeInfo: store})
	return nodeManager, nil
}

func NodeDiscoverer(store *kvstore.NodeStore) (orchestrator.NodeDiscoverer, error) {
	return discovery.NewStoreNodeDiscoverer(discovery.StoreNodeDiscovererParams{Store: store}), nil
}

func NodeRanker(store jobstore.Store, cfg node.RequesterConfig) (orchestrator.NodeRanker, error) {
	ranker := ranking.NewChain()
	ranker.Add(
		// rankers that act as filters and give a -1 score to nodes that do not match the filter
		ranking.NewEnginesNodeRanker(),
		ranking.NewPublishersNodeRanker(),
		ranking.NewStoragesNodeRanker(),
		ranking.NewLabelsNodeRanker(),
		ranking.NewMaxUsageNodeRanker(),
		ranking.NewMinVersionNodeRanker(ranking.MinVersionNodeRankerParams{MinVersion: cfg.MinBacalhauVersion}),
		ranking.NewPreviousExecutionsNodeRanker(ranking.PreviousExecutionsNodeRankerParams{JobStore: store}),
		// arbitrary rankers
		ranking.NewRandomNodeRanker(ranking.RandomNodeRankerParams{
			RandomnessRange: cfg.NodeRankRandomnessRange,
		}),
	)
	return ranker, nil
}

func NodeSelector(
	discoverer orchestrator.NodeDiscoverer,
	ranker orchestrator.NodeRanker,
) (orchestrator.NodeSelector, error) {
	// TODO can annotate this and return the concrete type
	return selector.NewNodeSelector(selector.NodeSelectorParams{
		NodeDiscoverer: discoverer,
		NodeRanker:     ranker,
	}), nil
}

func EvaluationBroker(lc fx.Lifecycle, cfg node.RequesterConfig) (orchestrator.EvaluationBroker, error) {
	evalBroker, err := evaluation.NewInMemoryBroker(evaluation.InMemoryBrokerParams{
		VisibilityTimeout:    cfg.EvalBrokerVisibilityTimeout,
		InitialRetryDelay:    cfg.EvalBrokerInitialRetryDelay,
		SubsequentRetryDelay: cfg.EvalBrokerSubsequentRetryDelay,
		MaxReceiveCount:      cfg.EvalBrokerMaxRetryCount,
	})
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			evalBroker.SetEnabled(true)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			evalBroker.SetEnabled(false)
			return nil
		},
	})
	return evalBroker, nil
}

func Planner(
	cfg node.RequesterConfig,
	transport *nats_transport.NATSTransport,
	store jobstore.Store,
	eventEmitter orchestrator.EventEmitter,
) (orchestrator.Planner, error) {
	// planners that execute the proposed plan by the scheduler
	// order of the planners is important as they are executed in order
	return planner.NewChain(
		// planner that persist the desired state as defined by the scheduler
		planner.NewStateUpdater(store),

		// planner that forwards the desired state to the compute nodes,
		// and updates the observed state if the compute node accepts the desired state
		planner.NewComputeForwarder(planner.ComputeForwarderParams{
			ID:             cfg.NodeID,
			ComputeService: transport.ComputeProxy(),
			JobStore:       store,
		}),

		// planner that publishes events on job completion or failure
		planner.NewEventEmitter(planner.EventEmitterParams{
			ID:           cfg.NodeID,
			EventEmitter: eventEmitter,
		}),

		// logs job completion or failure
		planner.NewLoggingPlanner(),
	), nil
}

// TODO this needs a config that is only ever modifed in testing
func RetryStrategy() (orchestrator.RetryStrategy, error) {
	// retry strategy
	retryStrategyChain := retry.NewChain()
	retryStrategyChain.Add(
		retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: true}),
	)
	return retryStrategyChain, nil
}

func SchedulerProvider(
	lc fx.Lifecycle,
	cfg node.RequesterConfig,
	store jobstore.Store,
	planner orchestrator.Planner,
	nodeSelector orchestrator.NodeSelector,
	strategy orchestrator.RetryStrategy,
	broker orchestrator.EvaluationBroker,
) (orchestrator.SchedulerProvider, error) {
	batch := scheduler.NewBatchServiceJobScheduler(scheduler.BatchServiceJobSchedulerParams{
		JobStore:      store,
		Planner:       planner,
		NodeSelector:  nodeSelector,
		RetryStrategy: strategy,
	})
	schedulerProvider := orchestrator.NewMappedSchedulerProvider(map[string]orchestrator.Scheduler{
		models.JobTypeBatch:   batch,
		models.JobTypeService: batch,
		models.JobTypeOps: scheduler.NewOpsJobScheduler(scheduler.OpsJobSchedulerParams{
			JobStore:     store,
			Planner:      planner,
			NodeSelector: nodeSelector,
		}),
		models.JobTypeDaemon: scheduler.NewDaemonJobScheduler(scheduler.DaemonJobSchedulerParams{
			JobStore:     store,
			Planner:      planner,
			NodeSelector: nodeSelector,
		}),
	})
	workers := make([]*orchestrator.Worker, 0, cfg.WorkerCount)
	for i := 1; i <= cfg.WorkerCount; i++ {
		// log.Debug().Msgf("Starting worker %d", i)
		// worker config the polls from the broker
		worker := orchestrator.NewWorker(orchestrator.WorkerParams{
			SchedulerProvider:     schedulerProvider,
			EvaluationBroker:      broker,
			DequeueTimeout:        cfg.WorkerEvalDequeueTimeout,
			DequeueFailureBackoff: backoff.NewExponential(cfg.WorkerEvalDequeueBaseBackoff, cfg.WorkerEvalDequeueMaxBackoff),
		})
		workers = append(workers, worker)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			for _, w := range workers {
				w.Start(ctx)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			for _, w := range workers {
				w.Stop()
			}
			return nil
		},
	})

	return schedulerProvider, nil
}

func EndpointV1(
	cfg node.RequesterConfig,
	transport *nats_transport.NATSTransport,
	broker orchestrator.EvaluationBroker,
	eventEmitter orchestrator.EventEmitter,
	store jobstore.Store,
) (*requester.BaseEndpoint, error) {
	return requester.NewBaseEndpoint(&requester.BaseEndpointParams{
		ID:                         cfg.NodeID,
		EvaluationBroker:           broker,
		Store:                      store,
		EventEmitter:               eventEmitter,
		ComputeEndpoint:            transport.ComputeProxy(),
		DefaultJobExecutionTimeout: cfg.JobDefaults.ExecutionTimeout,
		DefaultPublisher:           cfg.DefaultPublisher,
	}), nil
}

func RequesterDebugInfoProviders(
	discoverer orchestrator.NodeDiscoverer,
) ([]model.DebugInfoProvider, error) {
	return []model.DebugInfoProvider{discovery.NewDebugInfoProvider(discoverer)}, nil
}

type RequesterAPIParams struct {
	fx.In

	Router         *echo.Echo
	EndpointV1     requester.Endpoint
	Discoverer     orchestrator.NodeDiscoverer
	Store          jobstore.Store
	DebugProviders []model.DebugInfoProvider `name:"requester_debug_providers"`
}

func RequesterAPI(
	p RequesterAPIParams,
) (*requester_endpoint.Endpoint, error) {
	// register requester public http apis
	requesterAPIServer := requester_endpoint.NewEndpoint(requester_endpoint.EndpointParams{
		Router:             p.Router,
		Requester:          p.EndpointV1,
		JobStore:           p.Store,
		NodeDiscoverer:     p.Discoverer,
		DebugInfoProviders: p.DebugProviders,
	})
	return requesterAPIServer, nil
}

func OrchestratorAPI(
	router *echo.Echo,
	endpointV2 *orchestrator.BaseEndpoint,
	store jobstore.Store,
	nodeManager *manager.NodeManager,
) (*orchestrator_endpoint.Endpoint, error) {
	return orchestrator_endpoint.NewEndpoint(orchestrator_endpoint.EndpointParams{
		Router:       router,
		Orchestrator: endpointV2,
		JobStore:     store,
		NodeManager:  nodeManager,
	}), nil
}

func EndpointV2(
	cfg node.RequesterConfig,
	transport *nats_transport.NATSTransport,
	broker orchestrator.EvaluationBroker,
	eventEmitter orchestrator.EventEmitter,
	store jobstore.Store,
) (*orchestrator.BaseEndpoint, error) {
	// TODO make transformers a value group: https://uber-go.github.io/fx/value-groups/consume.html#with-annotated-functions
	jobTransformers := transformer.ChainedTransformer[*models.Job]{
		transformer.JobFn(transformer.IDGenerator),
		transformer.NameOptional(),
		transformer.DefaultsApplier(cfg.JobDefaults),
		transformer.RequesterInfo(cfg.NodeID),
		// transformer.NewInlineStoragePinner(storageProvider),
	}

	if cfg.DefaultPublisher != "" {
		// parse the publisher to generate a models.SpecConfig and add it to each job
		// which is without a publisher
		config, err := job.ParsePublisherString(cfg.DefaultPublisher)
		if err == nil {
			jobTransformers = append(jobTransformers, transformer.DefaultPublisher(config))
		}
	}

	// result transformers that are applied to the result before it is returned to the user
	resultTransformers := transformer.ChainedTransformer[*models.SpecConfig]{}

	if !cfg.S3PreSignedURLDisabled {
		// S3 result signer
		s3Config, err := s3helper.DefaultAWSConfig()
		if err != nil {
			return nil, err
		}
		resultSigner := s3helper.NewResultSigner(s3helper.ResultSignerParams{
			ClientProvider: s3helper.NewClientProvider(s3helper.ClientProviderParams{
				AWSConfig: s3Config,
			}),
			Expiration: cfg.S3PreSignedURLExpiration,
		})
		resultTransformers = append(resultTransformers, resultSigner)
	}

	var translationProvider translation.TranslatorProvider
	if cfg.TranslationEnabled {
		translationProvider = translation.NewStandardTranslatorsProvider()
	}

	endpointV2 := orchestrator.NewBaseEndpoint(&orchestrator.BaseEndpointParams{
		ID:                cfg.NodeID,
		EvaluationBroker:  broker,
		Store:             store,
		EventEmitter:      eventEmitter,
		ComputeProxy:      transport.ComputeProxy(),
		JobTransformer:    jobTransformers,
		TaskTranslator:    translationProvider,
		ResultTransformer: resultTransformers,
	})

	return endpointV2, nil
}

func Housekeeping(
	lc fx.Lifecycle,
	cfg node.RequesterConfig,
	endpoint requester.Endpoint,
	store jobstore.Store,
) (*requester.Housekeeping, error) {
	hk := requester.NewHousekeeping(requester.HousekeepingParams{
		Endpoint: endpoint,
		JobStore: store,
		NodeID:   cfg.NodeID,
		Interval: cfg.HousekeepingBackgroundTaskInterval,
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			hk.Start(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			hk.Stop()
			return nil
		},
	})
	return hk, nil
}

func EventTracer(lc fx.Lifecycle) (*eventhandler.Tracer, error) {
	eventTracer, err := eventhandler.NewTracer()
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return eventTracer.Shutdown()
		},
	})
	return eventTracer, nil
}

func RegisterEventConsumerHandlers(
	cfg node.RequesterConfig,
	tracer *eventhandler.Tracer,
	handler *eventhandler.ChainedJobEventHandler,
	provider *eventhandler.TracerContextProvider,
	endpoint *requester_endpoint.Endpoint,
) error {
	// Register event handlers
	lifecycleEventHandler := system.NewJobLifecycleEventHandler(cfg.NodeID)
	handler.AddHandlers(
		// add tracing metadata to the context about the read event
		eventhandler.JobEventHandlerFunc(lifecycleEventHandler.HandleConsumedJobEvent),
		// ends the span for the job if received a terminal event
		provider,
		// record the event in a log
		tracer,
		// dispatches events to listening websockets
		endpoint,
	)
	return nil
}

func RegisterTransportComputeCallback(
	transport *nats_transport.NATSTransport,
	endpointV1 compute.Callback,
) error {
	return transport.RegisterComputeCallback(endpointV1)
}

func RegisterTransportNodeManager(
	transport *nats_transport.NATSTransport,
	nodeManager *manager.NodeManager,
) error {
	return transport.RegisterManagementEndpoint(nodeManager)
}

func PopulateNodeManagerStore(provider *routing.NodeInfoProvider, nodeManager *manager.NodeManager) error {
	ctx := context.TODO()
	nodeInfo := provider.GetNodeInfo(ctx)
	nodeInfo.Approval = models.NodeApprovals.APPROVED
	return nodeManager.Add(ctx, nodeInfo)
}
