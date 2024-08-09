package compute

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/sensors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type ExecutorProvider interface {
	Store() store.ExecutionStore
	Executor() *compute.ExecutorBuffer
	DebugProvider() models.DebugInfoProvider
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

func NewExecutorProvider(
	ctx context.Context,
	name string,
	transport *nats_transport.NATSTransport,
	engines executor.ExecutorProvider,
	storages storage.StorageProvider,
	publishers publisher.PublisherProvider,
	capacity CapacityProvider,
) (*ComputeExecutorProvider, error) {

	// TODO provide a path for the compute execution store.
	executionStore, err := boltdb.NewStore(ctx, "TODO")
	if err != nil {
		return nil, fmt.Errorf("createing execution store: %w", err)
	}

	// TODO the results path needs to be a child of the repo path
	resultsPath, err := compute.NewResultsPath()
	if err != nil {
		return nil, fmt.Errorf("creating results path: %w", err)
	}

	baseExecutor := compute.NewBaseExecutor(compute.BaseExecutorParams{
		Callback:    transport.CallbackProxy(),
		ID:          name,
		Store:       executionStore,
		Storages:    storages,
		Executors:   engines,
		Publishers:  publishers,
		ResultsPath: *resultsPath,

		// TODO the storage directory needs to be a child of the repo path
		StorageDirectory: "TODO",

		// TODO(forrest) [correctness] this is leaking a testing attribute into core code. Don't do this
		//FailureInjectionConfig: config.FailureInjectionConfig,
	})

	bufferRunner := compute.NewExecutorBuffer(compute.ExecutorBufferParams{
		ID:                     name,
		Callback:               transport.CallbackProxy(),
		DelegateExecutor:       baseExecutor,
		RunningCapacityTracker: capacity.RunningTracker(),
		EnqueuedUsageTracker:   capacity.QueuedTracker(),
		// TODO(forrest) [correctness] this field doesn't make sense and should be removed:
		// It doesn't make sense for a compute node operator to set a default timeout on the executor
		// and disregard the value during bidding. There shouldn't be a default timeout, instead compute
		// nodes should timeout based on the values specified in the job. If the job doesn't specify a timeout
		// then the compute node should happily run it till the end of time, or until its canceled by an operator/client
		DefaultJobExecutionTimeout: models.NoTimeout,
	})

	runningInfoProvider := sensors.NewRunningExecutionsInfoProvider(sensors.RunningExecutionsInfoProviderParams{
		Name:          "ActiveJobs",
		BackendBuffer: bufferRunner,
	})
	return &ComputeExecutorProvider{
		executionStore: executionStore,
		executor:       bufferRunner,
		debugInfo:      runningInfoProvider,
		resultsPath:    resultsPath,
		loggingSensor: sensors.NewLoggingSensor(sensors.LoggingSensorParams{
			InfoProvider: runningInfoProvider,
			// NB(forrest): value pulled from NewDefaultComputeParams
			Interval: time.Second * 10,
		}),
	}, nil
}

type ComputeExecutorProvider struct {
	executionStore store.ExecutionStore
	executor       *compute.ExecutorBuffer
	debugInfo      models.DebugInfoProvider
	resultsPath    *compute.ResultsPath
	loggingSensor  *sensors.LoggingSensor
}

func (c *ComputeExecutorProvider) Store() store.ExecutionStore {
	return c.executionStore
}

func (c *ComputeExecutorProvider) Executor() *compute.ExecutorBuffer {
	return c.executor
}

func (c *ComputeExecutorProvider) DebugProvider() models.DebugInfoProvider {
	return c.debugInfo
}

func (c *ComputeExecutorProvider) Start(ctx context.Context) error {
	if err := compute.NewStartup(c.executionStore, c.executor).Execute(ctx); err != nil {
		return fmt.Errorf("failed to execute compute node startup tasks: %w", err)
	}
	c.loggingSensor.Start(ctx)
	return nil
}

func (c *ComputeExecutorProvider) Stop(ctx context.Context) error {
	var err error
	if err := c.executionStore.Close(ctx); err != nil {
		err = errors.Join(err, fmt.Errorf("closing execution store: %w", err))
	}
	if err := c.resultsPath.Close(); err != nil {
		err = errors.Join(err, fmt.Errorf("closing results path: %w", err))
	}
	return err
}
