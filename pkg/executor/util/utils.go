package util

import (
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor/language"
	"github.com/filecoin-project/bacalhau/pkg/executor/local"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	pythonwasm "github.com/filecoin-project/bacalhau/pkg/executor/python_wasm"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	apicopy "github.com/filecoin-project/bacalhau/pkg/storage/ipfs_apicopy"
	noop_storage "github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/storage/url/urldownload"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func NewStandardStorageProviders(
	cm *system.CleanupManager,
	ipfsMultiAddress string,
) (map[storage.StorageSourceType]storage.StorageProvider, error) {
	ipfsAPICopyStorage, err := apicopy.NewStorageProvider(cm, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	urlDownloadStorage, err := urldownload.NewStorageProvider(cm)
	if err != nil {
		return nil, err
	}

	return map[storage.StorageSourceType]storage.StorageProvider{
		// fuse driver is disabled so that - in case it poses a security
		// risk - arbitrary users can't request it
		// storage.IPFS_FUSE_DOCKER: ipfsFuseStorage,
		storage.StorageSourceIPFS: ipfsAPICopyStorage,
		// we make the copy driver the "default" storage driver for docker
		// users have to specify the fuse driver explicitly
		storage.StorageSourceURLDownload: urlDownloadStorage,
	}, nil
}

func NewNoopStorageProviders(
	cm *system.CleanupManager,
) (map[storage.StorageSourceType]storage.StorageProvider, error) {
	noopStorage, err := noop_storage.NewStorageProvider(cm)
	if err != nil {
		return nil, err
	}
	return map[storage.StorageSourceType]storage.StorageProvider{
		storage.StorageSourceIPFS:        noopStorage,
		storage.StorageSourceURLDownload: noopStorage,
	}, nil
}

func NewStandardExecutors(
	cm *system.CleanupManager,
	ipfsMultiAddress,
	dockerID string,
) (map[executor.EngineType]executor.Executor, error) {
	storageProviders, err := NewStandardStorageProviders(cm, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	dockerExecutor, err := docker.NewExecutor(cm, dockerID, storageProviders)

	if err != nil {
		return nil, err
	}

	executors := map[executor.EngineType]executor.Executor{
		executor.EngineDocker: dockerExecutor,
	}

	// language executors wrap other executors, so pass them a reference to all
	// the executors so they can look up the ones they need
	exLang, err := language.NewExecutor(cm, executors)
	executors[executor.EngineLanguage] = exLang
	if err != nil {
		return nil, err
	}
	exPythonWasm, err := pythonwasm.NewExecutor(cm, executors)
	executors[executor.EnginePythonWasm] = exPythonWasm
	if err != nil {
		return nil, err
	}
	return executors, nil
}

func NewLocalStandardExecutors(
	cm *system.CleanupManager,
	ipfsMultiAddress,
	dockerID string,
) (*local.Local, error) {
	// Don't allow user to choose the fuse driver in case it has security issues.
	// ipfsFuseStorage, err := fusedocker.NewStorageProvider(cm, ipfsMultiAddress)
	// if err != nil {
	// 	return nil, err
	// }

	ipfsAPICopyStorage, err := apicopy.NewStorageProvider(cm, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	urlDownloadStorage, err := urldownload.NewStorageProvider(cm)
	if err != nil {
		return nil, err
	}

	exDocker, err := local.NewExecutor(cm, dockerID,
		map[storage.StorageSourceType]storage.StorageProvider{
			// fuse driver is disabled so that - in case it poses a security
			// risk - arbitrary users can't request it
			// storage.IPFS_FUSE_DOCKER: ipfsFuseStorage,
			storage.StorageSourceIPFS: ipfsAPICopyStorage,
			// we make the copy driver the "default" storage driver for docker
			// users have to specify the fuse driver explicitly
			storage.StorageSourceURLDownload: urlDownloadStorage,
		})
	if err != nil {
		return nil, err
	}

	return exDocker, nil
}

// return noop executors for all engines
func NewNoopExecutors(
	cm *system.CleanupManager,
	config noop_executor.ExecutorConfig,
) (map[executor.EngineType]executor.Executor, error) {
	noopExecutor, err := noop_executor.NewExecutorWithConfig(config)

	if err != nil {
		return nil, err
	}

	return map[executor.EngineType]executor.Executor{
		executor.EngineDocker: noopExecutor,
		executor.EngineNoop:   noopExecutor,
	}, nil
}
