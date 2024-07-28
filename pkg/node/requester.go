package node

import (
	"context"
	"fmt"

	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
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
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/selector"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	auth_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/auth"
	orchestrator_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/orchestrator"
	requester_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/requester"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/kvstore"
	"github.com/bacalhau-project/bacalhau/pkg/routing/tracing"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/translation"
	"github.com/bacalhau-project/bacalhau/pkg/util"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/discovery"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/ranking"
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
	nodeID string,
	apiServer *publicapi.Server,
	nodeConfig NodeConfig,
	metricsConfig types.MetricsConfig,
	requesterConfig RequesterConfig,
	transportLayer *nats_transport.NATSTransport,
	computeProxy compute.Endpoint,
	payloadRegistry *ncl.PayloadRegistry,
) (*Requester, error) {
	nodeManager, heartbeatServer, err := createNodeManager(ctx, transportLayer, requesterConfig)
	if err != nil {
		return nil, err
	}

	// prepare event handlers
	tracerContextProvider := eventhandler.NewTracerContextProvider(nodeID)
	localJobEventConsumer := eventhandler.NewChainedJobEventHandler(tracerContextProvider)

	eventEmitter := orchestrator.NewEventEmitter(orchestrator.EventEmitterParams{
		EventConsumer: localJobEventConsumer,
	})

	jobStore := requesterConfig.JobStore

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

	// evaluations watcher
	evaluationsWatcher := evaluation.NewWatcher(jobStore, evalBroker)
	if err = evaluationsWatcher.Backfill(ctx); err != nil {
		return nil, fmt.Errorf("failed to backfill evaluations: %w", err)
	}
	evaluationsWatcher.Start(ctx)

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

	// node selector
	nodeRanker, err := createNodeRanker(requesterConfig, jobStore)
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
		QueueBackoff:  requesterConfig.SchedulerQueueBackoff,
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

	var translationProvider translation.TranslatorProvider
	if requesterConfig.TranslationEnabled {
		translationProvider = translation.NewStandardTranslatorsProvider()
	}

	jobTransformers := transformer.ChainedTransformer[*models.Job]{
		transformer.JobFn(transformer.IDGenerator),
		transformer.NameOptional(),
		transformer.DefaultsApplier(requesterConfig.JobDefaults),
		transformer.RequesterInfo(nodeID),
	}

	if requesterConfig.DefaultPublisher != "" {
		// parse the publisher to generate a models.SpecConfig and add it to each job
		// which is without a publisher
		config, err := job.ParsePublisherString(requesterConfig.DefaultPublisher)
		if err != nil {
			return nil, fmt.Errorf("parsing default publisher spec (%s): %w", requesterConfig.DefaultPublisher, err)
		}
		jobTransformers = append(jobTransformers, transformer.DefaultPublisher(config))
	}

	endpointV2 := orchestrator.NewBaseEndpoint(&orchestrator.BaseEndpointParams{
		ID:                nodeID,
		Store:             jobStore,
		EventEmitter:      eventEmitter,
		ComputeProxy:      computeProxy,
		JobTransformer:    jobTransformers,
		TaskTranslator:    translationProvider,
		ResultTransformer: resultTransformers,
	})

	housekeeping, err := orchestrator.NewHousekeeping(orchestrator.HousekeepingParams{
		JobStore:      jobStore,
		Interval:      requesterConfig.HousekeepingBackgroundTaskInterval,
		TimeoutBuffer: requesterConfig.HousekeepingTimeoutBuffer,
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

	authenticators, err := nodeConfig.DependencyInjector.AuthenticatorsFactory.Get(ctx, nodeConfig)
	if err != nil {
		return nil, err
	}
	metrics.NodeInfo.Add(ctx, 1,
		attribute.StringSlice("node_authenticators", authenticators.Keys(ctx)),
	)
	auth_endpoint.BindEndpoint(ctx, apiServer.Router, authenticators)

	// Register event handlers
	eventTracer, err := eventhandler.NewTracer(metricsConfig.EventTracerPath)
	if err != nil {
		return nil, err
	}

	// order of event handlers is important as triggering some handlers might depend on the state of others.
	localJobEventConsumer.AddHandlers(
		// ends the span for the job if received a terminal event
		tracerContextProvider,
		// record the event in a log
		eventTracer,
		// dispatches events to listening websockets
	)

	// ncl
	subscriber, err := ncl.NewSubscriber(transportLayer.Client(),
		ncl.WithSubscriberPayloadRegistry(payloadRegistry),
		ncl.WithSubscriberMessageHandlers(heartbeatServer),
	)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create ncl subscriber")
	}
	err = subscriber.Subscribe(requesterConfig.ControlPlaneSettings.HeartbeatTopic)
	if err != nil {
		return nil, err
	}

	// A single Cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		// close the ncl subscriber
		cleanupErr := subscriber.Close(ctx)
		if cleanupErr != nil {
			util.LogDebugIfContextCancelled(ctx, cleanupErr, "failed to cleanly shutdown ncl subscriber")
		}

		// stop the housekeeping background task
		housekeeping.Stop(ctx)
		for _, worker := range workers {
			worker.Stop()
		}
		evalBroker.SetEnabled(false)

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

	// This endpoint implements the protocol formerly known as `bprotocol`.
	// It provides the compute call back endpoints for interacting with compute nodes.
	// e.g. bidding, job completions, cancellations, and failures
	callback := orchestrator.NewCallback(&orchestrator.CallbackParams{
		ID:           nodeID,
		EventEmitter: eventEmitter,
		Store:        jobStore,
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

func createNodeRanker(requesterConfig RequesterConfig, jobStore jobstore.Store) (orchestrator.NodeRanker, error) {
	overSubscriptionNodeRanker, err := ranking.NewOverSubscriptionNodeRanker(requesterConfig.NodeOverSubscriptionFactor)
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
		ranking.NewMinVersionNodeRanker(ranking.MinVersionNodeRankerParams{MinVersion: requesterConfig.MinBacalhauVersion}),
		ranking.NewPreviousExecutionsNodeRanker(ranking.PreviousExecutionsNodeRankerParams{JobStore: jobStore}),
		ranking.NewAvailableCapacityNodeRanker(),
		// arbitrary rankers
		ranking.NewRandomNodeRanker(ranking.RandomNodeRankerParams{
			RandomnessRange: requesterConfig.NodeRankRandomnessRange,
		}),
	)
	return nodeRankerChain, nil
}

func createNodeManager(ctx context.Context,
	transportLayer *nats_transport.NATSTransport,
	requesterConfig RequesterConfig) (*manager.NodeManager, *heartbeat.HeartbeatServer, error) {
	nodeInfoStore, err := createNodeInfoStore(ctx, transportLayer)
	if err != nil {
		return nil, nil, err
	}

	// heartbeat service
	heartbeatParams := heartbeat.HeartbeatServerParams{
		CheckFrequency:        requesterConfig.ControlPlaneSettings.HeartbeatCheckFrequency.AsTimeDuration(),
		NodeDisconnectedAfter: requesterConfig.ControlPlaneSettings.NodeDisconnectedAfter.AsTimeDuration(),
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
		NodeInfo:             nodeInfoStore,
		Heartbeats:           heartbeatSvr,
		DefaultApprovalState: requesterConfig.DefaultApprovalState,
	})

	// Start the nodemanager, ensuring it doesn't block the main thread and
	// that any errors are logged. If we are unable to start the manager
	// then we should not start the node.
	if err = nodeManager.Start(ctx); err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to start node manager")
	}

	return nodeManager, heartbeatSvr, transportLayer.RegisterManagementEndpoint(nodeManager)
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

func (r *Requester) cleanup(ctx context.Context) {
	r.cleanupFunc(ctx)
}
