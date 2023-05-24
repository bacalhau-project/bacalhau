package util

import (
	"context"
	"fmt"
	"os"

	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor/language"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	pythonwasm "github.com/bacalhau-project/bacalhau/pkg/executor/python_wasm"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	dockerengine "github.com/bacalhau-project/bacalhau/pkg/model/specs/engine/docker"
	wasmengine "github.com/bacalhau-project/bacalhau/pkg/model/specs/engine/wasm"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/combo"
	filecoinunsealed "github.com/bacalhau-project/bacalhau/pkg/storage/filecoin_unsealed"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	ipfs_storage "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	localdirectory "github.com/bacalhau-project/bacalhau/pkg/storage/local_directory"
	noop_storage "github.com/bacalhau-project/bacalhau/pkg/storage/noop"
	repo "github.com/bacalhau-project/bacalhau/pkg/storage/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage/tracing"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type StandardStorageProviderOptions struct {
	API                   ipfs.Client
	FilecoinUnsealedPath  string
	DownloadPath          string
	EstuaryAPIKey         string
	AllowListedLocalPaths []string
}

type StandardExecutorOptions struct {
	DockerID string
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

	repoCloneStorage, err := repo.NewStorage(cm, ipfsAPICopyStorage, options.EstuaryAPIKey)
	if err != nil {
		return nil, err
	}

	inlineStorage := inline.NewStorage()

	s3Storage, err := configureS3StorageProvider(cm)
	if err != nil {
		return nil, err
	}

	localDirectoryStorage, err := localdirectory.NewStorageProvider(localdirectory.StorageProviderParams{
		AllowedPaths: localdirectory.ParseAllowPaths(options.AllowListedLocalPaths),
	})
	if err != nil {
		return nil, err
	}

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
		model.StorageSourceRepoClone:        tracing.Wrap(repoCloneStorage),
		model.StorageSourceRepoCloneLFS:     tracing.Wrap(repoCloneStorage),
		model.StorageSourceS3:               tracing.Wrap(s3Storage),
		model.StorageSourceLocalDirectory:   tracing.Wrap(localDirectoryStorage),
	}), nil
}

func configureS3StorageProvider(cm *system.CleanupManager) (*s3.StorageProvider, error) {
	dir, err := os.MkdirTemp(config.GetStoragePath(), "bacalhau-s3-input")
	if err != nil {
		return nil, err
	}

	cm.RegisterCallback(func() error {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("unable to clean up S3 storage directory: %w", err)
		}
		return nil
	})

	cfg, err := s3helper.DefaultAWSConfig()
	if err != nil {
		return nil, err
	}
	clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
		AWSConfig: cfg,
	})
	s3Storage := s3.NewStorage(s3.StorageProviderParams{
		LocalDir:       dir,
		ClientProvider: clientProvider,
	})
	return s3Storage, nil
}

func NewNoopStorageProvider(
	ctx context.Context,
	cm *system.CleanupManager,
	config noop_storage.StorageConfig,
) (storage.StorageProvider, error) {
	noopStorage := noop_storage.NewNoopStorageWithConfig(config)
	return model.NewNoopProvider[model.StorageSourceType, storage.Storage](noopStorage), nil
}

func NewStandardExecutorProvider(
	ctx context.Context,
	cm *system.CleanupManager,
	storageProvider storage.StorageProvider,
	executorOptions StandardExecutorOptions,
) (executor.ExecutorProvider, error) {
	dockerExecutor, err := docker.NewExecutor(ctx, cm, executorOptions.DockerID, storageProvider)
	if err != nil {
		return nil, err
	}

	wasmExecutor, err := wasm.NewExecutor(ctx, storageProvider)
	if err != nil {
		return nil, err
	}

	executors := model.NewMappedProvider(map[cid.Cid]executor.Executor{
		dockerengine.EngineSchema.Cid(): dockerExecutor,
		wasmengine.EngineSchema.Cid():   wasmExecutor,
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
	return model.NewNoopProvider[cid.Cid, executor.Executor](noopExecutor)
}
