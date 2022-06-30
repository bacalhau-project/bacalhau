package util

import (
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/apicopy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fusedocker"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func NewDockerIPFSExecutors(
	cm *system.CleanupManager,
	ipfsMultiAddress,
	dockerID string,
) (map[executor.EngineType]executor.Executor, error) {
	ipfsFuseStorage, err := fusedocker.NewStorageProvider(cm, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	ipfsAPICopyStorage, err := apicopy.NewStorageProvider(cm, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	ex, err := docker.NewExecutor(cm, dockerID,
		map[string]storage.StorageProvider{
			storage.IPFSFuseDocker: ipfsFuseStorage,
			storage.IPFSAPICopy:    ipfsAPICopyStorage,
			// we make the copy driver the "default" storage driver for docker
			// users have to specify the fuse driver explicitly
			storage.IPFSDefault: ipfsAPICopyStorage,
		})
	if err != nil {
		return nil, err
	}

	return map[executor.EngineType]executor.Executor{
		executor.EngineDocker: ex,
	}, nil
}

// return noop executors for all engines
func NewNoopExecutors(
	cm *system.CleanupManager,
) (map[executor.EngineType]executor.Executor, error) {
	noopExecutor, err := noop_executor.NewExecutor()

	if err != nil {
		return nil, err
	}

	return map[executor.EngineType]executor.Executor{
		executor.EngineDocker: noopExecutor,
		executor.EngineNoop:   noopExecutor,
	}, nil
}
