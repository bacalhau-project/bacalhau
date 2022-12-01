package node

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/backend"
	"github.com/filecoin-project/bacalhau/pkg/compute/bidstrategy"
	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/compute/capacity/disk"
	"github.com/filecoin-project/bacalhau/pkg/compute/frontend"
	"github.com/filecoin-project/bacalhau/pkg/compute/pubsub"
	"github.com/filecoin-project/bacalhau/pkg/compute/sensors"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
)

type Compute struct {
	// Visible for testing
	Frontend           frontend.Service
	nodeID             string
	ExecutionStore     store.ExecutionStore
	frontendProxy      pubsub.FrontendEventProxy
	debugInfoProviders []model.DebugInfoProvider
}

//nolint:funlen
func NewComputeNode(
	ctx context.Context,
	nodeID string,
	config ComputeConfig,
	jobStore localdb.LocalDB,
	executors executor.ExecutorProvider,
	verifiers verifier.VerifierProvider,
	publishers publisher.PublisherProvider,
	jobEventPublisher eventhandler.JobEventHandler) *Compute {
	debugInfoProviders := []model.DebugInfoProvider{}
	executionStore := inmemory.NewStore()

	// backend
	capacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: config.TotalResourceLimits,
	})
	debugInfoProviders = append(debugInfoProviders, sensors.NewCapacityDebugInfoProvider(sensors.CapacityDebugInfoProviderParams{
		CapacityTracker: capacityTracker,
	}))

	backendCallback := backend.NewChainedCallback(backend.ChainedCallbackParams{
		Callbacks: []backend.Callback{
			backend.NewStateUpdateCallback(backend.StateUpdateCallbackParams{
				ExecutionStore: executionStore,
			}),
			pubsub.NewBackendCallback(pubsub.BackendCallbackParams{
				NodeID:            nodeID,
				ExecutionStore:    executionStore,
				JobEventPublisher: jobEventPublisher,
			}),
		},
	})

	baseRunner := backend.NewBaseService(backend.BaseServiceParams{
		ID:         nodeID,
		Callback:   backendCallback,
		Store:      executionStore,
		Executors:  executors,
		Verifiers:  verifiers,
		Publishers: publishers,
	})

	bufferRunner := backend.NewServiceBuffer(backend.ServiceBufferParams{
		DelegateService:            baseRunner,
		Callback:                   backendCallback,
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

	// frontend
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

	frontendNode := frontend.NewBaseService(frontend.BaseServiceParams{
		ID:              nodeID,
		ExecutionStore:  executionStore,
		UsageCalculator: capacityCalculator,
		BidStrategy:     biddingStrategy,
		Backend:         bufferRunner,
	})

	frontendProxy := *pubsub.NewFrontendEventProxy(pubsub.FrontendEventProxyParams{
		NodeID:            nodeID,
		Frontend:          frontendNode,
		JobStore:          jobStore,
		ExecutionStore:    executionStore,
		JobEventPublisher: jobEventPublisher,
	})

	return &Compute{
		nodeID:             nodeID,
		Frontend:           frontendNode,
		ExecutionStore:     executionStore,
		frontendProxy:      frontendProxy,
		debugInfoProviders: debugInfoProviders,
	}
}
