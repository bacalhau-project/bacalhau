package node

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/disk"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/compute/sensors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/compute/watchers"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/nats"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	compute_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/compute"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	bprotocolcompute "github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol/compute"
	"github.com/bacalhau-project/bacalhau/pkg/transport/dispatcher"
)

type Compute struct {
	// Visible for testing
	ID                 string
	LocalEndpoint      compute.Endpoint
	LogstreamServer    logstream.Server
	Capacity           capacity.Tracker
	ExecutionStore     store.ExecutionStore
	Executors          executor.ExecProvider
	Storages           storage.StorageProvider
	Publishers         publisher.PublisherProvider
	Bidder             compute.Bidder
	Watchers           watcher.Manager
	cleanupFunc        func(ctx context.Context)
	debugInfoProviders []models.DebugInfoProvider
}

//nolint:funlen,gocyclo
func NewComputeNode(
	ctx context.Context,
	cfg NodeConfig,
	apiServer *publicapi.Server,
	transportLayer *nats_transport.NATSTransport,
	nodeInfoProvider *models.BaseNodeInfoProvider,
) (*Compute, error) {
	// Setup dependencies
	publishers, err := cfg.DependencyInjector.PublishersFactory.Get(ctx, cfg)
	if err != nil {
		return nil, err
	}

	executors, err := cfg.DependencyInjector.ExecutorsFactory.Get(ctx, cfg)
	if err != nil {
		return nil, err
	}

	storages, err := cfg.DependencyInjector.StorageProvidersFactory.Get(ctx, cfg)
	if err != nil {
		return nil, err
	}

	executionStore, err := createExecutionStore(ctx, cfg)
	if err != nil {
		return nil, err
	}

	executionDir, err := cfg.BacalhauConfig.ExecutionDir()
	if err != nil {
		return nil, err
	}

	allocatedResources, err := getAllocatedResources(ctx, cfg.BacalhauConfig, executionDir)
	if err != nil {
		return nil, err
	}

	// executor/backend
	runningCapacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: allocatedResources,
	})
	enqueuedUsageTracker := capacity.NewLocalUsageTracker()

	resultsPath, err := compute.NewResultsPath()
	if err != nil {
		return nil, err
	}

	baseExecutor := compute.NewBaseExecutor(compute.BaseExecutorParams{
		ID:                     cfg.NodeID,
		Store:                  executionStore,
		StorageDirectory:       executionDir,
		Storages:               storages,
		Executors:              executors,
		Publishers:             publishers,
		FailureInjectionConfig: cfg.FailureInjectionConfig,
		ResultsPath:            *resultsPath,
	})

	bufferRunner := compute.NewExecutorBuffer(compute.ExecutorBufferParams{
		ID:                     cfg.NodeID,
		DelegateExecutor:       baseExecutor,
		RunningCapacityTracker: runningCapacityTracker,
		EnqueuedUsageTracker:   enqueuedUsageTracker,
	})
	runningInfoProvider := sensors.NewRunningExecutionsInfoProvider(sensors.RunningExecutionsInfoProviderParams{
		Name:          "ActiveJobs",
		BackendBuffer: bufferRunner,
	})
	if cfg.BacalhauConfig.Logging.LogDebugInfoInterval > 0 {
		loggingSensor := sensors.NewLoggingSensor(sensors.LoggingSensorParams{
			InfoProvider: runningInfoProvider,
			Interval:     cfg.BacalhauConfig.Logging.LogDebugInfoInterval.AsTimeDuration(),
		})
		go loggingSensor.Start(ctx)
	}

	// endpoint/frontend
	capacityCalculator := capacity.NewChainedUsageCalculator(capacity.ChainedUsageCalculatorParams{
		Calculators: []capacity.UsageCalculator{
			capacity.NewDefaultsUsageCalculator(capacity.DefaultsUsageCalculatorParams{
				Defaults: cfg.SystemConfig.DefaultComputeJobResourceLimits,
			}),
			disk.NewDiskUsageCalculator(disk.DiskUsageCalculatorParams{
				Storages: storages,
			}),
		},
	})

	// logging server
	logserver := logstream.NewServer(logstream.ServerParams{
		ExecutionStore: executionStore,
		Executors:      executors,
	})

	bidder := NewBidder(cfg,
		allocatedResources,
		publishers,
		storages,
		executors,
		executionStore,
		capacityCalculator,
	)
	baseEndpoint := compute.NewBaseEndpoint(compute.BaseEndpointParams{
		ExecutionStore: executionStore,
	})

	// register debug info providers for the /debug endpoint
	debugInfoProviders := []models.DebugInfoProvider{
		runningInfoProvider,
		sensors.NewCompletedJobs(executionStore),
	}

	startup := compute.NewStartup(executionStore, bufferRunner)
	startupErr := startup.Execute(ctx)
	if startupErr != nil {
		return nil, fmt.Errorf("failed to execute compute node startup tasks: %s", startupErr)
	}

	// register compute public http apis
	compute_endpoint.NewEndpoint(compute_endpoint.EndpointParams{
		Router:             apiServer.Router,
		DebugInfoProviders: debugInfoProviders,
	})

	// node info provider
	nodeInfoProvider.RegisterNodeInfoDecorator(compute.NewNodeInfoDecorator(compute.NodeInfoDecoratorParams{
		Executors:              executors,
		Publisher:              publishers,
		Storages:               storages,
		RunningCapacityTracker: runningCapacityTracker,
		QueueCapacityTracker:   enqueuedUsageTracker,
		ExecutorBuffer:         bufferRunner,
		MaxJobRequirements:     allocatedResources,
	}))
	nodeInfoProvider.RegisterLabelProvider(capacity.NewGPULabelsProvider(allocatedResources))

	// legacyConnectionManager
	legacyConnectionManager, err := bprotocolcompute.NewConnectionManager(bprotocolcompute.Config{
		NodeID:           cfg.NodeID,
		ClientFactory:    nats.ClientFactoryFunc(transportLayer.CreateClient),
		NodeInfoProvider: nodeInfoProvider,
		HeartbeatConfig:  cfg.BacalhauConfig.Compute.Heartbeat,
		ComputeEndpoint:  baseEndpoint,
		EventStore:       executionStore.GetEventStore(),
	})
	if err != nil {
		return nil, err
	}
	if err = legacyConnectionManager.Start(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to start legacy connection manager. continuing without it")
		err = nil
	}

	// compute -> orchestrator ncl publisher
	natsConn, err := transportLayer.CreateClient(ctx)
	if err != nil {
		return nil, err
	}
	messageRegistry := MustCreateMessageRegistry()
	nclPublisher, err := ncl.NewOrderedPublisher(natsConn, ncl.OrderedPublisherConfig{
		Name:            cfg.NodeID,
		Destination:     computeOutSubject(cfg.NodeID),
		MessageRegistry: messageRegistry,
	})
	if err != nil {
		return nil, err
	}

	// orchestrator -> compute ncl subscriber
	nclSubscriber, err := ncl.NewSubscriber(natsConn, ncl.SubscriberConfig{
		Name:            cfg.NodeID,
		MessageRegistry: messageRegistry,
		MessageHandler:  compute.NewMessageHandler(executionStore),
	})
	if err != nil {
		return nil, err
	}
	if err = nclSubscriber.Subscribe(ctx, computeInSubscription(cfg.NodeID)); err != nil {
		return nil, err
	}

	watcherRegistry, nclDispatcher, err := setupComputeWatchers(
		ctx, executionStore, nclPublisher, bufferRunner, bidder)
	if err != nil {
		return nil, err
	}

	// A single Cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		if err = nclSubscriber.Close(ctx); err != nil {
			log.Error().Err(err).Msg("failed to close ncl subscriber")
		}
		if nclDispatcher != nil {
			if err = nclDispatcher.Stop(ctx); err != nil {
				log.Error().Err(err).Msg("failed to stop dispatcher")
			}
		}
		if err = watcherRegistry.Stop(ctx); err != nil {
			log.Error().Err(err).Msg("failed to stop watcher registry")
		}
		legacyConnectionManager.Stop(ctx)
		if err = executionStore.Close(ctx); err != nil {
			log.Error().Err(err).Msg("failed to close execution store")
		}
		if err = resultsPath.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close results path")
		}
	}

	return &Compute{
		ID:                 cfg.NodeID,
		LocalEndpoint:      baseEndpoint,
		LogstreamServer:    logserver,
		Capacity:           runningCapacityTracker,
		ExecutionStore:     executionStore,
		Executors:          executors,
		Storages:           storages,
		Publishers:         publishers,
		Bidder:             bidder,
		Watchers:           watcherRegistry,
		cleanupFunc:        cleanupFunc,
		debugInfoProviders: debugInfoProviders,
	}, nil
}

