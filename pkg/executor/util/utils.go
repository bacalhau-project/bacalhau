package util

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor/language"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	pythonwasm "github.com/filecoin-project/bacalhau/pkg/executor/python_wasm"
	"github.com/filecoin-project/bacalhau/pkg/executor/wasm"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/combo"
	filecoinunsealed "github.com/filecoin-project/bacalhau/pkg/storage/filecoin_unsealed"
	"github.com/filecoin-project/bacalhau/pkg/storage/inline"
	ipfs_storage "github.com/filecoin-project/bacalhau/pkg/storage/ipfs"
	noop_storage "github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/storage/tracing"
	"github.com/filecoin-project/bacalhau/pkg/storage/url/urldownload"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type StandardStorageProviderOptions struct {
	API                  ipfs.Client
	FilecoinUnsealedPath string
	DownloadPath         string
}

type StandardExecutorOptions struct {
	DockerID string
	Storage  StandardStorageProviderOptions
}

func NewStandardStorageProvider(
	_ context.Context,
	cm *system.CleanupManager,
	options StandardStorageProviderOptions,
) (storage.StorageProvider, error) {
	ipfsAPICopyStorage, err := ipfs_storage.NewStorage(cm, options.API)
	if err != nil {
		return nil, err
	}

	urlDownloadStorage, err := urldownload.NewStorage(cm)
	if err != nil {
		return nil, err
	}

	filecoinUnsealedStorage, err := filecoinunsealed.NewStorage(cm, options.FilecoinUnsealedPath)
	if err != nil {
		return nil, err
	}

	inlineStorage := inline.NewStorage()

	var useIPFSDriver storage.Storage = ipfsAPICopyStorage

	// if we are using a FilecoinUnsealedPath then construct a combo
	// driver that will give preference to the filecoin unsealed driver
	// if the cid is deemed to be local
	if options.FilecoinUnsealedPath != "" {
		comboDriver, err := combo.NewStorage(
			cm,
			func(ctx context.Context) ([]storage.Storage, error) {
				return []storage.Storage{
					filecoinUnsealedStorage,
					ipfsAPICopyStorage,
				}, nil
			},
			func(ctx context.Context, spec model.StorageSpec) (storage.Storage, error) {
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
			func(ctx context.Context) (storage.Storage, error) {
				return ipfsAPICopyStorage, nil
			},
		)

		if err != nil {
			return nil, err
		}

		useIPFSDriver = comboDriver
	}

	return model.NewMappedProvider(map[model.StorageSourceType]storage.Storage{
		model.StorageSourceIPFS:             tracing.Wrap(useIPFSDriver),
		model.StorageSourceURLDownload:      tracing.Wrap(urlDownloadStorage),
		model.StorageSourceFilecoinUnsealed: tracing.Wrap(filecoinUnsealedStorage),
		model.StorageSourceInline:           tracing.Wrap(inlineStorage),
	}), nil
}

func NewNoopStorageProvider(
	ctx context.Context,
	cm *system.CleanupManager,
	config noop_storage.StorageConfig,
) (storage.StorageProvider, error) {
	noopStorage, err := noop_storage.NewNoopStorage(ctx, cm, config)
	if err != nil {
		return nil, err
	}
	return model.NewNoopProvider[model.StorageSourceType, storage.Storage](noopStorage), nil
}

func NewStandardExecutorProvider(
	ctx context.Context,
	cm *system.CleanupManager,
	executorOptions StandardExecutorOptions,
) (executor.ExecutorProvider, error) {
	storageProvider, err := NewStandardStorageProvider(ctx, cm, executorOptions.Storage)
	if err != nil {
		return nil, err
	}

	dockerExecutor, err := docker.NewExecutor(ctx, cm, executorOptions.DockerID, storageProvider)
	if err != nil {
		return nil, err
	}

	wasmExecutor, err := wasm.NewExecutor(ctx, storageProvider)
	if err != nil {
		return nil, err
	}

	executors := model.NewMappedProvider(map[model.Engine]executor.Executor{
		model.EngineDocker: dockerExecutor,
		model.EngineWasm:   wasmExecutor,
	})

	// language executors wrap other executors, so pass them a reference to all
	// the executors so they can look up the ones they need
	exLang, err := language.NewExecutor(ctx, cm, executors)
	if err != nil {
		return nil, err
	}
	executors.Add(model.EngineLanguage, exLang)

	exPythonWasm, err := pythonwasm.NewExecutor(executors)
	if err != nil {
		return nil, err
	}
	executors.Add(model.EnginePythonWasm, exPythonWasm)

	return executors, nil
}

// return noop executors for all engines
func NewNoopExecutors(config noop_executor.ExecutorConfig) executor.ExecutorProvider {
	noopExecutor := noop_executor.NewNoopExecutorWithConfig(config)
	return model.NewNoopProvider[model.Engine, executor.Executor](noopExecutor)
}
