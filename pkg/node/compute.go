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
	"github.com/filecoin-project/bacalhau/pkg/transport/bprotocol"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/libp2p/go-libp2p/core/host"
)

type Compute struct {
	// Visible for testing
	Endpoint           compute.Endpoint
	LocalEndpoint      compute.Endpoint
	nodeID             string
	ExecutionStore     store.ExecutionStore
	debugInfoProviders []model.DebugInfoProvider
	computeCallback    *bprotocol.CallbackProxy
}

//nolint:funlen
func NewComputeNode(
	ctx context.Context,
	host host.Host,
	config ComputeConfig,
	executors executor.ExecutorProvider,
	verifiers verifier.VerifierProvider,
	publishers publisher.PublisherProvider) *Compute {
	debugInfoProviders := []model.DebugInfoProvider{}
	executionStore := inmemory.NewStore()

	// executor/backend
	capacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: config.TotalResourceLimits,
	})
	debugInfoProviders = append(debugInfoProviders, sensors.NewCapacityDebugInfoProvider(sensors.CapacityDebugInfoProviderParams{
		CapacityTracker: capacityTracker,
	}))

	computeCallback := bprotocol.NewCallbackProxy(bprotocol.CallbackProxyParams{
		Host: host,
	})

	baseExecutor := compute.NewBaseExecutor(compute.BaseExecutorParams{
		ID:         host.ID().String(),
		Callback:   computeCallback,
		Store:      executionStore,
		Executors:  executors,
		Verifiers:  verifiers,
		Publishers: publishers,
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
		go loggingSensor.Start(ctx)
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

	baseEndpoint := compute.NewBaseEndpoint(compute.BaseEndpointParams{
		ID:              host.ID().String(),
		ExecutionStore:  executionStore,
		UsageCalculator: capacityCalculator,
		BidStrategy:     biddingStrategy,
		Executor:        bufferRunner,
	})

	// register a handler for the bacalhau protocol handler that will forward requests to baseEndpoint
	bprotocol.NewComputeHandler(bprotocol.ComputeHandlerParams{
		Host:            host,
		ComputeEndpoint: baseEndpoint,
	})

	endpointProxy := bprotocol.NewComputeProxy(bprotocol.ComputeProxyParams{
		Host:          host,
		LocalEndpoint: baseEndpoint,
	})

	return &Compute{
		nodeID:             host.ID().String(),
		Endpoint:           endpointProxy,
		LocalEndpoint:      baseEndpoint,
		ExecutionStore:     executionStore,
		debugInfoProviders: debugInfoProviders,
		computeCallback:    computeCallback,
	}
}

func (c *Compute) RegisterLocalComputeCallback(callback compute.Callback) {
	c.computeCallback.RegisterLocalCallback(callback)
}