func createExecutionStore(ctx context.Context, cfg NodeConfig) (store.ExecutionStore, error) {
	executionStoreDBPath, err := cfg.BacalhauConfig.ExecutionStoreFilePath()
	if err != nil {
		return nil, err
	}
	executionStore, err := boltdb.NewStore(ctx, executionStoreDBPath)
	if err != nil {
		return nil, bacerrors.Wrap(err, "failed to create execution store")
	}
	return executionStore, nil
}

func (c *Compute) Cleanup(ctx context.Context) {
	c.cleanupFunc(ctx)
}

func NewBidder(
	cfg NodeConfig,
	allocatedResources models.Resources,
	publishers publisher.PublisherProvider,
	storages storage.StorageProvider,
	executors executor.ExecProvider,
	executionStore store.ExecutionStore,
	calculator capacity.UsageCalculator,
) compute.Bidder {
	var semanticBidStrats []bidstrategy.SemanticBidStrategy
	if cfg.SystemConfig.BidSemanticStrategy == nil {
		semanticBidStrats = []bidstrategy.SemanticBidStrategy{
			semantic.NewNetworkingStrategy(cfg.BacalhauConfig.JobAdmissionControl.AcceptNetworkedJobs),
			semantic.NewStatelessJobStrategy(semantic.StatelessJobStrategyParams{
				RejectStatelessJobs: cfg.BacalhauConfig.JobAdmissionControl.RejectStatelessJobs,
			}),
			semantic.NewProviderInstalledStrategy(
				publishers,
				func(j *models.Job) string { return j.Task().Publisher.Type },
			),
			semantic.NewStorageInstalledBidStrategy(storages),
			semantic.NewInputLocalityStrategy(semantic.InputLocalityStrategyParams{
				Locality: cfg.BacalhauConfig.JobAdmissionControl.Locality,
				Storages: storages,
			}),
			semantic.NewExternalCommandStrategy(semantic.ExternalCommandStrategyParams{
				Command: cfg.BacalhauConfig.JobAdmissionControl.ProbeExec,
			}),
			semantic.NewExternalHTTPStrategy(semantic.ExternalHTTPStrategyParams{
				URL: cfg.BacalhauConfig.JobAdmissionControl.ProbeHTTP,
			}),
			executor_util.NewExecutorSpecificBidStrategy(executors),
		}
	} else {
		semanticBidStrats = []bidstrategy.SemanticBidStrategy{cfg.SystemConfig.BidSemanticStrategy}
	}

	var resourceBidStrats []bidstrategy.ResourceBidStrategy
	if cfg.SystemConfig.BidResourceStrategy == nil {
		resourceBidStrats = []bidstrategy.ResourceBidStrategy{
			resource.NewMaxCapacityStrategy(resource.MaxCapacityStrategyParams{
				MaxJobRequirements: allocatedResources,
			}),
			executor_util.NewExecutorSpecificBidStrategy(executors),
		}
	} else {
		resourceBidStrats = []bidstrategy.ResourceBidStrategy{cfg.SystemConfig.BidResourceStrategy}
	}

	return compute.NewBidder(compute.BidderParams{
		SemanticStrategy: semanticBidStrats,
		ResourceStrategy: resourceBidStrats,
		UsageCalculator:  calculator,
		Store:            executionStore,
	})
}

