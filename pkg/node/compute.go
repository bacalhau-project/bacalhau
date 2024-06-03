package node

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

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
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	compute_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/compute"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	repo_storage "github.com/bacalhau-project/bacalhau/pkg/storage/repo"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type Compute struct {
	// Visible for testing
	ID                 string
	LocalEndpoint      compute.Endpoint
	Capacity           capacity.Tracker
	ExecutionStore     store.ExecutionStore
	Executors          executor.ExecutorProvider
	Storages           storage.StorageProvider
	Bidder             compute.Bidder
	ManagementClient   *compute.ManagementClient
	cleanupFunc        func(ctx context.Context)
	nodeInfoDecorator  models.NodeInfoDecorator
	labelsProvider     models.LabelsProvider
	debugInfoProviders []model.DebugInfoProvider
}

//nolint:funlen
func NewComputeNode(
	ctx context.Context,
	nodeID string,
	cleanupManager *system.CleanupManager,
	apiServer *publicapi.Server,
	config ComputeConfig,
	storagePath string,
	repoPath string,
	storages storage.StorageProvider,
	executors executor.ExecutorProvider,
	publishers publisher.PublisherProvider,
	computeCallback compute.Callback,
	managementProxy compute.ManagementEndpoint,
	configuredLabels map[string]string,
	heartbeatClient *heartbeat.HeartbeatClient,
) (*Compute, error) {
	executionStore := config.ExecutionStore

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
		runningCapacityTracker,
		nodeID,
		executionStore,
		computeCallback,
		bufferRunner,
		apiServer,
		capacityCalculator,
	)
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

	// Node labels
	labelsProvider := models.MergeLabelsInOrder(
		&ConfigLabelsProvider{staticLabels: configuredLabels},
		&RuntimeLabelsProvider{},
		capacity.NewGPULabelsProvider(config.TotalResourceLimits),
		repo_storage.NewLabelsProvider(),
	)

	var managementClient *compute.ManagementClient
	// TODO: When we no longer use libP2P for management, we should remove this
	// as the managementProxy will always be set for NATS
	if managementProxy != nil {
		// TODO: Make the registration lock folder a config option so that we have it
		// available and don't have to depend on getting the repo folder.
		regFilename := fmt.Sprintf("%s.registration.lock", nodeID)
		regFilename = filepath.Join(repoPath, pkgconfig.ComputeStorePath, regFilename)

		// Set up the management client which will attempt to register this node
		// with the requester node, and then if successful will send regular node
		// info updates.
		managementClient = compute.NewManagementClient(&compute.ManagementClientParams{
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
	}

	// A single Cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		if managementClient != nil {
			managementClient.Stop()
		}

		executionStore.Close(ctx)
		resultsPath.Close()
	}

	return &Compute{
		ID:                 nodeID,
		LocalEndpoint:      baseEndpoint,
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
	runningCapacityTracker capacity.Tracker,
	nodeID string,
	executionStore store.ExecutionStore,
	computeCallback compute.Callback,
	bufferRunner *compute.ExecutorBuffer,
	apiServer *publicapi.Server,
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
		NodeID:           nodeID,
		SemanticStrategy: semanticBidStrats,
		ResourceStrategy: resourceBidStrats,
		Store:            executionStore,
		Callback:         computeCallback,
		Executor:         bufferRunner,
		GetApproveURL: func() *url.URL {
			return apiServer.GetURI().JoinPath("/api/v1/compute/approve")
		},
		UsageCalculator: calculator,
	})
}
