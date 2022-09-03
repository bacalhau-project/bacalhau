package node

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	publisher_util "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
)

// Interfaces to inject dependencies into the stack
type StorageProvidersFactory interface {
	Get(ctx context.Context, nodeConfig NodeConfig) (map[model.StorageSourceType]storage.StorageProvider, error)
}

type ExecutorsFactory interface {
	Get(ctx context.Context, nodeConfig NodeConfig) (map[model.EngineType]executor.Executor, error)
}

type VerifiersFactory interface {
	Get(ctx context.Context,
		nodeConfig NodeConfig,
		controller *controller.Controller) (map[model.VerifierType]verifier.Verifier, error)
}

type PublishersFactory interface {
	Get(ctx context.Context,
		nodeConfig NodeConfig,
		controller *controller.Controller) (map[model.PublisherType]publisher.Publisher, error)
}

// Functions that implement the factories for easier creation of new implementations
type StorageProvidersFactoryFunc func(
	ctx context.Context, nodeConfig NodeConfig) (map[model.StorageSourceType]storage.StorageProvider, error)

func (f StorageProvidersFactoryFunc) Get(
	ctx context.Context, nodeConfig NodeConfig) (map[model.StorageSourceType]storage.StorageProvider, error) {
	return f(ctx, nodeConfig)
}

type ExecutorsFactoryFunc func(ctx context.Context, nodeConfig NodeConfig) (map[model.EngineType]executor.Executor, error)

func (f ExecutorsFactoryFunc) Get(ctx context.Context, nodeConfig NodeConfig) (map[model.EngineType]executor.Executor, error) {
	return f(ctx, nodeConfig)
}

type VerifiersFactoryFunc func(
	ctx context.Context,
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.VerifierType]verifier.Verifier, error)

func (f VerifiersFactoryFunc) Get(
	ctx context.Context,
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.VerifierType]verifier.Verifier, error) {
	return f(ctx, nodeConfig, controller)
}

type PublishersFactoryFunc func(
	ctx context.Context,
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.PublisherType]publisher.Publisher, error)

func (f PublishersFactoryFunc) Get(
	ctx context.Context,
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.PublisherType]publisher.Publisher, error) {
	return f(ctx, nodeConfig, controller)
}

// Standard implementations used in prod and when testing prod behavior
type StandardStorageProvidersFactory struct{}

func (f *StandardStorageProvidersFactory) Get(
	ctx context.Context,
	nodeConfig NodeConfig) (map[model.StorageSourceType]storage.StorageProvider, error) {
	return executor_util.NewStandardStorageProviders(
		ctx,
		nodeConfig.CleanupManager,
		executor_util.StandardStorageProviderOptions{
			IPFSMultiaddress:     nodeConfig.IPFSClient.APIAddress(),
			FilecoinUnsealedPath: nodeConfig.FilecoinUnsealedPath,
		},
	)
}

func NewStandardStorageProvidersFactory() *StandardStorageProvidersFactory {
	return &StandardStorageProvidersFactory{}
}

type StandardExecutorsFactory struct{}

func (f *StandardExecutorsFactory) Get(
	ctx context.Context,
	nodeConfig NodeConfig) (map[model.EngineType]executor.Executor, error) {
	return executor_util.NewStandardExecutors(
		ctx,
		nodeConfig.CleanupManager,
		executor_util.StandardExecutorOptions{
			DockerID:   fmt.Sprintf("bacalhau-%s", nodeConfig.HostID),
			IsBadActor: nodeConfig.IsBadActor,
			Storage: executor_util.StandardStorageProviderOptions{
				IPFSMultiaddress:     nodeConfig.IPFSClient.APIAddress(),
				FilecoinUnsealedPath: nodeConfig.FilecoinUnsealedPath,
			},
		},
	)
}

func NewStandardExecutorsFactory() *StandardExecutorsFactory {
	return &StandardExecutorsFactory{}
}

type StandardVerifiersFactory struct{}

func (f *StandardVerifiersFactory) Get(
	ctx context.Context,
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.VerifierType]verifier.Verifier, error) {
	return verifier_util.NewStandardVerifiers(
		ctx,
		nodeConfig.CleanupManager,
		controller.GetStateResolver(),
		nodeConfig.Transport.Encrypt,
		nodeConfig.Transport.Decrypt,
	)
}

func NewStandardVerifiersFactory() *StandardVerifiersFactory {
	return &StandardVerifiersFactory{}
}

type StandardPublishersFactory struct{}

func (f *StandardPublishersFactory) Get(
	ctx context.Context,
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.PublisherType]publisher.Publisher, error) {
	return publisher_util.NewIPFSPublishers(ctx, nodeConfig.CleanupManager, controller.GetStateResolver(), nodeConfig.IPFSClient.APIAddress())
}

func NewStandardPublishersFactory() *StandardPublishersFactory {
	return &StandardPublishersFactory{}
}

// Noop implementations used to mock certain components, which is useful for testing.
// type NoopStorageProvidersFactory struct {
// 	config noop_storage.StorageConfig
// }

// func (f *NoopStorageProvidersFactory) Get(
// 	ctx context.Context,
// 	nodeConfig NodeConfig) (map[model.StorageSourceType]storage.StorageProvider, error) {
// 	return executor_util.NewNoopStorageProviders(ctx, nodeConfig.CleanupManager, f.config)
// }

// func NewNoopStorageProvidersFactory() *NoopStorageProvidersFactory {
// 	return NewNoopStorageProvidersFactoryWithConfig(noop_storage.StorageConfig{})
// }

// func NewNoopStorageProvidersFactoryWithConfig(config noop_storage.StorageConfig) *NoopStorageProvidersFactory {
// 	return &NoopStorageProvidersFactory{config: config}
// }

// type NoopExecutorsFactory struct {
// 	config noop_executor.ExecutorConfig
// }

// func (f *NoopExecutorsFactory) Get(
// 	ctx context.Context,
// 	nodeConfig NodeConfig) (map[model.EngineType]executor.Executor, error) {
// 	return executor_util.NewNoopExecutors(ctx, nodeConfig.CleanupManager, f.config)
// }

// func NewNoopExecutorsFactory() *NoopExecutorsFactory {
// 	return NewNoopExecutorsFactoryWithConfig(noop_executor.ExecutorConfig{})
// }

// func NewNoopExecutorsFactoryWithConfig(config noop_executor.ExecutorConfig) *NoopExecutorsFactory {
// 	return &NoopExecutorsFactory{config: config}
// }

// type NoopVerifiersFactory struct{}

// func (f *NoopVerifiersFactory) Get(
// 	ctx context.Context,
// 	nodeConfig NodeConfig,
// 	controller *controller.Controller) (map[model.VerifierType]verifier.Verifier, error) {
// 	return verifier_util.NewNoopVerifiers(ctx, nodeConfig.CleanupManager, controller.GetStateResolver())
// }

// func NewNoopVerifiersFactory() *NoopVerifiersFactory {
// 	return &NoopVerifiersFactory{}
// }

// type NoopPublishersFactory struct{}

// func (f *NoopPublishersFactory) Get(
// 	ctx context.Context,
// 	nodeConfig NodeConfig,
// 	controller *controller.Controller) (map[model.PublisherType]publisher.Publisher, error) {
// 	return publisher_util.NewNoopPublishers(ctx, nodeConfig.CleanupManager, controller.GetStateResolver())
// }

// func NewNoopPublishersFactory() *NoopPublishersFactory {
// 	return &NoopPublishersFactory{}
// }
