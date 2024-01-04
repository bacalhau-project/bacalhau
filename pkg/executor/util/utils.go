package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
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
	DownloadPath          string
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
	ipfsAPICopyStorage, err := ipfs_storage.NewStorage(options.API)
	if err != nil {
		return nil, err
	}

	urlDownloadStorage := urldownload.NewStorage()
	if err != nil {
		return nil, err
	}

	repoCloneStorage, err := repo.NewStorage(ipfsAPICopyStorage)
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

	return provider.NewMappedProvider(map[string]storage.Storage{
		models.StorageSourceIPFS:           tracing.Wrap(useIPFSDriver),
		models.StorageSourceURL:            tracing.Wrap(urlDownloadStorage),
		models.StorageSourceInline:         tracing.Wrap(inlineStorage),
		models.StorageSourceRepoClone:      tracing.Wrap(repoCloneStorage),
		models.StorageSourceRepoCloneLFS:   tracing.Wrap(repoCloneStorage),
		models.StorageSourceS3:             tracing.Wrap(s3Storage),
		models.StorageSourceLocalDirectory: tracing.Wrap(localDirectoryStorage),
	}), nil
}

func configureS3StorageProvider(cm *system.CleanupManager) (*s3.StorageProvider, error) {
	cfg, err := s3helper.DefaultAWSConfig()
	if err != nil {
		return nil, err
	}
	clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
		AWSConfig: cfg,
	})
	s3Storage := s3.NewStorage(s3.StorageProviderParams{
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
	return provider.NewNoopProvider[storage.Storage](noopStorage), nil
}

func NewStandardExecutorProvider(
	ctx context.Context,
	cm *system.CleanupManager,
	executorOptions StandardExecutorOptions,
) (executor.ExecutorProvider, error) {
	dockerExecutor, err := docker.NewExecutor(ctx, executorOptions.DockerID)
	if err != nil {
		return nil, err
	}

	wasmExecutor, err := wasm.NewExecutor()
	if err != nil {
		return nil, err
	}

	return provider.NewMappedProvider(map[string]executor.Executor{
		models.EngineDocker: dockerExecutor,
		models.EngineWasm:   wasmExecutor,
	}), nil
}

// return noop executors for all engines
func NewNoopExecutors(config noop_executor.ExecutorConfig) executor.ExecutorProvider {
	noopExecutor := noop_executor.NewNoopExecutorWithConfig(config)
	return provider.NewNoopProvider[executor.Executor](noopExecutor)
}

type PluginExecutorOptions struct {
	Plugins []PluginExecutorManagerConfig
}

func NewPluginExecutorProvider(
	ctx context.Context,
	cm *system.CleanupManager,
	pluginOptions PluginExecutorOptions,
) (executor.ExecutorProvider, error) {
	pe := NewPluginExecutorManager()
	for _, cfg := range pluginOptions.Plugins {
		if err := pe.RegisterPlugin(cfg); err != nil {
			return nil, err
		}
	}
	if err := pe.Start(ctx); err != nil {
		return nil, err
	}

	cm.RegisterCallbackWithContext(pe.Stop)

	return pe, nil
}
