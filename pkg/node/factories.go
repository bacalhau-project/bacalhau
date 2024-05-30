package node

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/authn/ask"
	"github.com/bacalhau-project/bacalhau/pkg/authn/challenge"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	baccrypto "github.com/bacalhau-project/bacalhau/pkg/lib/crypto"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
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
func NewStandardStorageProvidersFactory(cfg types.NodeConfig) StorageProvidersFactory {
	return StorageProvidersFactoryFunc(func(
		ctx context.Context,
		nodeConfig NodeConfig,
	) (storage.StorageProvider, error) {
		pr, err := executor_util.NewStandardStorageProvider(
			time.Duration(cfg.VolumeSizeRequestTimeout),
			time.Duration(cfg.DownloadURLRequestTimeout),
			cfg.DownloadURLRequestRetries,
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

func NewStandardExecutorsFactory(cfg types.DockerCacheConfig) ExecutorsFactory {
	return ExecutorsFactoryFunc(
		func(ctx context.Context, nodeConfig NodeConfig) (executor.ExecutorProvider, error) {
			pr, err := executor_util.NewStandardExecutorProvider(
				cfg,
				executor_util.StandardExecutorOptions{
					DockerID: fmt.Sprintf("bacalhau-%s", nodeConfig.NodeID),
				},
			)
			if err != nil {
				return nil, err
			}
			return provider.NewConfiguredProvider(pr, nodeConfig.DisabledFeatures.Engines), err
		})
}

func NewPluginExecutorFactory(pluginPath string) ExecutorsFactory {
	return ExecutorsFactoryFunc(
		func(ctx context.Context, nodeConfig NodeConfig) (executor.ExecutorProvider, error) {
			pr, err := executor_util.NewPluginExecutorProvider(
				ctx,
				nodeConfig.CleanupManager,
				executor_util.PluginExecutorOptions{
					Plugins: []executor_util.PluginExecutorManagerConfig{
						{
							Name:             models.EngineDocker,
							Path:             pluginPath,
							Command:          "bacalhau-docker-executor",
							ProtocolVersion:  1,
							MagicCookieKey:   "EXECUTOR_PLUGIN",
							MagicCookieValue: "bacalhau_executor",
						},
						{
							Name:             models.EngineWasm,
							Path:             pluginPath,
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

func NewStandardPublishersFactory(storagePath string) PublishersFactory {
	return PublishersFactoryFunc(
		func(
			ctx context.Context,
			nodeConfig NodeConfig) (publisher.PublisherProvider, error) {
			pr, err := publisher_util.NewPublisherProvider(
				ctx,
				storagePath,
				nodeConfig.CleanupManager,
				nodeConfig.IPFSClient,
				&nodeConfig.ComputeConfig.LocalPublisher,
			)
			if err != nil {
				return nil, err
			}
			return provider.NewConfiguredProvider(pr, nodeConfig.DisabledFeatures.Publishers), err
		})
}

func NewStandardAuthenticatorsFactory(userKeyPath string) AuthenticatorsFactory {
	return AuthenticatorsFactoryFunc(
		func(ctx context.Context, nodeConfig NodeConfig) (authn.Provider, error) {
			var allErr error
			privKey, allErr := loadUserIDKey(userKeyPath)
			if allErr != nil {
				return nil, allErr
			}

			authns := make(map[string]authn.Authenticator, len(nodeConfig.AuthConfig.Methods))
			for name, authnConfig := range nodeConfig.AuthConfig.Methods {
				switch authnConfig.Type {
				case authn.MethodTypeChallenge:
					methodPolicy, err := policy.FromPathOrDefault(authnConfig.PolicyPath, challenge.AnonymousModePolicy)
					if err != nil {
						allErr = errors.Join(allErr, err)
						continue
					}

					authns[name] = challenge.NewAuthenticator(
						methodPolicy,
						challenge.NewStringMarshaller(nodeConfig.NodeID),
						privKey,
						nodeConfig.NodeID,
					)
				case authn.MethodTypeAsk:
					methodPolicy, err := policy.FromPath(authnConfig.PolicyPath)
					if err != nil {
						allErr = errors.Join(allErr, err)
						continue
					}

					authns[name] = ask.NewAuthenticator(
						methodPolicy,
						privKey,
						nodeConfig.NodeID,
					)
				default:
					allErr = errors.Join(allErr, fmt.Errorf("unknown authentication type: %q", authnConfig.Type))
				}
			}

			return provider.NewMappedProvider(authns), allErr
		},
	)
}

// loadUserIDKey loads the user ID key from the path
func loadUserIDKey(path string) (*rsa.PrivateKey, error) {
	return baccrypto.LoadPKCS1KeyFile(path)
}
