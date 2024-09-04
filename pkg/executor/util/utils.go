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

func NewStandardStorageProvider(cfg types.Bacalhau) (storage.StorageProvider, error) {
	providers := make(map[string]storage.Storage)

	// NB(forrest): defaults taken from v1 config
	getVolumeTimeout := 2 * time.Minute
	urlDownloadTimeout := 5 * time.Minute
	urlMaxRetries := 3

	if cfg.InputSources.Enabled(models.StorageSourceURL) {
		providers[models.StorageSourceURL] = tracing.Wrap(urldownload.NewStorage(urlDownloadTimeout, urlMaxRetries))
	}

	if cfg.InputSources.Enabled(models.StorageSourceInline) {
		providers[models.StorageSourceInline] = tracing.Wrap(inline.NewStorage())
	}

	if cfg.InputSources.Enabled(models.StorageSourceS3) {
		s3Cfg, err := s3helper.DefaultAWSConfig()
		if err != nil {
			return nil, err
		}
		clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
			AWSConfig: s3Cfg,
		})
		providers[models.StorageSourceS3] = tracing.Wrap(s3.NewStorage(getVolumeTimeout, clientProvider))
	}

	if cfg.InputSources.Enabled(models.StorageSourceLocalDirectory) {
		if len(cfg.Compute.AllowListedLocalPaths) > 0 {
			var err error
			providers[models.StorageSourceLocalDirectory], err = localdirectory.NewStorageProvider(
				localdirectory.StorageProviderParams{
					AllowedPaths: localdirectory.ParseAllowPaths(cfg.Compute.AllowListedLocalPaths),
				})
			if err != nil {
				return nil, err
			}
		}
	}

	if cfg.InputSources.Enabled(models.StorageSourceIPFS) {
		if cfg.InputSources.Types.IPFS.Installed() {
			ipfsClient, err := ipfs.NewClient(context.Background(), cfg.InputSources.Types.IPFS.Endpoint)
			if err != nil {
				return nil, err
			}
			ipfsStorage, err := ipfs_storage.NewStorage(*ipfsClient, getVolumeTimeout)
			if err != nil {
				return nil, err
			}
			providers[models.StorageSourceIPFS] = tracing.Wrap(ipfsStorage)

		}
	}

	return provider.NewMappedProvider(providers), nil
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
	cfg types.EngineConfig,
	executorOptions StandardExecutorOptions,
) (executor.ExecutorProvider, error) {
	providers := make(map[string]executor.Executor)

	if cfg.Enabled(models.EngineDocker) {
		cacheConfig := types.DockerManifestCache{
			Size:    1000,
			TTL:     types.Duration(1 * time.Hour),
			Refresh: types.Duration(1 * time.Hour),
		}
		if cfg.Docker.Installed() {
			cacheConfig = types.DockerManifestCache{
				Size:    cfg.Docker.ManifestCache.Size,
				TTL:     cfg.Docker.ManifestCache.TTL,
				Refresh: cfg.Docker.ManifestCache.Refresh,
			}
		}
		var err error
		providers[models.EngineDocker], err = docker.NewExecutor(executorOptions.DockerID, cacheConfig)
		if err != nil {
			return nil, err
		}
	}

	if cfg.Enabled(models.EngineWasm) {
		if cfg.WASM.Installed() {
			wasmExecutor, err := wasm.NewExecutor()
			if err != nil {
				return nil, err
			}
			providers[models.EngineWasm] = wasmExecutor
		}
	}

	return provider.NewMappedProvider(providers), nil
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
