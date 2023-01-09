package node

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/compute/bidstrategy"
	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/compute/capacity/disk"
	"github.com/filecoin-project/bacalhau/pkg/compute/sensors"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/pubsub"
	"github.com/filecoin-project/bacalhau/pkg/simulator"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/bprotocol"
	simulator_protocol "github.com/filecoin-project/bacalhau/pkg/transport/simulator"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/libp2p/go-libp2p/core/host"
)

type Compute struct {
	// Visible for testing
	LocalEndpoint      compute.Endpoint
	Capacity           capacity.Tracker
	host               host.Host
	ExecutionStore     store.ExecutionStore
	debugInfoProviders []model.DebugInfoProvider
	computeCallback    *bprotocol.CallbackProxy
	nodeInfoPublisher  *compute.NodeInfoPublisher
}

//nolint:funlen
func NewComputeNode(
	ctx context.Context,
	cleanupManager *system.CleanupManager,
	host host.Host,
	config ComputeConfig,
	simulatorNodeID string,
	simulatorRequestHandler *simulator.RequestHandler,
	executors executor.ExecutorProvider,
	verifiers verifier.VerifierProvider,
	publishers publisher.PublisherProvider,
	nodeInfoPubSub pubsub.PubSub[model.NodeInfo]) *Compute {
	debugInfoProviders := []model.DebugInfoProvider{}
	executionStore := inmemory.NewStore()

	// executor/backend
	capacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: config.TotalResourceLimits,
	})
	debugInfoProviders = append(debugInfoProviders, sensors.NewCapacityDebugInfoProvider(sensors.CapacityDebugInfoProviderParams{
		CapacityTracker: capacityTracker,
	}))

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
		RunningCapacityTracker:     capacityTracker,
		DefaultJobExecutionTimeout: config.DefaultJobExecutionTimeout,
		BackoffDuration:            50 * time.Millisecond,
	})
	runningInfoProvider := sensors.NewRunningExecutionsInfoProvider(sensors.RunningExecutionsInfoProviderParams{
		BackendBuffer: bufferRunner,
	})
	debugInfoProviders = append(debugInfoProviders, runningInfoProvider)
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

	biddingStrategy := bidstrategy.NewChainedBidStrategy(
		bidstrategy.NewNetworkingStrategy(config.JobSelectionPolicy.AcceptNetworkedJobs),
		bidstrategy.NewMaxCapacityStrategy(bidstrategy.MaxCapacityStrategyParams{
			MaxJobRequirements: config.JobResourceLimits,
		}),
		bidstrategy.NewAvailableCapacityStrategy(bidstrategy.AvailableCapacityStrategyParams{
			CapacityTracker: capacityTracker,
			CommitFactor:    config.OverCommitResourcesFactor,
		}),
		// TODO XXX: don't hardcode networkSize, calculate this dynamically from
		//  libp2p instead somehow. https://github.com/filecoin-project/bacalhau/issues/512
		bidstrategy.NewDistanceDelayStrategy(bidstrategy.DistanceDelayStrategyParams{
			NetworkSize: 1,
		}),
		bidstrategy.NewEnginesInstalledStrategy(bidstrategy.EnginesInstalledStrategyParams{
			Executors: executors,
			Verifiers: verifiers,
		}),
		bidstrategy.NewExternalCommandStrategy(bidstrategy.ExternalCommandStrategyParams{
			Command: config.JobSelectionPolicy.ProbeExec,
		}),
		bidstrategy.NewExternalHTTPStrategy(bidstrategy.ExternalHTTPStrategyParams{
			URL: config.JobSelectionPolicy.ProbeHTTP,
		}),
		bidstrategy.NewInputLocalityStrategy(bidstrategy.InputLocalityStrategyParams{
			Locality:  config.JobSelectionPolicy.Locality,
			Executors: executors,
		}),
		bidstrategy.NewStatelessJobStrategy(bidstrategy.StatelessJobStrategyParams{
			RejectStatelessJobs: config.JobSelectionPolicy.RejectStatelessJobs,
		}),
		bidstrategy.NewTimeoutStrategy(bidstrategy.TimeoutStrategyParams{
			MaxJobExecutionTimeout: config.MaxJobExecutionTimeout,
			MinJobExecutionTimeout: config.MinJobExecutionTimeout,
		}),
	)

	// node info publisher
	nodeInfoPublisher := compute.NewNodeInfoPublisher(compute.NodeInfoPublisherParams{
		PubSub:             nodeInfoPubSub,
		Host:               host,
		Executors:          executors,
		CapacityTracker:    capacityTracker,
		MaxJobRequirements: config.JobResourceLimits,
		Interval:           config.NodeInfoPublisherInterval,
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

	return &Compute{
		host:               host,
		LocalEndpoint:      baseEndpoint,
		Capacity:           capacityTracker,
		ExecutionStore:     executionStore,
		debugInfoProviders: debugInfoProviders,
		computeCallback:    standardComputeCallback,
		nodeInfoPublisher:  nodeInfoPublisher,
	}
}

func (c *Compute) RegisterLocalComputeCallback(callback compute.Callback) {
	c.computeCallback.RegisterLocalComputeCallback(callback)
}
