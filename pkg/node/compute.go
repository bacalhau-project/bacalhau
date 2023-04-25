package node

import (
	"context"
	"net/url"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/host"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/disk"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
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
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
	simulator_protocol "github.com/bacalhau-project/bacalhau/pkg/transport/simulator"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
)

type Compute struct {
	// Visible for testing
	ID                  string
	LocalEndpoint       compute.Endpoint
	Capacity            capacity.Tracker
	ExecutionStore      store.ExecutionStore
	Executors           executor.ExecutorProvider
	LogServer           *logstream.LogStreamServer
	Bidder              compute.Bidder
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

	var executionStore store.ExecutionStore
	// create the execution store
	if config.ExecutionStore == nil {
		var err error
		executionStore, err = createExecutionStore(host)
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

	semanticBidStrat := config.BidSemanticStrategy
	if semanticBidStrat == nil {
		semanticBidStrat = semantic.NewChainedSemanticBidStrategy(
			executor_util.NewExecutorSpecificBidStrategy(executors),
			semantic.FromJobSelectionPolicy(config.JobSelectionPolicy),
			semantic.NewInputLocalityStrategy(semantic.InputLocalityStrategyParams{
				Locality:  config.JobSelectionPolicy.Locality,
				Executors: executors,
			}),
			semantic.NewProviderInstalledStrategy[model.Verifier, verifier.Verifier](
				verifiers,
				func(j *model.Job) model.Verifier { return j.Spec.Verifier },
			),
			semantic.NewProviderInstalledStrategy[model.Publisher, publisher.Publisher](
				publishers,
				func(j *model.Job) model.Publisher { return j.Spec.PublisherSpec.Type },
			),
			semantic.NewStorageInstalledBidStrategy(storages),
			semantic.NewTimeoutStrategy(semantic.TimeoutStrategyParams{
				MaxJobExecutionTimeout:                config.MaxJobExecutionTimeout,
				MinJobExecutionTimeout:                config.MinJobExecutionTimeout,
				JobExecutionTimeoutClientIDBypassList: config.JobExecutionTimeoutClientIDBypassList,
			}),
			// TODO XXX: don't hardcode networkSize, calculate this dynamically from
			//  libp2p instead somehow. https://github.com/bacalhau-project/bacalhau/issues/512
			semantic.NewDistanceDelayStrategy(semantic.DistanceDelayStrategyParams{
				NetworkSize: 1,
			}),
		)
	}

	resourceBidStrat := config.BidResourceStrategy
	if resourceBidStrat == nil {
		resourceBidStrat = resource.NewChainedResourceBidStrategy(
			executor_util.NewExecutorSpecificBidStrategy(executors),
			resource.NewMaxCapacityStrategy(resource.MaxCapacityStrategyParams{
				MaxJobRequirements: config.JobResourceLimits,
			}),
			resource.NewAvailableCapacityStrategy(ctx, resource.AvailableCapacityStrategyParams{
				RunningCapacityTracker:  runningCapacityTracker,
				EnqueuedCapacityTracker: enqueuedCapacityTracker,
			}),
		)
	}

	// logging server
	logserver := logstream.NewLogStreamServer(logstream.LogStreamServerOptions{
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

	// node info
	nodeInfoProvider := compute.NewNodeInfoProvider(compute.NodeInfoProviderParams{
		Executors:          executors,
		Verifiers:          verifiers,
		Publisher:          publishers,
		Storages:           storages,
		CapacityTracker:    runningCapacityTracker,
		ExecutorBuffer:     bufferRunner,
		MaxJobRequirements: config.JobResourceLimits,
	})

	bidder := compute.NewBidder(compute.BidderParams{
		NodeID:           host.ID().String(),
		SemanticStrategy: semanticBidStrat,
		ResourceStrategy: resourceBidStrat,
		Store:            executionStore,
		Callback:         computeCallback,
		GetApproveURL: func() *url.URL {
			return apiServer.GetURI().JoinPath(compute_publicapi.APIPrefix, compute_publicapi.APIApproveSuffix)
		},
	})

	baseEndpoint := compute.NewBaseEndpoint(compute.BaseEndpointParams{
		ID:              host.ID().String(),
		ExecutionStore:  executionStore,
		UsageCalculator: capacityCalculator,
		Bidder:          bidder,
		Executor:        bufferRunner,
		LogServer:       *logserver,
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
		Bidder:             bidder,
		Store:              executionStore,
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
		ID:                  host.ID().String(),
		LocalEndpoint:       baseEndpoint,
		Capacity:            runningCapacityTracker,
		ExecutionStore:      executionStore,
		Executors:           executors,
		Bidder:              bidder,
		LogServer:           logserver,
		computeCallback:     standardComputeCallback,
		cleanupFunc:         cleanupFunc,
		computeInfoProvider: nodeInfoProvider,
	}, nil
}

func (c *Compute) RegisterLocalComputeCallback(callback compute.Callback) {
	c.computeCallback.RegisterLocalComputeCallback(callback)
}

func createExecutionStore(host host.Host) (store.ExecutionStore, error) {
	// include the host id in the state root dir to avoid conflicts when running multiple nodes on the same machine,
	// e.g. when running tests or when running devstack
	configDir, err := system.EnsureConfigDir()
	if err != nil {
		return nil, err
	}
	stateRootDir := filepath.Join(configDir, "execution-state-"+host.ID().String())
	err = os.MkdirAll(stateRootDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return inlocalstore.NewPersistentExecutionStore(inlocalstore.PersistentJobStoreParams{
		Store:   inmemory.NewStore(),
		RootDir: stateRootDir,
	})
}

func (c *Compute) cleanup(ctx context.Context) {
	c.cleanupFunc(ctx)
}
