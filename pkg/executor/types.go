package executor

import (
	"context"
	"io"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// Returns a executor for the given engine type
type ExecutorProvider interface {
	model.Provider[model.Engine, Executor]
}

// Executor represents an execution provider, which can execute jobs on some
// kind of backend, such as a docker daemon.
type Executor interface {
	model.Providable

	// used to filter and select jobs
	//    tells us if the storage resource is "close" i.e. cheap to access
	HasStorageLocally(context.Context, model.StorageSpec) (bool, error)

	// A BidStrategy that should return a positive response if the executor
	// could run the job or a negative response otherwise.
	GetSemanticBidStrategy(context.Context) (bidstrategy.SemanticBidStrategy, error)

	GetResourceBidStrategy(ctx context.Context) (bidstrategy.ResourceBidStrategy, error)

	//    tells us how much storage the given volume would consume
	//    which we then use to calculate if there is capacity
	//    alongside cpu & memory usage
	GetVolumeSize(context.Context, model.StorageSpec) (uint64, error)

	// GetOutputStream retrieves a muxed stream from the executor
	GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error)

	// run the given job - it's expected that we have already prepared the job
	// this will return a local filesystem path to the jobs results
	Run(
		ctx context.Context,
		executionID string,
		job model.Job,
		resultsDir string,
	) (*model.RunCommandResult, error)
}
