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
	Get(ctx context.Context, nodeConfig NodeConfig) (executor.ExecutorProvider, error)
}

type VerifiersFactory interface {
	Get(ctx context.Context,
		nodeConfig NodeConfig) (verifier.VerifierProvider, error)
}

type PublishersFactory interface {
	Get(ctx context.Context,
		nodeConfig NodeConfig) (publisher.PublisherProvider, error)
}

// Functions that implement the factories for easier creation of new implementations
type StorageProvidersFactoryFunc func(
	ctx context.Context, nodeConfig NodeConfig) (storage.StorageProvider, error)

func (f StorageProvidersFactoryFunc) Get(
	ctx context.Context, nodeConfig NodeConfig) (storage.StorageProvider, error) {
	return f(ctx, nodeConfig)
}

type ExecutorsFactoryFunc func(ctx context.Context, nodeConfig NodeConfig) (executor.ExecutorProvider, error)

func (f ExecutorsFactoryFunc) Get(ctx context.Context, nodeConfig NodeConfig) (executor.ExecutorProvider, error) {
	return f(ctx, nodeConfig)
}

type VerifiersFactoryFunc func(
	ctx context.Context,
	nodeConfig NodeConfig) (verifier.VerifierProvider, error)

func (f VerifiersFactoryFunc) Get(
	ctx context.Context,
	nodeConfig NodeConfig) (verifier.VerifierProvider, error) {
	return f(ctx, nodeConfig)
}

type PublishersFactoryFunc func(
	ctx context.Context,
	nodeConfig NodeConfig) (publisher.PublisherProvider, error)

func (f PublishersFactoryFunc) Get(
	ctx context.Context,
	nodeConfig NodeConfig) (publisher.PublisherProvider, error) {
	return f(ctx, nodeConfig)
}

// Standard implementations used in prod and when testing prod behavior
type StandardStorageProvidersFactory struct{}

func (f *StandardStorageProvidersFactory) Get(
	ctx context.Context,
	nodeConfig NodeConfig) (storage.StorageProvider, error) {
	provider, err := executor_util.NewStandardStorageProvider(
		ctx,
		nodeConfig.CleanupManager,
		executor_util.StandardStorageProviderOptions{
			API:                  nodeConfig.IPFSClient,
			FilecoinUnsealedPath: nodeConfig.FilecoinUnsealedPath,
		},
	)
	return model.NewConfiguredProvider[model.StorageSourceType, storage.Storage](
		provider,
		nodeConfig.DisabledFeatures.Storages,
	), err
}

func NewStandardStorageProvidersFactory() *StandardStorageProvidersFactory {
	return &StandardStorageProvidersFactory{}
}

type StandardExecutorsFactory struct{}

func (f *StandardExecutorsFactory) Get(ctx context.Context, nodeConfig NodeConfig) (executor.ExecutorProvider, error) {
	provider, err := executor_util.NewStandardExecutorProvider(
		ctx,
		nodeConfig.CleanupManager,
		executor_util.StandardExecutorOptions{
			DockerID: fmt.Sprintf("bacalhau-%s", nodeConfig.Host.ID().String()),
			Storage: executor_util.StandardStorageProviderOptions{
				API:                  nodeConfig.IPFSClient,
				FilecoinUnsealedPath: nodeConfig.FilecoinUnsealedPath,
				EstuaryAPIKey:        nodeConfig.EstuaryAPIKey,
			},
		},
	)
	return model.NewConfiguredProvider[model.Engine, executor.Executor](provider, nodeConfig.DisabledFeatures.Engines), err
}

func NewStandardExecutorsFactory() *StandardExecutorsFactory {
	return &StandardExecutorsFactory{}
}

type StandardVerifiersFactory struct{}

func (f *StandardVerifiersFactory) Get(
	ctx context.Context,
	nodeConfig NodeConfig) (verifier.VerifierProvider, error) {
	encrypter := verifier.NewEncrypter(nodeConfig.Host.Peerstore().PrivKey(nodeConfig.Host.ID()))
	provider, err := verifier_util.NewStandardVerifiers(
		ctx,
		nodeConfig.CleanupManager,
		encrypter.Encrypt,
		encrypter.Decrypt,
	)
	return model.NewConfiguredProvider[model.Verifier, verifier.Verifier](provider, nodeConfig.DisabledFeatures.Verifiers), err
}

func NewStandardVerifiersFactory() *StandardVerifiersFactory {
	return &StandardVerifiersFactory{}
}

type StandardPublishersFactory struct{}

func (f *StandardPublishersFactory) Get(
	ctx context.Context,
	nodeConfig NodeConfig) (publisher.PublisherProvider, error) {
	provider, err := publisher_util.NewIPFSPublishers(
		ctx,
		nodeConfig.CleanupManager,
		nodeConfig.IPFSClient,
		nodeConfig.EstuaryAPIKey,
		nodeConfig.LotusConfig,
	)
	return model.NewConfiguredProvider[model.Publisher, publisher.Publisher](provider, nodeConfig.DisabledFeatures.Publishers), err
}

func NewStandardPublishersFactory() *StandardPublishersFactory {
	return &StandardPublishersFactory{}
}
