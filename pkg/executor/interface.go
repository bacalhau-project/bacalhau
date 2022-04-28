package executor

import (
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type Executor interface {

	// tells you if the required software is installed on this machine
	// this is used in job selection
	IsInstalled() (bool, error)

	// given a list of storage resources, prepare the storage for the job
	// this is called before a job is run and the storage types
	// are extracted from the job spec
	// it is entirely up to the executor implementation what it actually means
	// to "prepare" some storage - in the case of Docker - it means start
	// the ipfs daemon in sidecar mode
	PrepareStorage(storageProvider storage.Storage, volume types.StorageSpec) error

	// run the given job - it's expected that we have already prepared the job
	RunJob(*types.Job) ([]types.StorageSpec, error)
}
