package executor

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
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
	//    tells us how much storage the given volume would consume
	//    which we then use to calculate if there is capacity
	//    alongside cpu & memory usage
	GetVolumeSize(context.Context, model.StorageSpec) (uint64, error)

	// run the given job - it's expected that we have already prepared the job
	// this will return a local filesystem path to the jobs results
	RunShard(
		ctx context.Context,
		shard model.JobShard,
		resultsDir string,
	) (*model.RunCommandResult, error)
}
