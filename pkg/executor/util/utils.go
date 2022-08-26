package util

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor/language"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	pythonwasm "github.com/filecoin-project/bacalhau/pkg/executor/python_wasm"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/combo"
	"github.com/filecoin-project/bacalhau/pkg/storage/filecoinunsealed"
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
	DockerID   string
	IsBadActor bool
	Storage    StandardStorageProviderOptions
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

	filecoinUnsealedStorage, err := filecoinunsealed.NewStorageProvider(cm, options.FilecoinUnsealedPath)
	if err != nil {
		return nil, err
	}

	var useIPFSDriver storage.StorageProvider = ipfsAPICopyStorage

	// if we are using a FilecoinUnsealedPath then construct a combo
	// driver that will give preference to the filecoin unsealed driver
	// if the cid is deemed to be local
	if options.FilecoinUnsealedPath != "" {
		comboDriver, err := combo.NewStorageProvider(
			cm,
			func(ctx context.Context) ([]storage.StorageProvider, error) {
				return []storage.StorageProvider{
					filecoinUnsealedStorage,
					ipfsAPICopyStorage,
				}, nil
			},
			func(ctx context.Context, spec storage.StorageSpec) (storage.StorageProvider, error) {
				filecoinUnsealedHasCid, err := filecoinUnsealedStorage.HasStorageLocally(ctx, spec)
				if err != nil {
					return ipfsAPICopyStorage, err
				}
				if filecoinUnsealedHasCid {
					return filecoinUnsealedStorage, nil
				} else {
					return ipfsAPICopyStorage, nil
				}
			},
			func(ctx context.Context) (storage.StorageProvider, error) {
				return ipfsAPICopyStorage, nil
			},
		)

		if err != nil {
			return nil, err
		}

		useIPFSDriver = comboDriver
	}

	return map[storage.StorageSourceType]storage.StorageProvider{
		storage.StorageSourceIPFS:             useIPFSDriver,
		storage.StorageSourceURLDownload:      urlDownloadStorage,
		storage.StorageSourceFilecoinUnsealed: filecoinUnsealedStorage,
	}, nil
}

func NewNoopStorageProviders(
	cm *system.CleanupManager,
	config noop_storage.StorageConfig,
) (map[storage.StorageSourceType]storage.StorageProvider, error) {
	noopStorage, err := noop_storage.NewStorageProvider(cm, config)
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
