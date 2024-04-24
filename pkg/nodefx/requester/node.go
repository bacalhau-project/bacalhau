package requester

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
	pkgerrors "github.com/pkg/errors"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
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
	fx.Provide(LoadConfig),
	fx.Provide(NewRequesterNode),
	fx.Provide(JobStore),
	fx.Provide(NodeStore),
	fx.Provide(NodeManager),
	fx.Provide(TracerContextProvider),
	fx.Provide(JobEventHandler),
	fx.Provide(EventEmitter),
	fx.Provide(EventTracer),
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
	fx.Provide(HeartbeatServer),

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
	NodeManager     *manager.NodeManager
	Scheduler       orchestrator.SchedulerProvider
}

type ConfigResult struct {
	fx.Out

	JobDefaults      types.JobDefaults
	EvaluationBroker types.EvaluationBrokerConfig
	JobStore         types.JobStoreConfig `name:"job_store_config"`
	Translation      types.TranslationConfig
	StorageProvider  types.StorageProviderConfig
	Metrics          types.MetricsConfig
	Worker           types.WorkerConfig
	NodeRanker       types.NodeRankerConfig
	Housekeeping     types.HousekeepingConfig
	NodeMembership   types.NodeMembershipConfig
	ControlPlan      types.RequesterControlPlaneConfig
	// JobSelectionPolicy model.JobSelectionPolicy
	// FailureInjectionConfig model.FailureInjectionRequesterConfig
	// TagCache types.DockerCacheConfig
}

func LoadConfig(c *config.Config) (ConfigResult, error) {
	var cfg types.RequesterConfig
	if err := c.ForKey(types.NodeRequester, &cfg); err != nil {
		return ConfigResult{}, err
	}

	var metricsCfg types.MetricsConfig
	if err := c.ForKey(types.Metrics, &metricsCfg); err != nil {
		return ConfigResult{}, err
	}

	return ConfigResult{
		JobDefaults:      cfg.JobDefaults,
		EvaluationBroker: cfg.EvaluationBroker,
		JobStore:         cfg.JobStore,
		Translation:      cfg.Translation,
		StorageProvider:  cfg.StorageProvider,
		Metrics:          metricsCfg,
		Worker:           cfg.Worker,
		NodeRanker:       cfg.NodeRanker,
		Housekeeping:     cfg.Housekeeping,
		NodeMembership:   cfg.NodeMembership,
		ControlPlan:      cfg.ControlPlaneSettings,
	}, nil
}

func NewRequesterNode(p RequesterParams) *RequesterNode {
	return &RequesterNode{
		Endpoint:        p.Endpoint,
		ComputeCallback: p.ComputeCallback,
		EndpointV2:      p.EndpointV2,
		JobStore:        p.JobStore,
		NodeManager:     p.NodeManager,
		Scheduler:       p.Scheduler,
	}
}

func TracerContextProvider(lc fx.Lifecycle, nodeID types.NodeID) (*eventhandler.TracerContextProvider, error) {
	provider := eventhandler.NewTracerContextProvider(string(nodeID))
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

func NodeManager(lc fx.Lifecycle, store *kvstore.NodeStore, hbs *heartbeat.HeartbeatServer, cfg types.NodeMembershipConfig) (*manager.NodeManager, error) {
	defaultApproval := models.NodeMembership.PENDING
	if cfg.AutoApproveNodes {
		defaultApproval = models.NodeMembership.APPROVED
	}
	nodeManager := manager.NewNodeManager(manager.NodeManagerParams{
		NodeInfo:             store,
		Heartbeats:           hbs,
		DefaultApprovalState: defaultApproval,
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nodeManager.Start(ctx)
		},
	})
	return nodeManager, nil
}

func NodeRanker(store jobstore.Store, cfg types.NodeRankerConfig) (orchestrator.NodeRanker, error) {
	ranker := ranking.NewChain()
	rndmRanker, err := ranking.NewRandomNodeRanker(
		ranking.RandomNodeRankerParams{RandomnessRange: cfg.NodeRankRandomnessRange},
	)
	if err != nil {
		return nil, err
	}
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
		rndmRanker,
	)
	return ranker, nil
}

