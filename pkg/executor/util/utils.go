package util

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
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
	"github.com/bacalhau-project/bacalhau/pkg/storage/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage/tracing"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type StandardStorageProviderOptions struct {
	IPFSConnect           string
	DownloadPath          string
	AllowListedLocalPaths []string
}

type StandardExecutorOptions struct {
	DockerID string
}

func NewStandardStorageProvider(
	getVolumeTimeout time.Duration,
	urlDownloadTimeout time.Duration,
	urlMaxRetries int,
	options StandardStorageProviderOptions,
) (storage.StorageProvider, error) {
	urlDownloadStorage := urldownload.NewStorage(urlDownloadTimeout, urlMaxRetries)

	inlineStorage := inline.NewStorage()

	s3Storage, err := configureS3StorageProvider(getVolumeTimeout)
	if err != nil {
		return nil, err
	}

	localDirectoryStorage, err := localdirectory.NewStorageProvider(localdirectory.StorageProviderParams{
		AllowedPaths: localdirectory.ParseAllowPaths(options.AllowListedLocalPaths),
	})
	if err != nil {
		return nil, err
	}

	providers := map[string]storage.Storage{
		models.StorageSourceURL:            tracing.Wrap(urlDownloadStorage),
		models.StorageSourceInline:         tracing.Wrap(inlineStorage),
		models.StorageSourceS3:             tracing.Wrap(s3Storage),
		models.StorageSourceLocalDirectory: tracing.Wrap(localDirectoryStorage),
	}

	if options.IPFSConnect != "" {
		ipfsClient, err := ipfs.NewClient(context.Background(), options.IPFSConnect)
		if err != nil {
			return nil, err
		}
		ipfsStorage, err := ipfs_storage.NewStorage(*ipfsClient, getVolumeTimeout)
		if err != nil {
			return nil, err
		}
		providers[models.StorageSourceIPFS] = tracing.Wrap(ipfsStorage)
	}
	return provider.NewMappedProvider(providers), nil
}

func configureS3StorageProvider(getVolumeTimeout time.Duration) (*s3.StorageProvider, error) {
	cfg, err := s3helper.DefaultAWSConfig()
	if err != nil {
		return nil, err
	}
	clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
		AWSConfig: cfg,
	})
	s3Storage := s3.NewStorage(getVolumeTimeout, clientProvider)
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
	dockerCacheCfg types.DockerCacheConfig,
	executorOptions StandardExecutorOptions,
) (executor.ExecutorProvider, error) {
	dockerExecutor, err := docker.NewExecutor(executorOptions.DockerID, dockerCacheCfg)
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
