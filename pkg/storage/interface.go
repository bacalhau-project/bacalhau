package storage

import "github.com/filecoin-project/bacalhau/pkg/types"

type Storage interface {

	// tells you if the required software is installed on this machine
	// this is used in job selection
	IsInstalled() (bool, error)

	// do we have access to the given storage resource?
	// this is used to filter the job
	HasResourceLocally(storage types.StorageSpec) (bool, error)
}