func NodeSelector(
	store *kvstore.NodeStore,
	ranker orchestrator.NodeRanker,
) (orchestrator.NodeSelector, error) {
	// TODO can annotate this and return the concrete type
	return selector.NewNodeSelector(selector.NodeSelectorParams{
		NodeDiscoverer: store,
		NodeRanker:     ranker,
	}), nil
}

func EvaluationBroker(lc fx.Lifecycle, cfg types.EvaluationBrokerConfig) (orchestrator.EvaluationBroker, error) {
	evalBroker, err := evaluation.NewInMemoryBroker(evaluation.InMemoryBrokerParams{
		VisibilityTimeout:    time.Duration(cfg.EvalBrokerVisibilityTimeout),
		InitialRetryDelay:    time.Duration(cfg.EvalBrokerInitialRetryDelay),
		SubsequentRetryDelay: time.Duration(cfg.EvalBrokerSubsequentRetryDelay),
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
	nodeID types.NodeID,
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
			ID:             string(nodeID),
			ComputeService: transport.ComputeProxy(),
			JobStore:       store,
		}),

		// planner that publishes events on job completion or failure
		planner.NewEventEmitter(planner.EventEmitterParams{
			ID:           string(nodeID),
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
	cfg types.WorkerConfig,
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
			DequeueTimeout:        time.Duration(cfg.WorkerEvalDequeueTimeout),
			DequeueFailureBackoff: backoff.NewExponential(time.Duration(cfg.WorkerEvalDequeueBaseBackoff), time.Duration(cfg.WorkerEvalDequeueMaxBackoff)),
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
	nodeID types.NodeID,
	cfg types.JobDefaults,
	transport *nats_transport.NATSTransport,
	broker orchestrator.EvaluationBroker,
	eventEmitter orchestrator.EventEmitter,
	store jobstore.Store,
) (*requester.BaseEndpoint, error) {
	return requester.NewBaseEndpoint(&requester.BaseEndpointParams{
		ID:                         string(nodeID),
		EvaluationBroker:           broker,
		Store:                      store,
		EventEmitter:               eventEmitter,
		ComputeEndpoint:            transport.ComputeProxy(),
		DefaultJobExecutionTimeout: time.Duration(cfg.ExecutionTimeout),
		DefaultPublisher:           cfg.DefaultPublisher,
	}), nil
}

func RequesterDebugInfoProviders(
	nodeStore *kvstore.NodeStore,
) ([]model.DebugInfoProvider, error) {
	return []model.DebugInfoProvider{discovery.NewDebugInfoProvider(nodeStore)}, nil
}

type RequesterAPIParams struct {
	fx.In

	Router         *echo.Echo
	EndpointV1     requester.Endpoint
	NodeStore      *kvstore.NodeStore
	JobStore       jobstore.Store
	DebugProviders []model.DebugInfoProvider `name:"requester_debug_providers"`
}

func RequesterAPI(
	p RequesterAPIParams,
) (*requester_endpoint.Endpoint, error) {
	// register requester public http apis
	requesterAPIServer := requester_endpoint.NewEndpoint(requester_endpoint.EndpointParams{
		Router:             p.Router,
		Requester:          p.EndpointV1,
		JobStore:           p.JobStore,
		NodeDiscoverer:     p.NodeStore,
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
	nodeID types.NodeID,
	jobcfg types.JobDefaults,
	strgCfg types.StorageProviderConfig,
	trnslCfg types.TranslationConfig,
	transport *nats_transport.NATSTransport,
	broker orchestrator.EvaluationBroker,
	eventEmitter orchestrator.EventEmitter,
	store jobstore.Store,
) (*orchestrator.BaseEndpoint, error) {
	// TODO make transformers a value group: https://uber-go.github.io/fx/value-groups/consume.html#with-annotated-functions
	jobTransformers := transformer.ChainedTransformer[*models.Job]{
		transformer.JobFn(transformer.IDGenerator),
		transformer.NameOptional(),
		transformer.DefaultsApplier(transformer.JobDefaults{ExecutionTimeout: time.Duration(jobcfg.ExecutionTimeout)}),
		transformer.RequesterInfo(string(nodeID)),
		// transformer.NewInlineStoragePinner(storageProvider),
	}

	if jobcfg.DefaultPublisher != "" {
		// parse the publisher to generate a models.SpecConfig and add it to each job
		// which is without a publisher
		config, err := job.ParsePublisherString(jobcfg.DefaultPublisher)
		if err == nil {
			jobTransformers = append(jobTransformers, transformer.DefaultPublisher(config))
		}
	}

	// result transformers that are applied to the result before it is returned to the user
	resultTransformers := transformer.ChainedTransformer[*models.SpecConfig]{}

	if !strgCfg.S3.PreSignedURLDisabled {
		// S3 result signer
		s3Config, err := s3helper.DefaultAWSConfig()
		if err != nil {
			return nil, err
		}
		resultSigner := s3helper.NewResultSigner(s3helper.ResultSignerParams{
			ClientProvider: s3helper.NewClientProvider(s3helper.ClientProviderParams{
				AWSConfig: s3Config,
			}),
			Expiration: time.Duration(strgCfg.S3.PreSignedURLExpiration),
		})
		resultTransformers = append(resultTransformers, resultSigner)
	}

	var translationProvider translation.TranslatorProvider
	if trnslCfg.TranslationEnabled {
		translationProvider = translation.NewStandardTranslatorsProvider()
	}

	endpointV2 := orchestrator.NewBaseEndpoint(&orchestrator.BaseEndpointParams{
		ID:                string(nodeID),
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
	nodeID types.NodeID,
	cfg types.HousekeepingConfig,
	endpoint requester.Endpoint,
	store jobstore.Store,
) (*requester.Housekeeping, error) {
	hk := requester.NewHousekeeping(requester.HousekeepingParams{
		Endpoint: endpoint,
		JobStore: store,
		NodeID:   string(nodeID),
		Interval: time.Duration(cfg.HousekeepingBackgroundTaskInterval),
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

type HeartbeatServerParams struct {
	fx.In

	Transport *nats_transport.NATSTransport
	Config    types.RequesterControlPlaneConfig
}

func HeartbeatServer(p HeartbeatServerParams) (*heartbeat.HeartbeatServer, error) {
	heartbeatParams := heartbeat.HeartbeatServerParams{
		Client:                p.Transport.Client().Client,
		Topic:                 p.Config.HeartbeatTopic,
		CheckFrequency:        time.Duration(p.Config.HeartbeatCheckFrequency),
		NodeDisconnectedAfter: time.Duration(p.Config.NodeDisconnectedAfter),
	}
	// TODO(forrest) [refactor] the heartbeat server can be started here, but we don't do it here because
	// the NodeManager starts the heartbeat server...weird setup, should probably make them into a single
	// module.
	heartbeatSvr, err := heartbeat.NewServer(heartbeatParams)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create heartbeat server using NATS transport connection info")
	}
	return heartbeatSvr, nil
}

func EventTracer(lc fx.Lifecycle, cfg types.MetricsConfig) (*eventhandler.Tracer, error) {
	eventTracer, err := eventhandler.NewTracer(cfg.EventTracerPath)
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
	nodeID types.NodeID,
	tracer *eventhandler.Tracer,
	handler *eventhandler.ChainedJobEventHandler,
	provider *eventhandler.TracerContextProvider,
	endpoint *requester_endpoint.Endpoint,
) error {
	// Register event handlers
	lifecycleEventHandler := system.NewJobLifecycleEventHandler(string(nodeID))
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

func PopulateNodeManagerStore(provider *routing.NodeStateProvider, nodeManager *manager.NodeManager) error {
	ctx := context.TODO()
	nodeInfo := provider.GetNodeState(ctx)
	nodeInfo.Membership = models.NodeMembership.APPROVED
	return nodeManager.Add(ctx, nodeInfo)
}
