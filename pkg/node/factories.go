package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/authn/ask"
	"github.com/bacalhau-project/bacalhau/pkg/authn/challenge"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	baccrypto "github.com/bacalhau-project/bacalhau/pkg/lib/crypto"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
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
func NewStandardStorageProvidersFactory(cfg types.Bacalhau) StorageProvidersFactory {
	return StorageProvidersFactoryFunc(func(
		ctx context.Context,
		nodeConfig NodeConfig,
	) (storage.StorageProvider, error) {
		pr, err := executor_util.NewStandardStorageProvider(cfg)
		if err != nil {
			return nil, err
		}
		return provider.NewConfiguredProvider(pr, nodeConfig.BacalhauConfig.InputSources.Disabled), err
	})
}

func NewStandardExecutorsFactory(cfg types.EngineConfig) ExecutorsFactory {
	return ExecutorsFactoryFunc(
		func(ctx context.Context, nodeConfig NodeConfig) (executor.ExecProvider, error) {
			pr, err := executor_util.NewStandardExecutorProvider(
				cfg,
				executor_util.StandardExecutorOptions{
					DockerID: fmt.Sprintf("bacalhau-%s", nodeConfig.NodeID),
				},
			)
			if err != nil {
				return nil, err
			}
			return provider.NewConfiguredProvider(pr, nodeConfig.BacalhauConfig.Engines.Disabled), err
		})
}

func NewStandardPublishersFactory(
	cfg types.Bacalhau,
	nclPublisherProvider ncl.PublisherProvider,
) PublishersFactory {
	return PublishersFactoryFunc(
		func(
			ctx context.Context,
			nodeConfig NodeConfig) (publisher.PublisherProvider, error) {
			pr, err := publisher_util.NewPublisherProvider(ctx, cfg, nclPublisherProvider)
			if err != nil {
				return nil, err
			}
			return provider.NewConfiguredProvider(pr, nodeConfig.BacalhauConfig.Publishers.Disabled), err
		})
}

func NewStandardAuthenticatorsFactory(userKey *baccrypto.UserKey) AuthenticatorsFactory {
	return AuthenticatorsFactoryFunc(
		func(ctx context.Context, nodeConfig NodeConfig) (authn.Provider, error) {
			var allErr error

			// If any new users or oauth2 config is specified , disable legacy auth methods endpoint
			if len(nodeConfig.BacalhauConfig.API.Auth.Users) > 0 || nodeConfig.BacalhauConfig.API.Auth.Oauth2.ProviderID != "" {
				auths := make(map[string]authn.Authenticator, 1)
				return provider.NewMappedProvider(auths), allErr
			}

			auths := make(map[string]authn.Authenticator, len(nodeConfig.BacalhauConfig.API.Auth.Methods))
			for name, authnConfig := range nodeConfig.BacalhauConfig.API.Auth.Methods {
				switch authnConfig.Type {
				case string(authn.MethodTypeChallenge):
					methodPolicy, err := policy.FromPathOrDefault(authnConfig.PolicyPath, challenge.AnonymousModePolicy)
					if err != nil {
						allErr = errors.Join(allErr, err)
						continue
					}

					auths[name] = challenge.NewAuthenticator(
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

					auths[name] = ask.NewAuthenticator(
						methodPolicy,
						userKey.PrivateKey(),
						nodeConfig.NodeID,
					)
				default:
					allErr = errors.Join(allErr, fmt.Errorf("unknown authentication type: %q", authnConfig.Type))
				}
			}

			return provider.NewMappedProvider(auths), allErr
		},
	)
}
