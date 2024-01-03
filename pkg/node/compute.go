package node

import (
	"context"
	"fmt"
	"net/url"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/disk"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/compute/sensors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	compute_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/compute"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	repo_storage "github.com/bacalhau-project/bacalhau/pkg/storage/repo"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/libp2p/go-libp2p/core/host"
)

type Compute struct {
	// Visible for testing
	ID                 string
	LocalEndpoint      compute.Endpoint
	Capacity           capacity.Tracker
	ExecutionStore     store.ExecutionStore
	Executors          executor.ExecutorProvider
	Storages           storage.StorageProvider
	LogServer          *logstream.LogStreamServer
	Bidder             compute.Bidder
	cleanupFunc        func(ctx context.Context)
	nodeInfoDecorator  models.NodeInfoDecorator
	autoLabelsProvider models.LabelsProvider
}

//nolint:funlen
func NewComputeNode(
	ctx context.Context,
	nodeID string,
	cleanupManager *system.CleanupManager,
	host host.Host,
	apiServer *publicapi.Server,
	config ComputeConfig,
	storagePath string,
	storages storage.StorageProvider,
	executors executor.ExecutorProvider,
	publishers publisher.PublisherProvider,
	fsRepo *repo.FsRepo,
	computeCallback compute.Callback,
) (*Compute, error) {
	var executionStore store.ExecutionStore
	// create the execution store
	if config.ExecutionStore == nil {
		var err error
		executionStore, err = fsRepo.InitExecutionStore(ctx, nodeID)
		if err != nil {
			return nil, err
		}
	} else {
		executionStore = config.ExecutionStore
	}

	// executor/backend
	runningCapacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: config.TotalResourceLimits,
	})
	enqueuedCapacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: config.QueueResourceLimits,
	})

	resultsPath, err := compute.NewResultsPath()
	if err != nil {
		return nil, err
	}
	baseExecutor := compute.NewBaseExecutor(compute.BaseExecutorParams{
		ID:                     nodeID,
		Callback:               computeCallback,
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
		Callback:                   computeCallback,
		RunningCapacityTracker:     runningCapacityTracker,
		EnqueuedCapacityTracker:    enqueuedCapacityTracker,
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
		loggingCtx, cancel := context.WithCancel(ctx)
		cleanupManager.RegisterCallback(func() error {
			cancel()
			return nil
		})
		go loggingSensor.Start(loggingCtx)
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

	semanticBidStrat := bidstrategy.WithSemantics(config.BidSemanticStrategy)
	if config.BidSemanticStrategy == nil {
		semanticBidStrat = bidstrategy.WithSemantics(
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
		)
	}

	resourceBidStrat := bidstrategy.WithResources(config.BidResourceStrategy)
	if config.BidResourceStrategy == nil {
		resourceBidStrat = bidstrategy.WithResources(
			resource.NewMaxCapacityStrategy(resource.MaxCapacityStrategyParams{
				MaxJobRequirements: config.JobResourceLimits,
			}),
			resource.NewAvailableCapacityStrategy(ctx, resource.AvailableCapacityStrategyParams{
				RunningCapacityTracker:  runningCapacityTracker,
				EnqueuedCapacityTracker: enqueuedCapacityTracker,
			}),
			executor_util.NewExecutorSpecificBidStrategy(executors),
		)
	}

	// logging server
	// TODO: make logging server agnostic to libp2p transport
	var logserver *logstream.LogStreamServer
	if host != nil {
		logserver = logstream.NewLogStreamServer(logstream.LogStreamServerOptions{
			Ctx:            ctx,
			Host:           host,
			ExecutionStore: executionStore,
			//
			Executors: executors,
		})
		_, loggingCancel := context.WithCancel(ctx)
		cleanupManager.RegisterCallback(func() error {
			loggingCancel()
			return nil
		})
	}

	// node info
	nodeInfoDecorator := compute.NewNodeInfoDecorator(compute.NodeInfoDecoratorParams{
		Executors:          executors,
		Publisher:          publishers,
		Storages:           storages,
		CapacityTracker:    runningCapacityTracker,
		ExecutorBuffer:     bufferRunner,
		MaxJobRequirements: config.JobResourceLimits,
	})

	bidStrat := bidstrategy.NewChainedBidStrategy(semanticBidStrat, resourceBidStrat)
	bidder := compute.NewBidder(compute.BidderParams{
		NodeID:           nodeID,
		SemanticStrategy: bidStrat,
		ResourceStrategy: bidStrat,
		Store:            executionStore,
		Callback:         computeCallback,
		Executor:         bufferRunner,
		GetApproveURL: func() *url.URL {
			return apiServer.GetURI().JoinPath("/api/v1/compute/approve")
		},
	})

	baseEndpoint := compute.NewBaseEndpoint(compute.BaseEndpointParams{
		ID:              nodeID,
		ExecutionStore:  executionStore,
		UsageCalculator: capacityCalculator,
		Bidder:          bidder,
		Executor:        bufferRunner,
		LogServer:       logserver,
	})

	// register debug info providers for the /debug endpoint
	debugInfoProviders := []model.DebugInfoProvider{
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
		Bidder:             bidder,
		Store:              executionStore,
		DebugInfoProviders: debugInfoProviders,
	})

	// A single cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		executionStore.Close(ctx)
		resultsPath.Close()
	}

	// Node labels
	labelsProvider := models.MergeLabelsInOrder(
		&RuntimeLabelsProvider{},
		capacity.NewGPULabelsProvider(config.TotalResourceLimits),
		repo_storage.NewLabelsProvider(),
	)

	return &Compute{
		ID:                 nodeID,
		LocalEndpoint:      baseEndpoint,
		Capacity:           runningCapacityTracker,
		ExecutionStore:     executionStore,
		Executors:          executors,
		Storages:           storages,
		Bidder:             bidder,
		LogServer:          logserver,
		cleanupFunc:        cleanupFunc,
		nodeInfoDecorator:  nodeInfoDecorator,
		autoLabelsProvider: labelsProvider,
	}, nil
}

func (c *Compute) cleanup(ctx context.Context) {
	c.cleanupFunc(ctx)
}
