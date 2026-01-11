package node

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/disk"
	"github.com/bacalhau-project/bacalhau/pkg/compute/env"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/compute/sensors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/compute/watchers"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
	bprotocolcompute "github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol/compute"
	nclprotocolcompute "github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/compute"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/dispatcher"
)

type Compute struct {
	// Visible for testing
	ID                 string
	LocalEndpoint      compute.Endpoint
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

//nolint:funlen
func NewComputeNode(
	ctx context.Context,
	cfg NodeConfig,
	apiServer *publicapi.Server,
	clientFactory nats.ClientFactory,
	nodeInfoProvider models.DecoratorNodeInfoProvider,
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

	resultsPath, err := compute.NewResultsPath(executionDir)
	if err != nil {
		return nil, err
	}

	// Create environment variable resolver to be used by both bidder and executor
	envResolver := env.NewResolver(env.ResolverParams{
		AllowList: cfg.BacalhauConfig.Compute.Env.AllowList,
	})

	portAllocator, err := compute.NewPortAllocator(
		cfg.BacalhauConfig.Compute.Network.PortRangeStart,
		cfg.BacalhauConfig.Compute.Network.PortRangeEnd,
	)
	if err != nil {
		return nil, err
	}

	// We set the default network type if the node rejects network jobs.
	// Otherwise, we let each executor set the proper network type if not explicitly defined.
	// - docker: sets the default as bridge, since it is supported across multiple platforms
	// - wasm: sets the default as host, since it doesn't support bridge mode
	defaultNetworkType := models.NetworkDefault
	if cfg.BacalhauConfig.JobAdmissionControl.RejectNetworkedJobs {
		defaultNetworkType = models.NetworkNone
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
		EnvResolver:            envResolver,
		PortAllocator:          portAllocator,
		DefaultNetworkType:     defaultNetworkType,
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

	bidder := NewBidder(cfg,
		allocatedResources,
		publishers,
		storages,
		executors,
		executionStore,
		capacityCalculator,
		envResolver,
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

	// Get the address this node should advertise
	// TODO: attempt to auto-detect the address if not provided
	address := cfg.BacalhauConfig.Compute.Network.AdvertisedAddress

	// node info provider
	nodeInfoProvider.RegisterNodeInfoDecorator(compute.NewNodeInfoDecorator(compute.NodeInfoDecoratorParams{
		Executors:              executors,
		Publisher:              publishers,
		Storages:               storages,
		RunningCapacityTracker: runningCapacityTracker,
		QueueCapacityTracker:   enqueuedUsageTracker,
		ExecutorBuffer:         bufferRunner,
		MaxJobRequirements:     allocatedResources,
		AdvertisedAddress:      address,
	}))
	nodeInfoProvider.RegisterLabelProvider(capacity.NewGPULabelsProvider(allocatedResources))

	// legacyConnectionManager
	legacyConnectionManager, err := bprotocolcompute.NewConnectionManager(bprotocolcompute.Config{
		NodeID:           cfg.NodeID,
		ClientFactory:    clientFactory,
		NodeInfoProvider: nodeInfoProvider,
		HeartbeatConfig:  cfg.BacalhauConfig.Compute.Heartbeat,
		ComputeEndpoint:  baseEndpoint,
		EventStore:       executionStore.GetEventStore(),
	})
	if err != nil {
		return nil, err
	}

	// connection manager
	connectionManager, err := nclprotocolcompute.NewConnectionManager(nclprotocolcompute.Config{
		NodeID:                  cfg.NodeID,
		ClientFactory:           clientFactory,
		NodeInfoProvider:        nodeInfoProvider,
		HeartbeatInterval:       cfg.BacalhauConfig.Compute.Heartbeat.Interval.AsTimeDuration(),
		NodeInfoUpdateInterval:  cfg.BacalhauConfig.Compute.Heartbeat.InfoUpdateInterval.AsTimeDuration(),
		DataPlaneMessageHandler: compute.NewMessageHandler(executionStore),
		DataPlaneMessageCreator: watchers.NewNCLMessageCreator(),
		EventStore:              executionStore.GetEventStore(),
		Checkpointer:            executionStore,
		DispatcherConfig:        dispatcher.DefaultConfig(),
		LogStreamServer: logstream.NewServer(logstream.ServerParams{
			ExecutionStore: executionStore,
			Executors:      executors,
			ResultsPath:    *resultsPath,
		}),
	})
	if err != nil {
		return nil, err
	}
	cfg.DependencyInjector.LazyPublisherProvider.SetProvider(connectionManager)

	// First we attempt to start the legacy connection manager to maintain backward compatibility
	// with older orchestrator nodes. If it fails, or we receive an upgrade available message, we
	// start the new connection manager.
	if err = legacyConnectionManager.Start(ctx); err != nil {
		if strings.Contains(err.Error(), bprotocol.ErrUpgradeAvailable.Error()) {
			log.Debug().Msg("Disabling bprotocol management client due to upgrade available")
		} else {
			log.Warn().Err(err).Msg("failed to start legacy connection manager. falling back to ncl protocol")
		}

		if err = connectionManager.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start connection manager: %w", err)
		}
	}

	watcherRegistry, err := setupComputeWatchers(
		ctx, executionStore, bufferRunner, bidder)
	if err != nil {
		return nil, err
	}

	// A single Cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		if err = watcherRegistry.Stop(ctx); err != nil {
			log.Error().Err(err).Msg("failed to stop watcher registry")
		}
		legacyConnectionManager.Stop(ctx)
		if err = connectionManager.Close(ctx); err != nil {
			log.Error().Err(err).Msg("failed to stop connection manager")
		}
		if err = executionStore.Close(ctx); err != nil {
			log.Error().Err(err).Msg("failed to close execution store")
		}
		// TODO: Remove this behaviour once we have proper execution metadata garbage collection.
		if err = resultsPath.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close results path")
		}
	}

	return &Compute{
		ID:                 cfg.NodeID,
		LocalEndpoint:      baseEndpoint,
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
	envResolver compute.EnvVarResolver,
) compute.Bidder {
	var semanticBidStrats []bidstrategy.SemanticBidStrategy
	if cfg.SystemConfig.BidSemanticStrategy == nil {
		semanticBidStrats = []bidstrategy.SemanticBidStrategy{
			semantic.NewNetworkingStrategy(cfg.BacalhauConfig.JobAdmissionControl.RejectNetworkedJobs),
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
			semantic.NewEnvResolverStrategy(semantic.EnvResolverStrategyParams{
				Resolver: envResolver,
			}),
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
	bufferRunner *compute.ExecutorBuffer,
	bidder compute.Bidder,
) (watcher.Manager, error) {
	watcherRegistry := watcher.NewManager(executionStore.GetEventStore())

	// Set up execution logger watcher
	_, err := watcherRegistry.Create(ctx, computeExecutionLoggerWatcherID,
		watcher.WithHandler(watchers.NewExecutionLogger(log.Logger)),
		watcher.WithEphemeral(),
		watcher.WithAutoStart(),
		watcher.WithInitialEventIterator(watcher.LatestIterator()))
	if err != nil {
		return nil, fmt.Errorf("failed to setup execution logger watcher: %w", err)
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
		return nil, fmt.Errorf("failed to setup execution handler watcher: %w", err)
	}

	return watcherRegistry, nil
}
