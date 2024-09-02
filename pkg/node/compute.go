package node

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/disk"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/compute/sensors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	compute_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/compute"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type Compute struct {
	// Visible for testing
	ID                 string
	LogstreamServer    logstream.Server
	Capacity           capacity.Tracker
	ExecutionStore     store.ExecutionStore
	Executors          executor.ExecutorProvider
	Storages           storage.StorageProvider
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
	nodeID string,
	apiServer *publicapi.Server,
	config ComputeConfig,
	storagePath string,
	repoPath string,
	storages storage.StorageProvider,
	executors executor.ExecutorProvider,
	publishers publisher.PublisherProvider,
	natsConn *nats.Conn,
	managementProxy compute.ManagementEndpoint,
	configuredLabels map[string]string,
	messageSerDeRegistry *ncl.MessageSerDeRegistry,
) (*Compute, error) {
	executionStore := config.ExecutionStore
	watcherRegistry := watcher.NewRegistry(executionStore.GetEventStore())

	// executor/backend
	runningCapacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: config.TotalResourceLimits,
	})
	enqueuedUsageTracker := capacity.NewLocalUsageTracker()

	resultsPath, err := compute.NewResultsPath()
	if err != nil {
		return nil, err
	}
	baseExecutor := compute.NewBaseExecutor(compute.BaseExecutorParams{
		ID:                     nodeID,
		Store:                  executionStore,
		StorageDirectory:       storagePath,
		Storages:               storages,
		Executors:              executors,
		Publishers:             publishers,
		FailureInjectionConfig: config.FailureInjectionConfig,
		ResultsPath:            *resultsPath,
	})

	bufferRunner := compute.NewExecutorBuffer(compute.ExecutorBufferParams{
		ID:                         nodeID,
		DelegateExecutor:           baseExecutor,
		Store:                      executionStore,
		RunningCapacityTracker:     runningCapacityTracker,
		EnqueuedUsageTracker:       enqueuedUsageTracker,
		DefaultJobExecutionTimeout: config.DefaultJobExecutionTimeout,
	})
	runningInfoProvider := sensors.NewRunningExecutionsInfoProvider(sensors.RunningExecutionsInfoProviderParams{
		Name:          "ActiveJobs",
		BackendBuffer: bufferRunner,
	})
	if config.LogRunningExecutionsInterval > 0 {
		loggingSensor := sensors.NewLoggingSensor(sensors.LoggingSensorParams{
			InfoProvider: runningInfoProvider,
			Interval:     config.LogRunningExecutionsInterval,
		})
		go loggingSensor.Start(ctx)
	}

	// endpoint/frontend
	capacityCalculator := capacity.NewChainedUsageCalculator(capacity.ChainedUsageCalculatorParams{
		Calculators: []capacity.UsageCalculator{
			capacity.NewDefaultsUsageCalculator(capacity.DefaultsUsageCalculatorParams{
				Defaults: config.DefaultJobResourceLimits,
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
		Buffer:         config.LogStreamBufferSize,
	})

	// node info
	nodeInfoDecorator := compute.NewNodeInfoDecorator(compute.NodeInfoDecoratorParams{
		Executors:              executors,
		Publisher:              publishers,
		Storages:               storages,
		RunningCapacityTracker: runningCapacityTracker,
		QueueCapacityTracker:   enqueuedUsageTracker,
		ExecutorBuffer:         bufferRunner,
		MaxJobRequirements:     config.JobResourceLimits,
	})

	bidder := NewBidder(config,
		publishers,
		storages,
		executors,
		executionStore,
		capacityCalculator,
	)

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
		&ConfigLabelsProvider{staticLabels: configuredLabels},
		&RuntimeLabelsProvider{},
		capacity.NewGPULabelsProvider(config.TotalResourceLimits),
	)

	computeStorePath := filepath.Join(repoPath, pkgconfig.ComputeStorePath)
	if err = os.MkdirAll(computeStorePath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create compute store directory: %s", err)
	}

	// TODO: Make the registration lock folder a config option so that we have it
	// available and don't have to depend on getting the repo folder.
	regFilename := fmt.Sprintf("%s.registration.lock", nodeID)
	regFilename = filepath.Join(computeStorePath, regFilename)

	// heartbeat client
	heartbeatPublisher, err := ncl.NewPublisher(natsConn,
		ncl.WithPublisherName(nodeID),
		ncl.WithPublisherDestination(config.ControlPlaneSettings.HeartbeatTopic),
		ncl.WithPublisherMessageSerDeRegistry(messageSerDeRegistry),
	)
	if err != nil {
		return nil, err
	}
	heartbeatClient := heartbeat.NewClient(nodeID, heartbeatPublisher)

	// Set up the management client which will attempt to register this node
	// with the requester node, and then if successful will send regular node
	// info updates.
	managementClient := compute.NewManagementClient(&compute.ManagementClientParams{
		NodeID:                   nodeID,
		LabelsProvider:           labelsProvider,
		ManagementProxy:          managementProxy,
		NodeInfoDecorator:        nodeInfoDecorator,
		RegistrationFilePath:     regFilename,
		AvailableCapacityTracker: runningCapacityTracker,
		QueueUsageTracker:        enqueuedUsageTracker,
		HeartbeatClient:          heartbeatClient,
		ControlPlaneSettings:     config.ControlPlaneSettings,
	})
	if err := managementClient.RegisterNode(ctx); err != nil {
		return nil, fmt.Errorf("failed to register node with requester: %s", err)
	}
	go managementClient.Start(ctx)

	// compute -> orchestrator ncl publisher
	nclPublisher, err := ncl.NewPublisher(natsConn,
		ncl.WithPublisherName(nodeID),
		ncl.WithPublisherDestinationPrefix(computeOutSubject(nodeID)),
		ncl.WithPublisherMessageSerDeRegistry(messageSerDeRegistry),
	)
	if err != nil {
		return nil, err
	}

	// TODO: Add checkpointing or else events will be missed
	_, err = watcherRegistry.Watch(ctx, executionForwarderWatcherID, compute.NewForwarder(nclPublisher),
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{compute.EventObjectExecutionUpsert},
		}),
		watcher.WithRetryStrategy(watcher.RetryStrategySkip),
		watcher.WithMaxRetries(3),
		watcher.WithInitialEventIterator(watcher.LatestIterator()))
	if err != nil {
		return nil, err
	}

	// orchestrator -> compute ncl subscriber
	nclSubscriber, err := ncl.NewSubscriber(natsConn,
		ncl.WithSubscriberMessageSerDeRegistry(messageSerDeRegistry),
		ncl.WithSubscriberMessageHandlers(compute.NewMessageHandler(executionStore)),
	)
	if err != nil {
		return nil, err
	}
	if err = nclSubscriber.Subscribe(computeInSubscription(nodeID)); err != nil {
		return nil, err
	}

	// TODO: Add checkpointing or else events will be missed
	executionHandler := compute.NewExecutionUpsertHandler(bufferRunner, bidder)
	_, err = watcherRegistry.Watch(ctx, executionHandlerWatcherID, executionHandler,
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{compute.EventObjectExecutionUpsert},
		}),
		watcher.WithRetryStrategy(watcher.RetryStrategySkip),
		watcher.WithMaxRetries(3),
		watcher.WithInitialEventIterator(watcher.LatestIterator()))
	if err != nil {
		return nil, err
	}

	// A single Cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		if err = nclSubscriber.Close(ctx); err != nil {
			log.Error().Err(err).Msg("failed to close ncl subscriber")
		}
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
		ID:                 nodeID,
		LogstreamServer:    logserver,
		Capacity:           runningCapacityTracker,
		ExecutionStore:     executionStore,
		Executors:          executors,
		Storages:           storages,
		Bidder:             bidder,
		cleanupFunc:        cleanupFunc,
		nodeInfoDecorator:  nodeInfoDecorator,
		labelsProvider:     labelsProvider,
		debugInfoProviders: debugInfoProviders,
		ManagementClient:   managementClient,
	}, nil
}

