package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/evaluation"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/planner"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/scheduler"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/ranking"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/selector"
)

type SchedulerService struct {
	Broker       *evaluation.InMemoryBroker
	Watcher      *evaluation.Watcher
	Workers      []*orchestrator.Worker
	Housekeeping *orchestrator.Housekeeping
}

// TODO we need to enable overriding of the rety strategy in testing
/*

	retryStrategy := requesterConfig.RetryStrategy
	if retryStrategy == nil {
		// retry strategy
		retryStrategyChain := retry.NewChain()
		retryStrategyChain.Add(
			retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: true}),
		)
		retryStrategy = retryStrategyChain
	}
*/

func SetupSchedulerService(
	name string,
	cfg v2.Orchestrator,
	computeEndpoint compute.Endpoint,
	jobStore *boltjobstore.BoltJobStore,
	eventEmitter orchestrator.EventEmitter,
	discoverer orchestrator.NodeDiscoverer,
	retryStrategy orchestrator.RetryStrategy,
) (*SchedulerService, error) {

	// evaluation broker
	broker, err := evaluation.NewInMemoryBroker(evaluation.InMemoryBrokerParams{
		VisibilityTimeout: time.Duration(cfg.Broker.VisibilityTimeout),
		MaxReceiveCount:   cfg.Broker.MaxRetries,
		// NB(forrest): pulled from the default config
		InitialRetryDelay: time.Second,
		// NB(forrest): pulled from the default config
		SubsequentRetryDelay: time.Second * 30,
	})
	if err != nil {
		return nil, fmt.Errorf("creating evaluation broker: %w", err)
	}

	// evaluations watcher
	watcher := evaluation.NewWatcher(jobStore, broker)

	// planners that execute the proposed plan by the scheduler
	// order of the planners is important as they are executed in order
	planners := planner.NewChain(
		// planner that persist the desired state as defined by the scheduler
		planner.NewStateUpdater(jobStore),

		// planner that forwards the desired state to the compute nodes,
		// and updates the observed state if the compute node accepts the desired state
		planner.NewComputeForwarder(planner.ComputeForwarderParams{
			ID:             name,
			ComputeService: computeEndpoint,
			JobStore:       jobStore,
		}),

		// planner that publishes events on job completion or failure
		planner.NewEventEmitter(planner.EventEmitterParams{
			ID:           name,
			EventEmitter: eventEmitter,
		}),

		// logs job completion or failure
		planner.NewLoggingPlanner(),
	)

	// node selector
	nodeRanker, err := createNodeRanker(jobStore)
	if err != nil {
		return nil, err
	}

	nodeSelector := selector.NewNodeSelector(
		discoverer,
		nodeRanker,
		// selector constraints: require nodes be online and approved to schedule
		orchestrator.NodeSelectionConstraints{
			RequireConnected: true,
			RequireApproval:  true,
		},
	)

	// scheduler provider
	batchServiceJobScheduler := scheduler.NewBatchServiceJobScheduler(scheduler.BatchServiceJobSchedulerParams{
		JobStore:      jobStore,
		Planner:       planners,
		NodeSelector:  nodeSelector,
		RetryStrategy: retryStrategy,
		// TODO this value depends on the config, testing uses seconds, production uses this.
		QueueBackoff: time.Minute,
	})
	schedulerProvider := orchestrator.NewMappedSchedulerProvider(map[string]orchestrator.Scheduler{
		models.JobTypeBatch:   batchServiceJobScheduler,
		models.JobTypeService: batchServiceJobScheduler,
		models.JobTypeOps: scheduler.NewOpsJobScheduler(scheduler.OpsJobSchedulerParams{
			JobStore:     jobStore,
			Planner:      planners,
			NodeSelector: nodeSelector,
		}),
		models.JobTypeDaemon: scheduler.NewDaemonJobScheduler(scheduler.DaemonJobSchedulerParams{
			JobStore:     jobStore,
			Planner:      planners,
			NodeSelector: nodeSelector,
		}),
	})

	workers := make([]*orchestrator.Worker, 0, cfg.Scheduler.Workers)
	for i := 1; i <= cfg.Scheduler.Workers; i++ {
		// worker config the polls from the broker
		worker := orchestrator.NewWorker(orchestrator.WorkerParams{
			SchedulerProvider: schedulerProvider,
			EvaluationBroker:  broker,
			// TODO this value depends on the config, testing uses second, production uses this
			DequeueTimeout: time.Second * 5,
			// NB(forrest): values taken from default config
			DequeueFailureBackoff: backoff.NewExponential(time.Second, time.Second*30),
		})
		workers = append(workers, worker)
	}

	housekeeping, err := orchestrator.NewHousekeeping(orchestrator.HousekeepingParams{
		JobStore:      jobStore,
		Interval:      time.Duration(cfg.Scheduler.HousekeepingInterval),
		TimeoutBuffer: time.Duration(cfg.Scheduler.HousekeepingTimeout),
	})
	if err != nil {
		return nil, fmt.Errorf("creating house keeping service: %w", err)
	}

	return &SchedulerService{
		Broker:       broker,
		Watcher:      watcher,
		Workers:      workers,
		Housekeeping: housekeeping,
	}, nil
}

func (s *SchedulerService) Start(ctx context.Context) error {
	s.Broker.SetEnabled(true)
	if err := s.Watcher.Backfill(ctx); err != nil {
		return fmt.Errorf("failed to backfill evaluations: %w", err)
	}
	s.Watcher.Start(ctx)
	for i, worker := range s.Workers {
		log.Info().Msgf("starting worker %d", i)
		worker.Start(ctx)
	}
	s.Housekeeping.Start(ctx)
	return nil
}

func (s *SchedulerService) Stop(ctx context.Context) error {
	s.Housekeeping.Stop(ctx)
	for _, worker := range s.Workers {
		worker.Stop()
	}
	s.Broker.SetEnabled(false)
	s.Watcher.Stop()
	return nil
}

func createNodeRanker(jobStore jobstore.Store) (orchestrator.NodeRanker, error) {
	// NB(forrest) 1.5 is from a default
	overSubscriptionNodeRanker, err := ranking.NewOverSubscriptionNodeRanker(1.5)
	if err != nil {
		return nil, err
	}
	// compute node ranker
	nodeRankerChain := ranking.NewChain()
	nodeRankerChain.Add(
		// rankers that act as filters and give a -1 score to nodes that do not match the filter
		ranking.NewEnginesNodeRanker(),
		ranking.NewPublishersNodeRanker(),
		ranking.NewStoragesNodeRanker(),
		ranking.NewLabelsNodeRanker(),
		ranking.NewMaxUsageNodeRanker(),
		overSubscriptionNodeRanker,
		ranking.NewMinVersionNodeRanker(ranking.MinVersionNodeRankerParams{
			// NB(forrest) taken from default
			// TODO does this value make sense, its a very old version.
			MinVersion: models.BuildVersionInfo{
				Major:      "1",
				Minor:      "0",
				GitVersion: "v1.0.4",
			},
		}),
		ranking.NewPreviousExecutionsNodeRanker(ranking.PreviousExecutionsNodeRankerParams{JobStore: jobStore}),
		ranking.NewAvailableCapacityNodeRanker(),
		// arbitrary rankers
		ranking.NewRandomNodeRanker(ranking.RandomNodeRankerParams{
			// NB(forrest): this is taken from the default
			RandomnessRange: 5,
		}),
	)
	return nodeRankerChain, nil
}
