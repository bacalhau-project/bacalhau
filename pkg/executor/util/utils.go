package util

import (
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor/language"
	"github.com/filecoin-project/bacalhau/pkg/executor/python_wasm"
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

	executors := map[executor.EngineType]executor.Executor{
		executor.EngineDocker: exDocker,
	}

	// language executors wrap other executors, so pass them a reference to all
	// the executors so they can look up the ones they need
	exLang, err := language.NewExecutor(cm, executors)
	executors[executor.EngineLanguage] = exLang
	if err != nil {
		return nil, err
	}
	exPythonWasm, err := python_wasm.NewExecutor(cm, executors)
	executors[executor.EnginePythonWasm] = exPythonWasm
	if err != nil {
		return nil, err
	}
	return executors, nil
}
