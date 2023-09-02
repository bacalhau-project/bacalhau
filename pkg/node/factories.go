package node

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	publisher_util "github.com/bacalhau-project/bacalhau/pkg/publisher/util"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

// Interfaces to inject dependencies into the stack
type StorageProvidersFactory interface {
	Get(ctx context.Context, nodeConfig NodeConfig) (storage.StorageProvider, error)
}

type ExecutorsFactory interface {
	Get(ctx context.Context, nodeConfig NodeConfig) (executor.ExecutorProvider, error)
}

type PublishersFactory interface {
	Get(ctx context.Context, nodeConfig NodeConfig) (publisher.PublisherProvider, error)
}

// Functions that implement the factories for easier creation of new implementations
type StorageProvidersFactoryFunc func(ctx context.Context, nodeConfig NodeConfig) (storage.StorageProvider, error)

func (f StorageProvidersFactoryFunc) Get(ctx context.Context, nodeConfig NodeConfig) (storage.StorageProvider, error) {
	return f(ctx, nodeConfig)
}

type ExecutorsFactoryFunc func(
	ctx context.Context,
	nodeConfig NodeConfig,
) (executor.ExecutorProvider, error)

func (f ExecutorsFactoryFunc) Get(
	ctx context.Context,
	nodeConfig NodeConfig,
) (executor.ExecutorProvider, error) {
	return f(ctx, nodeConfig)
}

type PublishersFactoryFunc func(ctx context.Context, nodeConfig NodeConfig) (publisher.PublisherProvider, error)

func (f PublishersFactoryFunc) Get(ctx context.Context, nodeConfig NodeConfig) (publisher.PublisherProvider, error) {
	return f(ctx, nodeConfig)
}

// Standard implementations used in prod and when testing prod behavior
func NewStandardStorageProvidersFactory() StorageProvidersFactory {
	return StorageProvidersFactoryFunc(func(
		ctx context.Context,
		nodeConfig NodeConfig,
	) (storage.StorageProvider, error) {
		pr, err := executor_util.NewStandardStorageProvider(
			ctx,
			nodeConfig.CleanupManager,
			executor_util.StandardStorageProviderOptions{
				API:                   nodeConfig.IPFSClient,
				AllowListedLocalPaths: nodeConfig.AllowListedLocalPaths,
			},
		)
		if err != nil {
			return nil, err
		}
		return provider.NewConfiguredProvider(pr, nodeConfig.DisabledFeatures.Storages), err
	})
}

func NewStandardExecutorsFactory() ExecutorsFactory {
	return ExecutorsFactoryFunc(
		func(ctx context.Context, nodeConfig NodeConfig) (executor.ExecutorProvider, error) {
			pr, err := executor_util.NewStandardExecutorProvider(
				ctx,
				nodeConfig.CleanupManager,
				executor_util.StandardExecutorOptions{
					DockerID: fmt.Sprintf("bacalhau-%s", nodeConfig.Host.ID().String()),
				},
			)
			if err != nil {
				return nil, err
			}
			return provider.NewConfiguredProvider(pr, nodeConfig.DisabledFeatures.Engines), err
		})
}

func NewPluginExecutorFactory() ExecutorsFactory {
	return ExecutorsFactoryFunc(
		func(ctx context.Context, nodeConfig NodeConfig) (executor.ExecutorProvider, error) {
			pr, err := executor_util.NewPluginExecutorProvider(
				ctx,
				nodeConfig.CleanupManager,
				executor_util.PluginExecutorOptions{
					Plugins: []executor_util.PluginExecutorManagerConfig{
						{
							Name:             "Docker",
							Path:             config.GetExecutorPluginsPath(),
							Command:          "bacalhau-docker-executor",
							ProtocolVersion:  1,
							MagicCookieKey:   "EXECUTOR_PLUGIN",
							MagicCookieValue: "bacalhau_executor",
						},
						{
							Name:             "Wasm",
							Path:             config.GetExecutorPluginsPath(),
							Command:          "bacalhau-wasm-executor",
							ProtocolVersion:  1,
							MagicCookieKey:   "EXECUTOR_PLUGIN",
							MagicCookieValue: "bacalhau_executor",
						},
					},
				})
			if err != nil {
				return nil, err
			}
			return provider.NewConfiguredProvider(pr, nodeConfig.DisabledFeatures.Engines), err
		})
}

func NewStandardPublishersFactory() PublishersFactory {
	return PublishersFactoryFunc(
		func(
			ctx context.Context,
			nodeConfig NodeConfig) (publisher.PublisherProvider, error) {
			pr, err := publisher_util.NewIPFSPublishers(
				ctx,
				nodeConfig.CleanupManager,
				nodeConfig.IPFSClient,
			)
			if err != nil {
				return nil, err
			}
			return provider.NewConfiguredProvider(pr, nodeConfig.DisabledFeatures.Publishers), err
		})
}
