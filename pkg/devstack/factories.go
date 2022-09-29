package devstack

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/localdb"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	publisher_util "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	noop_storage "github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
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

func NewNoopNode(
	ctx context.Context,
	config node.NodeConfig) (*node.Node, error) {
	return node.NewNode(ctx, config, NewNoopNodeDependencyInjector())
}

type NoopStorageProvidersFactory struct {
	config noop_storage.StorageConfig
}

func (f *NoopStorageProvidersFactory) Get(
	ctx context.Context,
	nodeConfig node.NodeConfig) (map[model.StorageSourceType]storage.StorageProvider, error) {
	return executor_util.NewNoopStorageProviders(ctx, nodeConfig.CleanupManager, f.config)
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
	nodeConfig node.NodeConfig) (map[model.Engine]executor.Executor, error) {
	return executor_util.NewNoopExecutors(ctx, nodeConfig.CleanupManager, f.config)
}

func NewNoopExecutorsFactory() *NoopExecutorsFactory {
	return NewNoopExecutorsFactoryWithConfig(noop_executor.ExecutorConfig{})
}

func NewNoopExecutorsFactoryWithConfig(config noop_executor.ExecutorConfig) *NoopExecutorsFactory {
	return &NoopExecutorsFactory{config: config}
}

type NoopVerifiersFactory struct{}

func (f *NoopVerifiersFactory) Get(
	ctx context.Context,
<<<<<<< HEAD
	nodeConfig node.NodeConfig,
	controller *controller.Controller) (map[model.Verifier]verifier.Verifier, error) {
	return verifier_util.NewNoopVerifiers(ctx, nodeConfig.CleanupManager, controller.GetStateResolver())
||||||| 5d1cca3e
	nodeConfig node.NodeConfig,
	controller *controller.Controller) (map[model.VerifierType]verifier.Verifier, error) {
	return verifier_util.NewNoopVerifiers(ctx, nodeConfig.CleanupManager, controller.GetStateResolver())
=======
	nodeConfig node.NodeConfig) (map[model.VerifierType]verifier.Verifier, error) {
	return verifier_util.NewNoopVerifiers(ctx, nodeConfig.CleanupManager, localdb.GetStateResolver(nodeConfig.LocalDB))
>>>>>>> main
}

func NewNoopVerifiersFactory() *NoopVerifiersFactory {
	return &NoopVerifiersFactory{}
}

type NoopPublishersFactory struct{}

func (f *NoopPublishersFactory) Get(
	ctx context.Context,
<<<<<<< HEAD
	nodeConfig node.NodeConfig,
	controller *controller.Controller) (map[model.Publisher]publisher.Publisher, error) {
	return publisher_util.NewNoopPublishers(ctx, nodeConfig.CleanupManager, controller.GetStateResolver())
||||||| 5d1cca3e
	nodeConfig node.NodeConfig,
	controller *controller.Controller) (map[model.PublisherType]publisher.Publisher, error) {
	return publisher_util.NewNoopPublishers(ctx, nodeConfig.CleanupManager, controller.GetStateResolver())
=======
	nodeConfig node.NodeConfig) (map[model.PublisherType]publisher.Publisher, error) {
	return publisher_util.NewNoopPublishers(ctx, nodeConfig.CleanupManager, localdb.GetStateResolver(nodeConfig.LocalDB))
>>>>>>> main
}

func NewNoopPublishersFactory() *NoopPublishersFactory {
	return &NoopPublishersFactory{}
}
