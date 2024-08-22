package node

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/authn/ask"
	"github.com/bacalhau-project/bacalhau/pkg/authn/challenge"
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
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
func NewStandardStorageProvidersFactory(cfg types2.InputSourcesConfig) StorageProvidersFactory {
	return StorageProvidersFactoryFunc(func(
		ctx context.Context,
		nodeConfig NodeConfig,
	) (storage.StorageProvider, error) {
		ipfsConnect := ""
		if cfg.Enabled(types2.KindStorageIPFS) && cfg.HasConfig(types2.KindStorageIPFS) {
			ipfscfg, err := types2.DecodeProviderConfig[types2.IpfsInputSourceConfig](cfg)
			if err != nil {
				return nil, err
			}
			ipfsConnect = ipfscfg.Connect
		}
		// TODO(forrest): pass the config down through this method and only enable the non-disabled one.
		pr, err := executor_util.NewStandardStorageProvider(
			// NB(forrest): defaults taken from v1 config
			2*time.Minute,
			5*time.Minute,
			3,

			executor_util.StandardStorageProviderOptions{
				IPFSConnect:           ipfsConnect,
				AllowListedLocalPaths: nodeConfig.AllowListedLocalPaths,
			},
		)
		if err != nil {
			return nil, err
		}
		return provider.NewConfiguredProvider(pr, nodeConfig.DisabledFeatures.Storages), err
	})
}

func NewStandardExecutorsFactory(cfg types2.ExecutorsConfig) ExecutorsFactory {
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

func NewStandardPublishersFactory(cfg types2.Bacalhau) PublishersFactory {
	return PublishersFactoryFunc(
		func(
			ctx context.Context,
			nodeConfig NodeConfig) (publisher.PublisherProvider, error) {
			executionDir, err := cfg.ExecutionDir()
			if err != nil {
				return nil, err
			}
			pr, err := publisher_util.NewPublisherProvider(
				ctx,
				executionDir,
				nodeConfig.CleanupManager,
				cfg.Publishers,
				&nodeConfig.ComputeConfig.LocalPublisher,
			)
			if err != nil {
				return nil, err
			}
			return provider.NewConfiguredProvider(pr, nodeConfig.DisabledFeatures.Publishers), err
		})
}

func NewStandardAuthenticatorsFactory(userKey *baccrypto.UserKey) AuthenticatorsFactory {
	return AuthenticatorsFactoryFunc(
		func(ctx context.Context, nodeConfig NodeConfig) (authn.Provider, error) {
			var allErr error

			authns := make(map[string]authn.Authenticator, len(nodeConfig.AuthConfig.Methods))
			for name, authnConfig := range nodeConfig.AuthConfig.Methods {
				switch authnConfig.Type {
				case string(authn.MethodTypeChallenge):
					methodPolicy, err := policy.FromPathOrDefault(authnConfig.PolicyPath, challenge.AnonymousModePolicy)
					if err != nil {
						allErr = errors.Join(allErr, err)
						continue
					}

					authns[name] = challenge.NewAuthenticator(
						methodPolicy,
						challenge.NewStringMarshaller(nodeConfig.NodeID),
						userKey.PrivateKey(),
						nodeConfig.NodeID,
					)
				case string(authn.MethodTypeAsk):
					methodPolicy, err := policy.FromPath(authnConfig.PolicyPath)
					if err != nil {
						allErr = errors.Join(allErr, err)
						continue
					}

					authns[name] = ask.NewAuthenticator(
						methodPolicy,
						userKey.PrivateKey(),
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
