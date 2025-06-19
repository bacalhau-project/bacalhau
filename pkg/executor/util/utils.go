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
	local_storage "github.com/bacalhau-project/bacalhau/pkg/storage/local"
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

	if cfg.InputSources.IsNotDisabled(models.StorageSourceURL) {
		providers[models.StorageSourceURL] = tracing.Wrap(urldownload.NewStorage(
			time.Duration(cfg.InputSources.ReadTimeout),
			cfg.InputSources.MaxRetryCount,
		))
	}

	if cfg.InputSources.IsNotDisabled(models.StorageSourceInline) {
		providers[models.StorageSourceInline] = tracing.Wrap(inline.NewStorage())
	}

	if cfg.InputSources.IsNotDisabled(models.StorageSourceS3) {
		s3Cfg, err := s3helper.DefaultAWSConfig()
		if err != nil {
			return nil, err
		}
		clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
			AWSConfig: s3Cfg,
		})
		providers[models.StorageSourceS3] = tracing.Wrap(s3.NewStorage(
			time.Duration(cfg.InputSources.ReadTimeout),
			clientProvider,
		))
	}

	if cfg.InputSources.IsNotDisabled(models.StorageSourceLocal) {
		localStorage, err := local_storage.NewStorageProvider(
			local_storage.StorageProviderParams{
				AllowedPaths: local_storage.ParseAllowPaths(cfg.Compute.AllowListedLocalPaths),
			})
		if err != nil {
			return nil, err
		}

		// Register under "local" and "localDirectory" names
		providers[models.StorageSourceLocal] = localStorage
		providers[models.StorageSourceLocalDirectory] = localStorage
	}

	if cfg.InputSources.IsNotDisabled(models.StorageSourceIPFS) {
		if cfg.InputSources.Types.IPFS.Endpoint != "" {
			ipfsClient, err := ipfs.NewClient(context.Background(), cfg.InputSources.Types.IPFS.Endpoint)
			if err != nil {
				return nil, err
			}
			ipfsStorage, err := ipfs_storage.NewStorage(*ipfsClient, time.Duration(cfg.InputSources.ReadTimeout))
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
) (executor.ExecProvider, error) {
	providers := make(map[string]executor.Executor)

	if cfg.IsNotDisabled(models.EngineDocker) {
		var err error
		providers[models.EngineDocker], err = docker.NewExecutor(docker.ExecutorParams{
			ID:     executorOptions.DockerID,
			Config: cfg.Types.Docker,
		})
		if err != nil {
			return nil, err
		}
	}

	if cfg.IsNotDisabled(models.EngineWasm) {
		wasmExecutor, err := wasm.NewExecutor()
		if err != nil {
			return nil, err
		}
		providers[models.EngineWasm] = wasmExecutor
	}

	return provider.NewMappedProvider(providers), nil
}

// return noop executors for all engines
func NewNoopExecutors(config noop_executor.ExecutorConfig) executor.ExecProvider {
	noopExecutor := noop_executor.NewNoopExecutorWithConfig(config)
	return provider.NewNoopProvider[executor.Executor](noopExecutor)
}