func setupComputeWatchers(
	ctx context.Context,
	executionStore store.ExecutionStore,
	nclPublisher ncl.OrderedPublisher,
	bufferRunner *compute.ExecutorBuffer,
	bidder compute.Bidder,
) (watcher.Manager, *dispatcher.Dispatcher, error) {
	watcherRegistry := watcher.NewManager(executionStore.GetEventStore())

	// Set up execution logger watcher
	_, err := watcherRegistry.Create(ctx, computeExecutionLoggerWatcherID,
		watcher.WithHandler(watchers.NewExecutionLogger(log.Logger)),
		watcher.WithAutoStart(),
		watcher.WithInitialEventIterator(watcher.LatestIterator()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup execution logger watcher: %w", err)
	}

	// Set up execution handler watcher
	_, err = watcherRegistry.Create(ctx, computeExecutionHandlerWatcherID,
		watcher.WithHandler(watchers.NewExecutionUpsertHandler(bufferRunner, bidder)),
		watcher.WithAutoStart(),
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{compute.EventObjectExecutionUpsert},
		}),
		watcher.WithRetryStrategy(watcher.RetryStrategySkip),
		watcher.WithMaxRetries(3),
		watcher.WithInitialEventIterator(watcher.LatestIterator()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup execution handler watcher: %w", err)
	}

	// setup ncl dispatcher
	nclDispatcherWatcher, err := watcherRegistry.Create(ctx, computeNCLDispatcherWatcherID,
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{compute.EventObjectExecutionUpsert},
		}),
		watcher.WithRetryStrategy(watcher.RetryStrategyBlock),
		watcher.WithInitialEventIterator(watcher.LatestIterator()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup ncl dispatcher watcher: %w", err)
	}

	nclDispatcher, err := dispatcher.New(
		nclPublisher, nclDispatcherWatcher, watchers.NewNCLMessageCreator(), dispatcher.DefaultConfig())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dispatcher: %w", err)
	}

	if err = nclDispatcher.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start dispatcher: %w", err)
	}

	return watcherRegistry, nclDispatcher, nil
}
