package util

import (
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/api_copy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fuse_docker"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func NewDockerIPFSExecutors(cm *system.CleanupManager, ipfsMultiAddress string,
	dockerId string) (map[executor.EngineType]executor.Executor, error) {

	ipfsFuseStorage, err := fuse_docker.NewStorageProvider(cm, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	ipfsApiCopyStorage, err := api_copy.NewStorageProvider(cm, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	ex, err := docker.NewExecutor(cm, dockerId,
		map[string]storage.StorageProvider{
			storage.IPFS_FUSE_DOCKER: ipfsFuseStorage,
			storage.IPFS_API_COPY:    ipfsApiCopyStorage,
			// we make the copy driver the "default" storage driver for docker
			// users have to specify the fuse driver explicitly
			storage.IPFS_DEFAULT: ipfsApiCopyStorage,
		})
	if err != nil {
		return nil, err
	}

	return map[executor.EngineType]executor.Executor{
		executor.EngineDocker: ex,
	}, nil
}
