package executor

import (
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type Executor interface {
	// tells you if the required software is installed on this machine
	// this is used in job selection
	IsInstalled() (bool, error)

	// used to filter and select jobs
	HasStorage(volume types.StorageSpec) (bool, error)

	// run the given job - it's expected that we have already prepared the job
	// this will return a local filesystem path to the jobs results
	RunJob(job *types.Job) (string, error)
}
