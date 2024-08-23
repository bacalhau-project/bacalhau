package util

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
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
	cfg types2.ExecutorsConfig,
	executorOptions StandardExecutorOptions,
) (executor.ExecutorProvider, error) {
	providers := make(map[string]executor.Executor)
	if cfg.Enabled(models.EngineDocker) {
		var dockerExecutor *docker.Executor
		if cfg.Installed(models.EngineDocker) {
			dockercfg, err := types2.DecodeProviderConfig[types2.Docker](cfg)
			if err != nil {
				return nil, err
			}
			dockerExecutor, err = docker.NewExecutor(executorOptions.DockerID, types.DockerCacheConfig{
				Size:      dockercfg.ManifestCache.Size,
				Duration:  types.Duration(dockercfg.ManifestCache.TTL),
				Frequency: types.Duration(dockercfg.ManifestCache.Refresh),
			})
			if err != nil {
				return nil, err
			}
		} else {
			var err error
			dockerExecutor, err = docker.NewExecutor(executorOptions.DockerID, types.DockerCacheConfig{
				Size:      1000,
				Duration:  types.Duration(1 * time.Hour),
				Frequency: types.Duration(1 * time.Hour),
			})
			if err != nil {
				return nil, err
			}
		}
		providers[models.EngineDocker] = dockerExecutor
	}

	// NB(forrest): wasm doesn't have a config, so just check that its enabled.
	if cfg.Enabled(models.EngineWasm) {
		wasmExecutor, err := wasm.NewExecutor()
		if err != nil {
			return nil, err
		}
		providers[models.EngineWasm] = wasmExecutor
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
