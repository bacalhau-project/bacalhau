package executor

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/types"
)

// Executor is an interface representing something that can execute jobs
// on some kind of backend, such as a docker daemon.
type Executor interface {
	// tells you if the required software is installed on this machine
	// this is used in job selection
	IsInstalled(ctx context.Context) (bool, error)

	// used to filter and select jobs
	HasStorage(ctx context.Context, volume types.StorageSpec) (bool, error)

	// run the given job - it's expected that we have already prepared the job
	// this will return a local filesystem path to the jobs results
	RunJob(ctx context.Context, job *types.Job) (string, error)
}
