package util

import (
	"context"
	"fmt"
	"os"

	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	dockerengine "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
	wasmengine "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/wasm"
	spec_git "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/git"
	spec_gitlfs "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/gitlfs"
	spec_inline "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/inline"
	spec_ipfs "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	spec_local "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/local"
	spec_s3 "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/s3"
	spec_url "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/url"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
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

	return model.NewMappedProvider(map[cid.Cid]storage.Storage{
		spec_ipfs.StorageType:   tracing.Wrap(useIPFSDriver),
		spec_url.StorageType:    tracing.Wrap(urlDownloadStorage),
		spec_inline.StorageType: tracing.Wrap(inlineStorage),
		spec_git.StorageType:    tracing.Wrap(repoCloneStorage),
		spec_gitlfs.StorageType: tracing.Wrap(repoCloneStorage),
		spec_s3.StorageType:     tracing.Wrap(s3Storage),
		spec_local.StorageType:  tracing.Wrap(localDirectoryStorage),
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
	return model.NewNoopProvider[cid.Cid, storage.Storage](noopStorage), nil
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
		dockerengine.EngineType: dockerExecutor,
		wasmengine.EngineType:   wasmExecutor,
	})

	return executors, nil
}

// return noop executors for all engines
func NewNoopExecutors(config noop_executor.ExecutorConfig) executor.ExecutorProvider {
	noopExecutor := noop_executor.NewNoopExecutorWithConfig(config)
	return model.NewNoopProvider[cid.Cid, executor.Executor](noopExecutor)
}
