package node

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	natsutil "github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/nats/proxy"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node/metrics"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/evaluation"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes/kvstore"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/planner"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/scheduler"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/discovery"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/ranking"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/selector"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/watchers"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	auth_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/auth"
	orchestrator_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/s3managed"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	bprotocolorchestrator "github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
	transportorchestrator "github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/orchestrator"
)

var (
	minBacalhauVersion = models.BuildVersionInfo{
		Major: "1", Minor: "0", GitVersion: "v1.0.4",
	}
)

type Requester struct {
	// Visible for testing
	Endpoint *orchestrator.BaseEndpoint
	JobStore jobstore.Store
	// We need a reference to the node info store until libp2p is removed
	NodeInfoStore      nodes.Lookup
	cleanupFunc        func(ctx context.Context)
	debugInfoProviders []models.DebugInfoProvider
}

//nolint:funlen,gocyclo
func NewRequesterNode(
	ctx context.Context,
	cfg NodeConfig,
	apiServer *publicapi.Server,
	transportLayer *nats_transport.NATSTransport,
	metadataStore MetadataStore,
	nodeInfoProvider models.DecoratorNodeInfoProvider) (*Requester, error) {
	jobStore, err := createJobStore(ctx, cfg)
	if err != nil {
		return nil, err
	}

	natsConn, err := transportLayer.CreateClient(ctx)
	if err != nil {
		return nil, err
	}

	nodeID := cfg.NodeID
	nodesManager, _, err := createNodeManager(ctx, cfg, jobStore.GetEventStore(), nodeInfoProvider, natsConn)
	if err != nil {
		return nil, err
	}

	// evaluation broker
	evalBroker, err := evaluation.NewInMemoryBroker(evaluation.InMemoryBrokerParams{
		VisibilityTimeout: cfg.BacalhauConfig.Orchestrator.EvaluationBroker.VisibilityTimeout.AsTimeDuration(),
		MaxReceiveCount:   cfg.BacalhauConfig.Orchestrator.EvaluationBroker.MaxRetryCount,
	})
	if err != nil {
		return nil, err
	}
	evalBroker.SetEnabled(true)

	// planners that execute the proposed plan by the scheduler
	// order of the planners is important as they are executed in order
	planners := planner.NewChain(
		// logs job completion or failure
		planner.NewLoggingPlanner(),

		// metrics planner
		planner.NewMetricsPlanner(),

		// planner that persist the desired state as defined by the scheduler
		planner.NewStateUpdater(jobStore),
	)

	retryStrategy := cfg.SystemConfig.RetryStrategy
	if retryStrategy == nil {
		// retry strategy
		retryStrategyChain := retry.NewChain()
		retryStrategyChain.Add(
			retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: true}),
		)
		retryStrategy = retryStrategyChain
	}

	protocolRouter, err := watchers.NewProtocolRouter(watchers.ProtocolRouterParams{
		NodeStore:          nodesManager,
		SupportedProtocols: []models.Protocol{models.ProtocolBProtocolV2, models.ProtocolNCLV1},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create protocol router: %w", err)
	}

	// node selector
	nodeRanker, err := createNodeRanker(cfg, jobStore)
	if err != nil {
		return nil, err
	}

	nodeSelector := selector.NewNodeSelector(
		nodesManager,
		nodeRanker,
		// selector constraints: require nodes be online and approved to schedule
		orchestrator.NodeSelectionConstraints{
			RequireConnected: true,
			RequireApproval:  true,
		},
	)

	executionRateLimiter := scheduler.NewBatchRateLimiter(scheduler.BatchRateLimiterParams{
		MaxExecutionsPerEval:  cfg.SystemConfig.MaxExecutionsPerEval,
		ExecutionLimitBackoff: cfg.SystemConfig.ExecutionLimitBackoff,
	})

	// scheduler provider
	batchServiceJobScheduler := scheduler.NewBatchServiceJobScheduler(scheduler.BatchServiceJobSchedulerParams{
		JobStore:      jobStore,
		Planner:       planners,
		NodeSelector:  nodeSelector,
		RetryStrategy: retryStrategy,
		QueueBackoff:  cfg.BacalhauConfig.Orchestrator.Scheduler.QueueBackoff.AsTimeDuration(),
		RateLimiter:   executionRateLimiter,
	})
	schedulerProvider := orchestrator.NewMappedSchedulerProvider(map[string]orchestrator.Scheduler{
		models.JobTypeBatch:   batchServiceJobScheduler,
		models.JobTypeService: batchServiceJobScheduler,
		models.JobTypeOps: scheduler.NewOpsJobScheduler(scheduler.OpsJobSchedulerParams{
			JobStore:     jobStore,
			Planner:      planners,
			NodeSelector: nodeSelector,
			RateLimiter:  executionRateLimiter,
		}),
		models.JobTypeDaemon: scheduler.NewDaemonJobScheduler(scheduler.DaemonJobSchedulerParams{
			JobStore:     jobStore,
			Planner:      planners,
			NodeSelector: nodeSelector,
			RateLimiter:  executionRateLimiter,
		}),
	})

	workerCount := cfg.BacalhauConfig.Orchestrator.Scheduler.WorkerCount
	workers := make([]*orchestrator.Worker, 0, workerCount)
	for i := 1; i <= workerCount; i++ {
		log.Debug().Msgf("Starting worker %d", i)
		// worker config the polls from the broker
		worker := orchestrator.NewWorker(orchestrator.WorkerParams{
			SchedulerProvider: schedulerProvider,
			EvaluationBroker:  evalBroker,
		})
		workers = append(workers, worker)
		worker.Start(ctx)
	}

	s3Config, err := s3helper.DefaultAWSConfig()
	if err != nil {
		return nil, err
	}

	// S3 managed publisher URL generator
	// This will return an error if the configuration is provided but incorrect,
	// If the configuration is not provided, we still create the generator
	// so we can return meaningful errors to compute nodes that try to use the managed publisher.
	s3ManagedPublisherURLGenerator, err := s3managed.NewPreSignedURLGenerator(s3managed.PreSignedURLGeneratorParams{
		ClientProvider:  s3helper.NewClientProvider(s3helper.ClientProviderParams{AWSConfig: s3Config}),
		PublisherConfig: cfg.BacalhauConfig.Publishers.Types.S3Managed,
	})
	if err != nil {
		return nil, bacerrors.Wrap(err, "failed to create S3 managed publisher URL generator").
			WithHint("Check if the S3 managed publisher configuration is correct and if the S3 client is available")
	}

	// result transformers that are applied to the result before it is returned to the user
	resultTransformers := transformer.ChainedTransformer[*models.SpecConfig]{}

	if !cfg.BacalhauConfig.Publishers.Types.S3.PreSignedURLDisabled {
		// S3 result signer
		resultSigner := s3helper.NewResultSigner(s3helper.ResultSignerParams{
			ClientProvider: s3helper.NewClientProvider(s3helper.ClientProviderParams{
				AWSConfig: s3Config,
			}),
			Expiration: cfg.BacalhauConfig.Publishers.Types.S3.PreSignedURLExpiration.AsTimeDuration(),
		})
		resultTransformers = append(resultTransformers, resultSigner)
	}

	// S3 managed publisher result transformer
	resultTransformers = append(resultTransformers, s3managed.NewResultTransformer(s3ManagedPublisherURLGenerator))

	jobTransformers := transformer.ChainedTransformer[*models.Job]{
		transformer.JobFn(transformer.IDGenerator),
		transformer.NameOptional(),
		transformer.RequesterInfo(nodeID),
		transformer.OrchestratorInstallationID(system.InstallationID()),
		transformer.OrchestratorInstanceID(metadataStore.InstanceID()),
		transformer.DefaultsApplier(cfg.BacalhauConfig.JobDefaults),
		transformer.NewLegacyWasmModuleTransformer(),
	}

	logStreamProxy, err := proxy.NewLogStreamProxy(proxy.LogStreamProxyParams{
		Conn: natsConn,
	})
	if err != nil {
		return nil, err
	}

	endpointV2 := orchestrator.NewBaseEndpoint(&orchestrator.BaseEndpointParams{
		ID:                nodeID,
		Store:             jobStore,
		LogstreamServer:   logStreamProxy,
		JobTransformer:    jobTransformers,
		ResultTransformer: resultTransformers,
	})

	housekeeping, err := orchestrator.NewHousekeeping(orchestrator.HousekeepingParams{
		JobStore:      jobStore,
		Interval:      cfg.BacalhauConfig.Orchestrator.Scheduler.HousekeepingInterval.AsTimeDuration(),
		TimeoutBuffer: cfg.BacalhauConfig.Orchestrator.Scheduler.HousekeepingTimeout.AsTimeDuration(),
	})
	if err != nil {
		return nil, err
	}
	housekeeping.Start(ctx)

	// register debug info providers for the /debug endpoint
	debugInfoProviders := []models.DebugInfoProvider{
		discovery.NewDebugInfoProvider(nodesManager),
	}

	orchestrator_endpoint.NewEndpoint(orchestrator_endpoint.EndpointParams{
		Router:       apiServer.Router,
		Orchestrator: endpointV2,
		JobStore:     jobStore,
		NodeManager:  nodesManager,
	})

	authenticators, err := cfg.DependencyInjector.AuthenticatorsFactory.Get(ctx, cfg)
	if err != nil {
		return nil, err
	}
	metrics.NodeInfo.Add(ctx, 1,
		attribute.StringSlice("node_authenticators", authenticators.Keys(ctx)),
	)
	auth_endpoint.BindEndpoint(ctx, apiServer.Router, authenticators)

	// legacy connection manager
	legacyConnectionManager, err := bprotocolorchestrator.NewConnectionManager(bprotocolorchestrator.Config{
		NodeID:         nodeID,
		NatsConn:       natsConn,
		NodeManager:    nodesManager,
		EventStore:     jobStore.GetEventStore(),
		ProtocolRouter: protocolRouter,
		Callback:       orchestrator.NewCallback(&orchestrator.CallbackParams{ID: nodeID, Store: jobStore}),
	})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create connection manager")
	}
	if err = legacyConnectionManager.Start(ctx); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to start connection manager")
	}

	// connection manager
	connectionManager, err := transportorchestrator.NewComputeManager(transportorchestrator.Config{
		NodeID:                  cfg.NodeID,
		ClientFactory:           natsutil.ClientFactoryFunc(transportLayer.CreateClient),
		NodeManager:             nodesManager,
		HeartbeatTimeout:        cfg.BacalhauConfig.Orchestrator.NodeManager.DisconnectTimeout.AsTimeDuration(),
		DataPlaneMessageHandler: orchestrator.NewMessageHandler(jobStore),
		DataPlaneMessageCreatorFactory: watchers.NewNCLMessageCreatorFactory(watchers.NCLMessageCreatorFactoryParams{
			ProtocolRouter: protocolRouter,
			SubjectFn:      nclprotocol.NatsSubjectComputeInMsgs,
		}),
		EventStore: jobStore.GetEventStore(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	if err = connectionManager.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start connection manager: %w", err)
	}

	// Register S3 managed publisher handlers.
	// We want to always register these, even if the managed S3 publisher is not enabled,
	// so the orchestrator can return meaningful errors to compute nodes that try to use the managed publisher.

	// Message handler for generating pre-signed URLs for S3 managed publisher
	err = connectionManager.RegisterDataPlaneHandler(
		ctx,
		messages.ManagedPublisherPreSignURLRequestType,
		s3managed.NewPreSignedURLRequestHandler(s3ManagedPublisherURLGenerator),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register a handler for S3 managed publisher pre-sign url messages: %w", err)
	}

	watcherRegistry, err := setupOrchestratorWatchers(ctx, jobStore, evalBroker)
	if err != nil {
		return nil, err
	}

	// Create ReEvaluator for automatic job re-evaluation on node state changes
	reEvaluator, err := nodes.NewReEvaluator(nodes.ReEvaluatorParams{
		JobStore:     jobStore,
		BatchDelay:   cfg.SystemConfig.NodeReEvaluatorBatchDelay,
		MaxBatchSize: cfg.SystemConfig.NodeReEvaluatorMaxBatchSize,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create re-evaluator: %w", err)
	}

	// Register ReEvaluator with node manager for connection state events
	nodesManager.OnConnectionStateChange(reEvaluator.HandleNodeConnectionEvent)

	// Start the ReEvaluator
	if err = reEvaluator.Start(ctx); err != nil {
		return nil, err
	}

	if err = nodesManager.Start(ctx); err != nil {
		return nil, err
	}

	// A single Cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		var cleanupErr error

		// stop the node reEvaluator
		if cleanupErr = reEvaluator.Stop(ctx); cleanupErr != nil {
			logDebugIfContextCancelled(ctx, cleanupErr, "failed to cleanly shutdown re-evaluator")
		}

		// stop the legacy connection manager
		legacyConnectionManager.Stop(ctx)

		// stop the connection manager
		if cleanupErr = connectionManager.Stop(ctx); cleanupErr != nil {
			logDebugIfContextCancelled(ctx, cleanupErr, "failed to cleanly shutdown connection manager")
		}

		if cleanupErr = watcherRegistry.Stop(ctx); cleanupErr != nil {
			logDebugIfContextCancelled(ctx, cleanupErr, "failed to stop watcher registry")
		}

		// stop the housekeeping background task
		housekeeping.Stop(ctx)
		for _, worker := range workers {
			worker.Stop()
		}
		evalBroker.SetEnabled(false)

		// Close the jobstore after the evaluation broker is disabled
		cleanupErr = jobStore.Close(ctx)
		if cleanupErr != nil {
			logDebugIfContextCancelled(ctx, cleanupErr, "failed to cleanly shutdown jobstore")
		}

		// stop node manager
		cleanupErr = nodesManager.Stop(ctx)
		if cleanupErr != nil {
			logDebugIfContextCancelled(ctx, cleanupErr, "failed to cleanly shutdown node manager")
		}
	}

	return &Requester{
		Endpoint:           endpointV2,
		NodeInfoStore:      nodesManager,
		JobStore:           jobStore,
		cleanupFunc:        cleanupFunc,
		debugInfoProviders: debugInfoProviders,
	}, nil
}

func createNodeRanker(cfg NodeConfig, jobStore jobstore.Store) (orchestrator.NodeRanker, error) {
	overSubscriptionNodeRanker, err := ranking.NewOverSubscriptionNodeRanker(cfg.SystemConfig.OverSubscriptionFactor)
	if err != nil {
		return nil, err
	}
	// compute node ranker
	nodeRankerChain := ranking.NewChain()
	nodeRankerChain.Add(
		// rankers that act as filters and give a -1 score to nodes that do not match the filter
		ranking.NewEnginesNodeRanker(),
		ranking.NewPublishersNodeRanker(),
		ranking.NewStoragesNodeRanker(),
		ranking.NewLabelsNodeRanker(),
		ranking.NewMaxUsageNodeRanker(),
		overSubscriptionNodeRanker,
		ranking.NewMinVersionNodeRanker(ranking.MinVersionNodeRankerParams{MinVersion: minBacalhauVersion}),
		ranking.NewPreviousExecutionsNodeRanker(ranking.PreviousExecutionsNodeRankerParams{JobStore: jobStore}),
		ranking.NewAvailableCapacityNodeRanker(),
		// arbitrary rankers
		ranking.NewRandomNodeRanker(ranking.RandomNodeRankerParams{
			RandomnessRange: cfg.SystemConfig.NodeRankRandomnessRange,
		}),
	)
	return nodeRankerChain, nil
}

func createJobStore(ct context.Context, cfg NodeConfig) (jobstore.Store, error) {
	jobStoreDBPath, err := cfg.BacalhauConfig.JobStoreFilePath()
	if err != nil {
		return nil, err
	}
	jobStore, err := boltjobstore.NewBoltJobStore(jobStoreDBPath)
	if err != nil {
		return nil, bacerrors.Wrap(err, "failed to create job store")
	}
	return jobStore, nil
}

func createNodeManager(ctx context.Context,
	cfg NodeConfig,
	eventStore watcher.EventStore,
	nodeInfoProvider models.DecoratorNodeInfoProvider,
	natsConn *nats.Conn) (nodes.Manager, nodes.Store, error) {
	nodeInfoStore, err := kvstore.NewNodeStore(ctx, kvstore.NodeStoreParams{
		BucketName: kvstore.BucketNameCurrent,
		Client:     natsConn,
	})
	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to create node info store using NATS transport connection info")
	}

	nodeManager, err := nodes.NewManager(nodes.ManagerParams{
		Store:                 nodeInfoStore,
		NodeDisconnectedAfter: cfg.BacalhauConfig.Orchestrator.NodeManager.DisconnectTimeout.AsTimeDuration(),
		ManualApproval:        cfg.BacalhauConfig.Orchestrator.NodeManager.ManualApproval,
		EventStore:            eventStore,
		NodeInfoProvider:      nodeInfoProvider,
	})

	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to create node manager")
	}

	return nodeManager, nodeInfoStore, nil
}

