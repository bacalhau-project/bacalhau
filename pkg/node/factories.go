package node

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/authn/challenge"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	publisher_util "github.com/bacalhau-project/bacalhau/pkg/publisher/util"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

// Interfaces to inject dependencies into the stack
type Factory[P provider.Providable] interface {
	Get(ctx context.Context, nodeConfig NodeConfig) (provider.Provider[P], error)
}

type (
	StorageProvidersFactory = Factory[storage.Storage]
	ExecutorsFactory        = Factory[executor.Executor]
	PublishersFactory       = Factory[publisher.Publisher]
	AuthenticatorsFactory   = Factory[authn.Authenticator]
)

// Functions that implement the factories for easier creation of new implementations
type FactoryFunc[P provider.Providable] func(ctx context.Context, nodeConfig NodeConfig) (provider.Provider[P], error)

func (f FactoryFunc[P]) Get(ctx context.Context, nodeConfig NodeConfig) (provider.Provider[P], error) {
	return f(ctx, nodeConfig)
}

type (
	StorageProvidersFactoryFunc = FactoryFunc[storage.Storage]
	ExecutorsFactoryFunc        = FactoryFunc[executor.Executor]
	PublishersFactoryFunc       = FactoryFunc[publisher.Publisher]
	AuthenticatorsFactoryFunc   = FactoryFunc[authn.Authenticator]
)

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
							Name:             models.EngineDocker,
							Path:             config.GetExecutorPluginsPath(),
							Command:          "bacalhau-docker-executor",
							ProtocolVersion:  1,
							MagicCookieKey:   "EXECUTOR_PLUGIN",
							MagicCookieValue: "bacalhau_executor",
						},
						{
							Name:             models.EngineWasm,
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

func NewStandardAuthenticatorsFactory() AuthenticatorsFactory {
	return AuthenticatorsFactoryFunc(
		func(ctx context.Context, nodeConfig NodeConfig) (authn.Provider, error) {
			privKey, err := config.GetClientPrivateKey()
			if err != nil {
				return nil, err
			}

			return provider.NewMappedProvider(
				map[string]authn.Authenticator{
					"ClientKey": challenge.NewAuthenticator(
						challenge.AnonymousModePolicy,
						nodeConfig.Host.ID(),
						privKey,
						nodeConfig.Host.ID().String(),
					),
				},
			), nil
		},
	)
}
