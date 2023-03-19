package node

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	compute_bidstrategies "github.com/bacalhau-project/bacalhau/pkg/compute/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/disk"
	compute_publicapi "github.com/bacalhau-project/bacalhau/pkg/compute/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/compute/sensors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inlocalstore"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/simulator"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	storage_bidstrategy "github.com/bacalhau-project/bacalhau/pkg/storage/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
	simulator_protocol "github.com/bacalhau-project/bacalhau/pkg/transport/simulator"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"github.com/libp2p/go-libp2p/core/host"
)

type Compute struct {
	// Visible for testing
	LocalEndpoint       compute.Endpoint
	Capacity            capacity.Tracker
	ExecutionStore      store.ExecutionStore
	Executors           executor.ExecutorProvider
	computeCallback     *bprotocol.CallbackProxy
	cleanupFunc         func(ctx context.Context)
	computeInfoProvider model.ComputeNodeInfoProvider
}

//nolint:funlen
func NewComputeNode(
	ctx context.Context,
	cleanupManager *system.CleanupManager,
	host host.Host,
	apiServer *publicapi.APIServer,
	config ComputeConfig,
	simulatorNodeID string,
	simulatorRequestHandler *simulator.RequestHandler,
	storages storage.StorageProvider,
	executors executor.ExecutorProvider,
	verifiers verifier.VerifierProvider,
	publishers publisher.PublisherProvider) (*Compute, error) {
	executionStore := inlocalstore.NewPersistentJobStore(inmemory.NewStore())

	// executor/backend
	runningCapacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: config.TotalResourceLimits,
	})
	enqueuedCapacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: config.QueueResourceLimits,
	})

	// Callback to send compute events (i.e. requester endpoint)
	var computeCallback compute.Callback
	standardComputeCallback := bprotocol.NewCallbackProxy(bprotocol.CallbackProxyParams{
		Host: host,
	})
	if simulatorNodeID != "" {
		simulatorProxy := simulator_protocol.NewCallbackProxy(simulator_protocol.CallbackProxyParams{
			SimulatorNodeID: simulatorNodeID,
			Host:            host,
		})
		if simulatorRequestHandler != nil {
			// if this node is the simulator node, we need to register a local callback to allow self dialing
			simulatorProxy.RegisterLocalComputeCallback(simulatorRequestHandler)
			// set standard callback implementation so that the simulator can forward requests to the correct endpoints
			// after it finishes its validation and processing of the request
			simulatorRequestHandler.SetRequesterProxy(standardComputeCallback)
		}
		computeCallback = simulatorProxy
	} else {
		computeCallback = standardComputeCallback
	}

	baseExecutor := compute.NewBaseExecutor(compute.BaseExecutorParams{
		ID:              host.ID().String(),
		Callback:        computeCallback,
		Store:           executionStore,
		Executors:       executors,
		Verifiers:       verifiers,
		Publishers:      publishers,
		SimulatorConfig: config.SimulatorConfig,
	})

	bufferRunner := compute.NewExecutorBuffer(compute.ExecutorBufferParams{
		ID:                         host.ID().String(),
		DelegateExecutor:           baseExecutor,
		Callback:                   computeCallback,
		RunningCapacityTracker:     runningCapacityTracker,
		EnqueuedCapacityTracker:    enqueuedCapacityTracker,
		DefaultJobExecutionTimeout: config.DefaultJobExecutionTimeout,
		BackoffDuration:            config.ExecutorBufferBackoffDuration,
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
				Executors: executors,
			}),
		},
	})

	biddingStrategy := config.BidStrategy
	if biddingStrategy == nil {
		biddingStrategy = bidstrategy.NewChainedBidStrategy(
			bidstrategy.FromJobSelectionPolicy(config.JobSelectionPolicy),
			compute_bidstrategies.NewMaxCapacityStrategy(compute_bidstrategies.MaxCapacityStrategyParams{
				MaxJobRequirements: config.JobResourceLimits,
			}),
			compute_bidstrategies.NewAvailableCapacityStrategy(ctx, compute_bidstrategies.AvailableCapacityStrategyParams{
				RunningCapacityTracker:  runningCapacityTracker,
				EnqueuedCapacityTracker: enqueuedCapacityTracker,
			}),
			// TODO XXX: don't hardcode networkSize, calculate this dynamically from
			//  libp2p instead somehow. https://github.com/bacalhau-project/bacalhau/issues/512
			bidstrategy.NewDistanceDelayStrategy(bidstrategy.DistanceDelayStrategyParams{
				NetworkSize: 1,
			}),
			executor_util.NewExecutorSpecificBidStrategy(executors),
			executor_util.NewInputLocalityStrategy(executor_util.InputLocalityStrategyParams{
				Locality:  config.JobSelectionPolicy.Locality,
				Executors: executors,
			}),
			bidstrategy.NewProviderInstalledStrategy[model.Verifier, verifier.Verifier](
				verifiers,
				func(j *model.Job) model.Verifier { return j.Spec.Verifier },
			),
			bidstrategy.NewProviderInstalledStrategy[model.Publisher, publisher.Publisher](
				publishers,
				func(j *model.Job) model.Publisher { return j.Spec.Publisher },
			),
			storage_bidstrategy.NewStorageInstalledBidStrategy(storages),
			bidstrategy.NewTimeoutStrategy(bidstrategy.TimeoutStrategyParams{
				MaxJobExecutionTimeout:                config.MaxJobExecutionTimeout,
				MinJobExecutionTimeout:                config.MinJobExecutionTimeout,
				JobExecutionTimeoutClientIDBypassList: config.JobExecutionTimeoutClientIDBypassList,
			}),
		)
	}

	// node info
	nodeInfoProvider := compute.NewNodeInfoProvider(compute.NodeInfoProviderParams{
		Executors:          executors,
		CapacityTracker:    runningCapacityTracker,
		ExecutorBuffer:     bufferRunner,
		MaxJobRequirements: config.JobResourceLimits,
	})

	baseEndpoint := compute.NewBaseEndpoint(compute.BaseEndpointParams{
		ID:              host.ID().String(),
		ExecutionStore:  executionStore,
		UsageCalculator: capacityCalculator,
		BidStrategy:     biddingStrategy,
		Executor:        bufferRunner,
	})

	// if this node is the simulator, then we set the simulator request handler as the stream handler
	if simulatorRequestHandler != nil {
		bprotocol.NewComputeHandler(bprotocol.ComputeHandlerParams{
			Host:            host,
			ComputeEndpoint: simulatorRequestHandler,
		})
	} else {
		bprotocol.NewComputeHandler(bprotocol.ComputeHandlerParams{
			Host:            host,
			ComputeEndpoint: baseEndpoint,
		})
	}

	// register debug info providers for the /debug endpoint
	debugInfoProviders := []model.DebugInfoProvider{
		runningInfoProvider,
		sensors.NewCompletedJobs(executionStore),
	}

	// register compute public http apis
	computeAPIServer := compute_publicapi.NewComputeAPIServer(compute_publicapi.ComputeAPIServerParams{
		APIServer:          apiServer,
		DebugInfoProviders: debugInfoProviders,
	})
	err := computeAPIServer.RegisterAllHandlers()
	if err != nil {
		return nil, err
	}

	// A single cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		// pass
	}

	return &Compute{
		LocalEndpoint:       baseEndpoint,
		Capacity:            runningCapacityTracker,
		ExecutionStore:      executionStore,
		Executors:           executors,
		computeCallback:     standardComputeCallback,
		cleanupFunc:         cleanupFunc,
		computeInfoProvider: nodeInfoProvider,
	}, nil
}

func (c *Compute) RegisterLocalComputeCallback(callback compute.Callback) {
	c.computeCallback.RegisterLocalComputeCallback(callback)
}

func (c *Compute) cleanup(ctx context.Context) {
	c.cleanupFunc(ctx)
}
