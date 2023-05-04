package node

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	publisher_util "github.com/bacalhau-project/bacalhau/pkg/publisher/util"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	verifier_util "github.com/bacalhau-project/bacalhau/pkg/verifier/util"
)

// Interfaces to inject dependencies into the stack
type StorageProvidersFactory interface {
	Get(ctx context.Context, nodeConfig NodeConfig) (storage.StorageProvider, error)
}

type ExecutorsFactory interface {
	Get(ctx context.Context, nodeConfig NodeConfig, storages storage.StorageProvider) (executor.ExecutorProvider, error)
}

type VerifiersFactory interface {
	Get(ctx context.Context, nodeConfig NodeConfig, publishers publisher.PublisherProvider) (verifier.VerifierProvider, error)
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
	storages storage.StorageProvider,
) (executor.ExecutorProvider, error)

func (f ExecutorsFactoryFunc) Get(
	ctx context.Context,
	nodeConfig NodeConfig,
	storages storage.StorageProvider,
) (executor.ExecutorProvider, error) {
	return f(ctx, nodeConfig, storages)
}

type VerifiersFactoryFunc func(
	ctx context.Context,
	nodeConfig NodeConfig,
	publishers publisher.PublisherProvider,
) (verifier.VerifierProvider, error)

func (f VerifiersFactoryFunc) Get(
	ctx context.Context,
	nodeConfig NodeConfig,
	publishers publisher.PublisherProvider,
) (verifier.VerifierProvider, error) {
	return f(ctx, nodeConfig, publishers)
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
		provider, err := executor_util.NewStandardStorageProvider(
			ctx,
			nodeConfig.CleanupManager,
			executor_util.StandardStorageProviderOptions{
				API:                   nodeConfig.IPFSClient,
				EstuaryAPIKey:         nodeConfig.EstuaryAPIKey,
				FilecoinUnsealedPath:  nodeConfig.FilecoinUnsealedPath,
				AllowListedLocalPaths: nodeConfig.AllowListedLocalPaths,
			},
		)
		if err != nil {
			return nil, err
		}
		return model.NewConfiguredProvider(provider, nodeConfig.DisabledFeatures.Storages), err
	})
}

func NewStandardExecutorsFactory() ExecutorsFactory {
	return ExecutorsFactoryFunc(
		func(ctx context.Context, nodeConfig NodeConfig, storages storage.StorageProvider) (executor.ExecutorProvider, error) {
			provider, err := executor_util.NewStandardExecutorProvider(
				ctx,
				nodeConfig.CleanupManager,
				storages,
				executor_util.StandardExecutorOptions{
					DockerID: fmt.Sprintf("bacalhau-%s", nodeConfig.Host.ID().String()),
				},
			)
			if err != nil {
				return nil, err
			}
			return model.NewConfiguredProvider(provider, nodeConfig.DisabledFeatures.Engines), err
		})
}

func NewStandardVerifiersFactory() VerifiersFactory {
	return VerifiersFactoryFunc(
		func(ctx context.Context, nodeConfig NodeConfig, publishers publisher.PublisherProvider) (verifier.VerifierProvider, error) {
			encrypter := verifier.NewEncrypter(nodeConfig.Host.Peerstore().PrivKey(nodeConfig.Host.ID()))
			provider, err := verifier_util.NewStandardVerifiers(
				ctx,
				nodeConfig.CleanupManager,
				publishers,
				encrypter.Encrypt,
				encrypter.Decrypt,
			)
			if err != nil {
				return nil, err
			}
			return model.NewConfiguredProvider(provider, nodeConfig.DisabledFeatures.Verifiers), err
		})
}

func NewStandardPublishersFactory() PublishersFactory {
	return PublishersFactoryFunc(
		func(
			ctx context.Context,
			nodeConfig NodeConfig) (publisher.PublisherProvider, error) {
			provider, err := publisher_util.NewIPFSPublishers(
				ctx,
				nodeConfig.CleanupManager,
				nodeConfig.IPFSClient,
				nodeConfig.EstuaryAPIKey,
				nodeConfig.LotusConfig,
			)
			if err != nil {
				return nil, err
			}
			return model.NewConfiguredProvider(provider, nodeConfig.DisabledFeatures.Publishers), err
		})
}
