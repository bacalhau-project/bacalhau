package node

import (
	"context"
	"fmt"

	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
	"github.com/bacalhau-project/bacalhau/pkg/node/manager"
	"github.com/bacalhau-project/bacalhau/pkg/node/metrics"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/evaluation"
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
	requester_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/requester"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/kvstore"
	"github.com/bacalhau-project/bacalhau/pkg/routing/tracing"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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
	NodeInfoStore      routing.NodeInfoStore
	NodeDiscoverer     orchestrator.NodeDiscoverer
	nodeManager        *manager.NodeManager
	cleanupFunc        func(ctx context.Context)
	debugInfoProviders []models.DebugInfoProvider
}

//nolint:funlen,gocyclo
func NewRequesterNode(
	ctx context.Context,
	cfg NodeConfig,
	apiServer *publicapi.Server,
	transportLayer *nats_transport.NATSTransport,
	computeProxy compute.Endpoint,
	logstreamServer logstream.Server,
	messageSerDeRegistry *envelope.Registry,
	metadataStore MetadataStore,
) (*Requester, error) {
	nodeID := cfg.NodeID
	nodeManager, heartbeatServer, err := createNodeManager(ctx, cfg, transportLayer)
	if err != nil {
		return nil, err
	}

	jobStore, err := createJobStore(ctx, cfg)
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
		// planner that persist the desired state as defined by the scheduler
		planner.NewStateUpdater(jobStore),

		// logs job completion or failure
		planner.NewLoggingPlanner(),
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

	// node selector
	nodeRanker, err := createNodeRanker(cfg, jobStore)
	if err != nil {
		return nil, err
	}

	nodeSelector := selector.NewNodeSelector(
		nodeManager,
		nodeRanker,
		// selector constraints: require nodes be online and approved to schedule
		orchestrator.NodeSelectionConstraints{
			RequireConnected: true,
			RequireApproval:  true,
		},
	)

	// scheduler provider
	batchServiceJobScheduler := scheduler.NewBatchServiceJobScheduler(scheduler.BatchServiceJobSchedulerParams{
		JobStore:      jobStore,
		Planner:       planners,
		NodeSelector:  nodeSelector,
		RetryStrategy: retryStrategy,
		QueueBackoff:  cfg.BacalhauConfig.Orchestrator.Scheduler.QueueBackoff.AsTimeDuration(),
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

	// result transformers that are applied to the result before it is returned to the user
	resultTransformers := transformer.ChainedTransformer[*models.SpecConfig]{}

	if !cfg.BacalhauConfig.Publishers.Types.S3.PreSignedURLDisabled {
		// S3 result signer
		s3Config, err := s3helper.DefaultAWSConfig()
		if err != nil {
			return nil, err
		}
		resultSigner := s3helper.NewResultSigner(s3helper.ResultSignerParams{
			ClientProvider: s3helper.NewClientProvider(s3helper.ClientProviderParams{
				AWSConfig: s3Config,
			}),
			Expiration: cfg.BacalhauConfig.Publishers.Types.S3.PreSignedURLExpiration.AsTimeDuration(),
		})
		resultTransformers = append(resultTransformers, resultSigner)
	}

	jobTransformers := transformer.ChainedTransformer[*models.Job]{
		transformer.JobFn(transformer.IDGenerator),
		transformer.NameOptional(),
		transformer.RequesterInfo(nodeID),
		transformer.OrchestratorInstallationID(system.InstallationID()),
		transformer.OrchestratorInstanceID(metadataStore.InstanceID()),
		transformer.DefaultsApplier(cfg.BacalhauConfig.JobDefaults),
	}

	endpointV2 := orchestrator.NewBaseEndpoint(&orchestrator.BaseEndpointParams{
		ID:                nodeID,
		Store:             jobStore,
		LogstreamServer:   logstreamServer,
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
		discovery.NewDebugInfoProvider(nodeManager),
	}

	// TODO: delete this when we are ready to stop serving a deprecation notice.
	requester_endpoint.NewEndpoint(apiServer.Router)

	orchestrator_endpoint.NewEndpoint(orchestrator_endpoint.EndpointParams{
		Router:       apiServer.Router,
		Orchestrator: endpointV2,
		JobStore:     jobStore,
		NodeManager:  nodeManager,
	})

	authenticators, err := cfg.DependencyInjector.AuthenticatorsFactory.Get(ctx, cfg)
	if err != nil {
		return nil, err
	}
	metrics.NodeInfo.Add(ctx, 1,
		attribute.StringSlice("node_authenticators", authenticators.Keys(ctx)),
	)
	auth_endpoint.BindEndpoint(ctx, apiServer.Router, authenticators)

	// nclPublisher
	nclPublisher, err := ncl.NewPublisher(transportLayer.Client(),
		ncl.WithPublisherName(nodeID),
		ncl.WithPublisherMessageSerDeRegistry(messageSerDeRegistry),
	)
	if err != nil {
		return nil, err
	}

	watcherRegistry, err := setupOrchestratorWatchers(
		ctx, nodeID, jobStore, nclPublisher, evalBroker, nodeManager, computeProxy)
	if err != nil {
		return nil, err
	}

	// ncl subscriber
	subscriber, err := ncl.NewSubscriber(transportLayer.Client(),
		ncl.WithSubscriberMessageSerDeRegistry(messageSerDeRegistry),
		ncl.WithSubscriberMessageHandlers(orchestrator.NewMessageHandler(jobStore)),
	)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create ncl subscriber")
	}
	if err = subscriber.Subscribe(orchestratorInSubscription()); err != nil {
		return nil, err
	}

	// ncl heartbeat subscriber
	heartbeatSubscriber, err := ncl.NewSubscriber(transportLayer.Client(),
		ncl.WithSubscriberMessageSerDeRegistry(messageSerDeRegistry),
		ncl.WithSubscriberMessageHandlers(heartbeatServer),
	)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create heartbeat ncl subscriber")
	}
	if err = heartbeatSubscriber.Subscribe(orchestratorHeartbeatSubscription()); err != nil {
		return nil, err
	}

	// A single Cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		// close the ncl subscriber
		cleanupErr := subscriber.Close(ctx)
		if cleanupErr != nil {
			logDebugIfContextCancelled(ctx, cleanupErr, "failed to cleanly shutdown ncl subscriber")
		}

		// close the ncl heartbeat subscriber
		cleanupErr = heartbeatSubscriber.Close(ctx)
		if cleanupErr != nil {
			logDebugIfContextCancelled(ctx, cleanupErr, "failed to cleanly shutdown ncl heartbeat subscriber")
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
	}

	// This endpoint implements the protocol formerly known as `bprotocol`.
	// It provides the compute call back endpoints for interacting with compute nodes.
	// e.g. bidding, job completions, cancellations, and failures
	callback := orchestrator.NewCallback(&orchestrator.CallbackParams{
		ID:    nodeID,
		Store: jobStore,
	})
	if err = transportLayer.RegisterComputeCallback(callback); err != nil {
		return nil, err
	}

	return &Requester{
		Endpoint:           endpointV2,
		NodeDiscoverer:     nodeManager,
		NodeInfoStore:      nodeManager,
		JobStore:           jobStore,
		nodeManager:        nodeManager,
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

func createNodeManager(ctx context.Context,
	cfg NodeConfig,
	transportLayer *nats_transport.NATSTransport) (*manager.NodeManager, *heartbeat.HeartbeatServer, error) {
	nodeInfoStore, err := createNodeInfoStore(ctx, transportLayer)
	if err != nil {
		return nil, nil, err
	}

	// heartbeat service
	heartbeatParams := heartbeat.HeartbeatServerParams{
		NodeID:                cfg.NodeID,
		Client:                transportLayer.Client(),
		NodeDisconnectedAfter: cfg.BacalhauConfig.Orchestrator.NodeManager.DisconnectTimeout.AsTimeDuration(),
	}
	heartbeatSvr, err := heartbeat.NewServer(heartbeatParams)
	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to create heartbeat server using NATS transport connection info")
	}

	// node manager
	// Create a new node manager to keep track of compute nodes connecting
	// to the network. Provide it with a mechanism to lookup (and enhance)
	// node info, and a reference to the heartbeat server
	nodeManager := manager.NewNodeManager(manager.NodeManagerParams{
		NodeInfo:       nodeInfoStore,
		Heartbeats:     heartbeatSvr,
		ManualApproval: cfg.BacalhauConfig.Orchestrator.NodeManager.ManualApproval,
	})

	// Start the nodemanager, ensuring it doesn't block the main thread and
	// that any errors are logged. If we are unable to start the manager
	// then we should not start the node.
	if err = nodeManager.Start(ctx); err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to start node manager")
	}

	return nodeManager, heartbeatSvr, transportLayer.RegisterManagementEndpoint(nodeManager)
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

func createNodeInfoStore(ctx context.Context, transportLayer *nats_transport.NATSTransport) (routing.NodeInfoStore, error) {
	// nodeInfoStore
	nodeInfoStore, err := kvstore.NewNodeStore(ctx, kvstore.NodeStoreParams{
		BucketName: kvstore.BucketNameCurrent,
		Client:     transportLayer.Client(),
	})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create node info store using NATS transport connection info")
	}

	tracingInfoStore := tracing.NewNodeStore(nodeInfoStore)

	// Once the KV store has been created, it can be offered to the transport layer to be used as a consumer
	// of node info.
	if err = transportLayer.RegisterNodeInfoConsumer(ctx, tracingInfoStore); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to register node info consumer with nats transport")
	}
	return tracingInfoStore, nil
}

