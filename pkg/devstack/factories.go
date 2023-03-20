package devstack

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	noop_publisher "github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	publisher_util "github.com/bacalhau-project/bacalhau/pkg/publisher/util"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	noop_storage "github.com/bacalhau-project/bacalhau/pkg/storage/noop"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	noop_verifier "github.com/bacalhau-project/bacalhau/pkg/verifier/noop"
	verifier_util "github.com/bacalhau-project/bacalhau/pkg/verifier/util"
)

// Noop implementations of node factories used to mock certain components, which is useful for testing.
func NewNoopNodeDependencyInjector() node.NodeDependencyInjector {
	return node.NodeDependencyInjector{
		StorageProvidersFactory: NewNoopStorageProvidersFactory(),
		ExecutorsFactory:        NewNoopExecutorsFactory(),
		VerifiersFactory:        NewNoopVerifiersFactory(),
		PublishersFactory:       NewNoopPublishersFactory(),
	}
}

type NoopStorageProvidersFactory struct {
	config noop_storage.StorageConfig
}

func (f *NoopStorageProvidersFactory) Get(
	ctx context.Context,
	nodeConfig node.NodeConfig) (storage.StorageProvider, error) {
	return executor_util.NewNoopStorageProvider(ctx, nodeConfig.CleanupManager, f.config)
}

func NewNoopStorageProvidersFactory() *NoopStorageProvidersFactory {
	return NewNoopStorageProvidersFactoryWithConfig(noop_storage.StorageConfig{})
}

func NewNoopStorageProvidersFactoryWithConfig(config noop_storage.StorageConfig) *NoopStorageProvidersFactory {
	return &NoopStorageProvidersFactory{config: config}
}

type NoopExecutorsFactory struct {
	config noop_executor.ExecutorConfig
}

func (f *NoopExecutorsFactory) Get(
	ctx context.Context,
	nodeConfig node.NodeConfig) (executor.ExecutorProvider, error) {
	return executor_util.NewNoopExecutors(f.config), nil
}

func NewNoopExecutorsFactory() *NoopExecutorsFactory {
	return NewNoopExecutorsFactoryWithConfig(noop_executor.ExecutorConfig{})
}

func NewNoopExecutorsFactoryWithConfig(config noop_executor.ExecutorConfig) *NoopExecutorsFactory {
	return &NoopExecutorsFactory{config: config}
}

type NoopVerifiersFactory struct {
	config noop_verifier.VerifierConfig
}

func (f *NoopVerifiersFactory) Get(
	ctx context.Context,
	nodeConfig node.NodeConfig) (verifier.VerifierProvider, error) {
	return verifier_util.NewNoopVerifiers(ctx, nodeConfig.CleanupManager, f.config)
}

func NewNoopVerifiersFactory() *NoopVerifiersFactory {
	return NewNoopVerifiersFactoryWithConfig(noop_verifier.VerifierConfig{})
}

func NewNoopVerifiersFactoryWithConfig(config noop_verifier.VerifierConfig) *NoopVerifiersFactory {
	return &NoopVerifiersFactory{config: config}
}

type NoopPublishersFactory struct {
	config noop_publisher.PublisherConfig
}

func (f *NoopPublishersFactory) Get(
	ctx context.Context,
	nodeConfig node.NodeConfig) (publisher.PublisherProvider, error) {
	return publisher_util.NewNoopPublishers(ctx, nodeConfig.CleanupManager, f.config)
}

func NewNoopPublishersFactory() *NoopPublishersFactory {
	return NewNoopPublishersFactoryWithConfig(noop_publisher.PublisherConfig{})
}

func NewNoopPublishersFactoryWithConfig(config noop_publisher.PublisherConfig) *NoopPublishersFactory {
	return &NoopPublishersFactory{config: config}
}