//nolint:funlen
func setupOrchestratorWatchers(
	ctx context.Context,
	jobStore jobstore.Store,
	evalBroker orchestrator.EvaluationBroker,
) (watcher.Manager, error) {
	watcherRegistry := watcher.NewManager(jobStore.GetEventStore())

	// Start watching for evaluation events using latest iterator
	_, err := watcherRegistry.Create(ctx, orchestratorEvaluationWatcherID,
		watcher.WithHandler(evaluation.NewWatchHandler(evalBroker)),
		watcher.WithAutoStart(),
		watcher.WithInitialEventIterator(watcher.LatestIterator()),
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{jobstore.EventObjectEvaluation},
			Operations:  []watcher.Operation{watcher.OperationCreate},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start evaluation watcher: %w", err)
	}

	// Set up execution logger watcher
	_, err = watcherRegistry.Create(ctx, orchestratorExecutionLoggerWatcherID,
		watcher.WithHandler(watchers.NewExecutionLogger(log.Logger)),
		watcher.WithEphemeral(),
		watcher.WithAutoStart(),
		watcher.WithInitialEventIterator(watcher.LatestIterator()),
		watcher.WithRetryStrategy(watcher.RetryStrategySkip),
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{jobstore.EventObjectExecutionUpsert},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup orchestrator logger watcher: %w", err)
	}

	return watcherRegistry, nil
}

func (r *Requester) cleanup(ctx context.Context) {
	r.cleanupFunc(ctx)
}
