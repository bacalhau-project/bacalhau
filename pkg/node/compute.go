package node

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/nats-io/nats.go"
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
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher/handlers"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	compute_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/compute"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

var (
	// TODO: remove compute level resource defaults. This should be handled at the orchestrator,
	//  but we still need to validate the behaviour is a job without resource limits land on a compute node
	defaultJobResourceLimits = models.Resources{
		CPU:    0.1,               // 100m
		Memory: 100 * 1024 * 1024, // 100Mi
	}
)

type Compute struct {
	// Visible for testing
	ID                 string
	LocalEndpoint      compute.Endpoint
	Capacity           capacity.Tracker
	ExecutionStore     store.ExecutionStore
	Executors          executor.ExecutorProvider
	Storages           storage.StorageProvider
	Publishers         publisher.PublisherProvider
	Bidder             compute.Bidder
	ManagementClient   *compute.ManagementClient
	cleanupFunc        func(ctx context.Context)
	nodeInfoDecorator  models.NodeInfoDecorator
	labelsProvider     models.LabelsProvider
	debugInfoProviders []models.DebugInfoProvider
}

//nolint:funlen
func NewComputeNode(
	ctx context.Context,
	cfg NodeConfig,
	apiServer *publicapi.Server,
	natsConn *nats.Conn,
	computeCallback compute.Callback,
	managementProxy compute.ManagementEndpoint,
	messageSerDeRegistry *ncl.MessageSerDeRegistry,
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
	watcherRegistry := watcher.NewRegistry(executionStore.GetEventStore())
	_, err = watcherRegistry.Watch(ctx, "compute-logger", handlers.NewLoggingHandler(log.Logger),
		watcher.WithInitialEventIterator(watcher.LatestIterator()))
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
		Callback:               computeCallback,
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
		Callback:               computeCallback,
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
				Defaults: defaultJobResourceLimits,
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

	// node info
	nodeInfoDecorator := compute.NewNodeInfoDecorator(compute.NodeInfoDecoratorParams{
		Executors:              executors,
		Publisher:              publishers,
		Storages:               storages,
		RunningCapacityTracker: runningCapacityTracker,
		QueueCapacityTracker:   enqueuedUsageTracker,
		ExecutorBuffer:         bufferRunner,
		MaxJobRequirements:     allocatedResources,
	})

	bidder := NewBidder(cfg,
		allocatedResources,
		publishers,
		storages,
		executors,
		executionStore,
		computeCallback,
		bufferRunner,
		capacityCalculator,
	)
	baseEndpoint := compute.NewBaseEndpoint(compute.BaseEndpointParams{
		ID:              cfg.NodeID,
		ExecutionStore:  executionStore,
		UsageCalculator: capacityCalculator,
		Bidder:          bidder,
		Executor:        bufferRunner,
		LogServer:       logserver,
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

	// Node labels
	labelsProvider := models.MergeLabelsInOrder(
		&ConfigLabelsProvider{staticLabels: cfg.BacalhauConfig.Labels},
		&RuntimeLabelsProvider{},
		capacity.NewGPULabelsProvider(allocatedResources),
	)

	// TODO: Make the registration lock folder a config option so that we have it
	// available and don't have to depend on getting the repo folder.
	computeDir, err := cfg.BacalhauConfig.ComputeDir()
	if err != nil {
		return nil, err
	}
	regFilename := fmt.Sprintf("%s.registration.lock", cfg.NodeID)
	regFilename = filepath.Join(computeDir, regFilename)

	// heartbeat client
	heartbeatPublisher, err := ncl.NewPublisher(natsConn,
		ncl.WithPublisherName(cfg.NodeID),
		ncl.WithPublisherDestination(computeHeartbeatTopic(cfg.NodeID)),
		ncl.WithPublisherMessageSerDeRegistry(messageSerDeRegistry),
	)
	if err != nil {
		return nil, err
	}
	heartbeatClient, err := heartbeat.NewClient(natsConn, cfg.NodeID, heartbeatPublisher)
	if err != nil {
		return nil, err
	}

	// Set up the management client which will attempt to register this node
	// with the requester node, and then if successful will send regular node
	// info updates.
	managementClient := compute.NewManagementClient(&compute.ManagementClientParams{
		NodeID:                   cfg.NodeID,
		LabelsProvider:           labelsProvider,
		ManagementProxy:          managementProxy,
		NodeInfoDecorator:        nodeInfoDecorator,
		RegistrationFilePath:     regFilename,
		AvailableCapacityTracker: runningCapacityTracker,
		QueueUsageTracker:        enqueuedUsageTracker,
		HeartbeatClient:          heartbeatClient,
		HeartbeatConfig:          cfg.BacalhauConfig.Compute.Heartbeat,
	})
	if err := managementClient.RegisterNode(ctx); err != nil {
		return nil, fmt.Errorf("failed to register node with requester: %s", err)
	}
	go managementClient.Start(ctx)

	// A single Cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		if err = watcherRegistry.Stop(ctx); err != nil {
			log.Error().Err(err).Msg("failed to stop watcher registry")
		}
		managementClient.Stop()
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
		Capacity:           runningCapacityTracker,
		ExecutionStore:     executionStore,
		Executors:          executors,
		Storages:           storages,
		Publishers:         publishers,
		Bidder:             bidder,
		cleanupFunc:        cleanupFunc,
		nodeInfoDecorator:  nodeInfoDecorator,
		labelsProvider:     labelsProvider,
		debugInfoProviders: debugInfoProviders,
		ManagementClient:   managementClient,
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
	executors executor.ExecutorProvider,
	executionStore store.ExecutionStore,
	computeCallback compute.Callback,
	bufferRunner *compute.ExecutorBuffer,
	calculator capacity.UsageCalculator,
) compute.Bidder {
	semanticBidStrats := []bidstrategy.SemanticBidStrategy{
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

	resourceBidStrats := []bidstrategy.ResourceBidStrategy{
		resource.NewMaxCapacityStrategy(resource.MaxCapacityStrategyParams{
			MaxJobRequirements: allocatedResources,
		}),
		executor_util.NewExecutorSpecificBidStrategy(executors),
	}

	return compute.NewBidder(compute.BidderParams{
		NodeID:           cfg.NodeID,
		SemanticStrategy: semanticBidStrats,
		ResourceStrategy: resourceBidStrats,
		Store:            executionStore,
		Callback:         computeCallback,
		Executor:         bufferRunner,
		UsageCalculator:  calculator,
	})
}