func setupOrchestratorWatchers(
	ctx context.Context,
	nodeID string,
	jobStore jobstore.Store,
	nclPublisher ncl.Publisher,
	evalBroker orchestrator.EvaluationBroker,
	nodeManager routing.NodeInfoStore,
	computeProxy compute.Endpoint,
) (watcher.Registry, error) {
	watcherRegistry := watcher.NewRegistry(jobStore.GetEventStore())

	// Start watching for evaluation events using latest iterator
	_, err := watcherRegistry.Watch(ctx, orchestratorEvaluationWatcherID, evaluation.NewWatchHandler(evalBroker),
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
	_, err = watcherRegistry.Watch(ctx, orchestratorExecutionLoggerWatcherID, watchers.NewExecutionLogger(log.Logger),
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{jobstore.EventObjectExecutionUpsert},
		}),
		watcher.WithInitialEventIterator(watcher.LatestIterator()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup orchestrator logger watcher: %w", err)
	}

	dispatcher, err := watchers.NewDispatcher(watchers.DispatcherParams{
		NodeStore: nodeManager,
		Dispatchers: map[models.Protocol]watcher.EventHandler{
			models.ProtocolBProtocolV2: watchers.NewBProtocolDispatcher(watchers.BProtocolDispatcherParams{
				ID:             nodeID,
				ComputeService: computeProxy,
				JobStore:       jobStore,
			}),
			models.ProtocolNCLV1: watchers.NewNCLDispatcher(watchers.NCLDispatcherParams{
				Publisher: nclPublisher,
				SubjectFn: orchestratorOutSubjectPrefix,
				JobStore:  jobStore,
			}),
		},
	})
	if err != nil {
		return nil, err
	}

	// TODO: Add checkpointing or else events will be missed
	_, err = watcherRegistry.Watch(ctx, orchestratorToComputeDispatcherWatcherID, dispatcher,
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{jobstore.EventObjectExecutionUpsert},
		}),
		watcher.WithRetryStrategy(watcher.RetryStrategySkip),
		watcher.WithMaxRetries(3),
		watcher.WithInitialEventIterator(watcher.LatestIterator()))
	if err != nil {
		return nil, err
	}

	return watcherRegistry, nil
}

func (r *Requester) cleanup(ctx context.Context) {
	r.cleanupFunc(ctx)
}