func (c *Compute) Cleanup(ctx context.Context) {
	c.cleanupFunc(ctx)
}

func NewBidder(
	config ComputeConfig,
	publishers publisher.PublisherProvider,
	storages storage.StorageProvider,
	executors executor.ExecutorProvider,
	executionStore store.ExecutionStore,
	calculator capacity.UsageCalculator,
) compute.Bidder {
	var semanticBidStrats []bidstrategy.SemanticBidStrategy
	if config.BidSemanticStrategy == nil {
		semanticBidStrats = []bidstrategy.SemanticBidStrategy{
			semantic.NewNetworkingStrategy(config.JobSelectionPolicy.AcceptNetworkedJobs),
			semantic.NewTimeoutStrategy(semantic.TimeoutStrategyParams{
				MaxJobExecutionTimeout:                config.MaxJobExecutionTimeout,
				MinJobExecutionTimeout:                config.MinJobExecutionTimeout,
				JobExecutionTimeoutClientIDBypassList: config.JobExecutionTimeoutClientIDBypassList,
			}),
			semantic.NewStatelessJobStrategy(semantic.StatelessJobStrategyParams{
				RejectStatelessJobs: config.JobSelectionPolicy.RejectStatelessJobs,
			}),
			semantic.NewProviderInstalledStrategy(
				publishers,
				func(j *models.Job) string { return j.Task().Publisher.Type },
			),
			semantic.NewStorageInstalledBidStrategy(storages),
			semantic.NewInputLocalityStrategy(semantic.InputLocalityStrategyParams{
				Locality: config.JobSelectionPolicy.Locality,
				Storages: storages,
			}),
			semantic.NewExternalCommandStrategy(semantic.ExternalCommandStrategyParams{
				Command: config.JobSelectionPolicy.ProbeExec,
			}),
			semantic.NewExternalHTTPStrategy(semantic.ExternalHTTPStrategyParams{
				URL: config.JobSelectionPolicy.ProbeHTTP,
			}),
			executor_util.NewExecutorSpecificBidStrategy(executors),
		}
	} else {
		semanticBidStrats = []bidstrategy.SemanticBidStrategy{config.BidSemanticStrategy}
	}

	var resourceBidStrats []bidstrategy.ResourceBidStrategy
	if config.BidResourceStrategy == nil {
		resourceBidStrats = []bidstrategy.ResourceBidStrategy{
			resource.NewMaxCapacityStrategy(resource.MaxCapacityStrategyParams{
				MaxJobRequirements: config.JobResourceLimits,
			}),
			executor_util.NewExecutorSpecificBidStrategy(executors),
		}
	} else {
		resourceBidStrats = []bidstrategy.ResourceBidStrategy{config.BidResourceStrategy}
	}

	return compute.NewBidder(compute.BidderParams{
		SemanticStrategy: semanticBidStrats,
		ResourceStrategy: resourceBidStrats,
		UsageCalculator:  calculator,
		Store:            executionStore,
	})
}
