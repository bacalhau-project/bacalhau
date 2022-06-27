package util

import (
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor/language"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/api_copy"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func NewStandardExecutors(cm *system.CleanupManager, ipfsMultiAddress string,
	dockerId string) (map[executor.EngineType]executor.Executor, error) {

	// ipfsFuseStorage, err := fuse_docker.NewStorageProvider(cm, ipfsMultiAddress)
	// if err != nil {
	// 	return nil, err
	// }

	ipfsApiCopyStorage, err := api_copy.NewStorageProvider(cm, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	exDocker, err := docker.NewExecutor(cm, dockerId,
		map[string]storage.StorageProvider{
			// fuse driver is disabled so that - in case it poses a security
			// risk - arbitrary users can't request it
			// storage.IPFS_FUSE_DOCKER: ipfsFuseStorage,
			storage.IPFS_API_COPY: ipfsApiCopyStorage,
			// we make the copy driver the "default" storage driver for docker
			// users have to specify the fuse driver explicitly
			storage.IPFS_DEFAULT: ipfsApiCopyStorage,
		})
	if err != nil {
		return nil, err
	}

	// language executors are just wrappers around docker at the end of the day,
	// so thread the dockerId param all the way down
	exLang, err := language.NewExecutor(cm, dockerId,
		map[string]storage.StorageProvider{
			storage.IPFS_API_COPY: ipfsApiCopyStorage,
			storage.IPFS_DEFAULT:  ipfsApiCopyStorage,
		})
	if err != nil {
		return nil, err
	}
	return map[executor.EngineType]executor.Executor{
		executor.EngineDocker:   exDocker,
		executor.EngineLanguage: exLang,
	}, nil
}
