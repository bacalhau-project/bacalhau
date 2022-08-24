package util

import (
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor/language"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	pythonwasm "github.com/filecoin-project/bacalhau/pkg/executor/python_wasm"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/filecoin_unsealed"
	apicopy "github.com/filecoin-project/bacalhau/pkg/storage/ipfs_apicopy"
	noop_storage "github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/storage/url/urldownload"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type StandardStorageProviderOptions struct {
	IPFSMultiaddress     string
	FilecoinUnsealedPath string
}

type StandardExecutorOptions struct {
	DockerID string
	Storage  StandardStorageProviderOptions
}

func NewStandardStorageProviders(
	cm *system.CleanupManager,
	options StandardStorageProviderOptions,
) (map[storage.StorageSourceType]storage.StorageProvider, error) {
	ipfsAPICopyStorage, err := apicopy.NewStorageProvider(cm, options.IPFSMultiaddress)
	if err != nil {
		return nil, err
	}

	urlDownloadStorage, err := urldownload.NewStorageProvider(cm)
	if err != nil {
		return nil, err
	}

	filecoinUnsealedStorage, err := filecoin_unsealed.NewStorageProvider(cm, options.FilecoinUnsealedPath)
	if err != nil {
		return nil, err
	}

	return map[storage.StorageSourceType]storage.StorageProvider{
		storage.StorageSourceIPFS:             ipfsAPICopyStorage,
		storage.StorageSourceURLDownload:      urlDownloadStorage,
		storage.StorageSourceFilecoinUnsealed: filecoinUnsealedStorage,
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
	executorOptions StandardExecutorOptions,
) (map[executor.EngineType]executor.Executor, error) {
	storageProviders, err := NewStandardStorageProviders(cm, executorOptions.Storage)
	if err != nil {
		return nil, err
	}

	dockerExecutor, err := docker.NewExecutor(cm, executorOptions.DockerID, storageProviders)

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
