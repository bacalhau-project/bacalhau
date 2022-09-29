package node

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/localdb"

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
	Get(ctx context.Context, nodeConfig NodeConfig) (map[model.Engine]executor.Executor, error)
}

type VerifiersFactory interface {
	Get(ctx context.Context,
<<<<<<< HEAD
		nodeConfig NodeConfig,
		controller *controller.Controller) (map[model.Verifier]verifier.Verifier, error)
||||||| 5d1cca3e
		nodeConfig NodeConfig,
		controller *controller.Controller) (map[model.VerifierType]verifier.Verifier, error)
=======
		nodeConfig NodeConfig) (map[model.VerifierType]verifier.Verifier, error)
>>>>>>> main
}

type PublishersFactory interface {
	Get(ctx context.Context,
<<<<<<< HEAD
		nodeConfig NodeConfig,
		controller *controller.Controller) (map[model.Publisher]publisher.Publisher, error)
||||||| 5d1cca3e
		nodeConfig NodeConfig,
		controller *controller.Controller) (map[model.PublisherType]publisher.Publisher, error)
=======
		nodeConfig NodeConfig) (map[model.PublisherType]publisher.Publisher, error)
>>>>>>> main
}

// Functions that implement the factories for easier creation of new implementations
type StorageProvidersFactoryFunc func(
	ctx context.Context, nodeConfig NodeConfig) (map[model.StorageSourceType]storage.StorageProvider, error)

func (f StorageProvidersFactoryFunc) Get(
	ctx context.Context, nodeConfig NodeConfig) (map[model.StorageSourceType]storage.StorageProvider, error) {
	return f(ctx, nodeConfig)
}

type ExecutorsFactoryFunc func(ctx context.Context, nodeConfig NodeConfig) (map[model.Engine]executor.Executor, error)

func (f ExecutorsFactoryFunc) Get(ctx context.Context, nodeConfig NodeConfig) (map[model.Engine]executor.Executor, error) {
	return f(ctx, nodeConfig)
}

type VerifiersFactoryFunc func(
	ctx context.Context,
<<<<<<< HEAD
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.Verifier]verifier.Verifier, error)
||||||| 5d1cca3e
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.VerifierType]verifier.Verifier, error)
=======
	nodeConfig NodeConfig) (map[model.VerifierType]verifier.Verifier, error)
>>>>>>> main

func (f VerifiersFactoryFunc) Get(
	ctx context.Context,
<<<<<<< HEAD
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.Verifier]verifier.Verifier, error) {
	return f(ctx, nodeConfig, controller)
||||||| 5d1cca3e
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.VerifierType]verifier.Verifier, error) {
	return f(ctx, nodeConfig, controller)
=======
	nodeConfig NodeConfig) (map[model.VerifierType]verifier.Verifier, error) {
	return f(ctx, nodeConfig)
>>>>>>> main
}

type PublishersFactoryFunc func(
	ctx context.Context,
<<<<<<< HEAD
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.Publisher]publisher.Publisher, error)
||||||| 5d1cca3e
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.PublisherType]publisher.Publisher, error)
=======
	nodeConfig NodeConfig) (map[model.PublisherType]publisher.Publisher, error)
>>>>>>> main

func (f PublishersFactoryFunc) Get(
	ctx context.Context,
<<<<<<< HEAD
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.Publisher]publisher.Publisher, error) {
	return f(ctx, nodeConfig, controller)
||||||| 5d1cca3e
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.PublisherType]publisher.Publisher, error) {
	return f(ctx, nodeConfig, controller)
=======
	nodeConfig NodeConfig) (map[model.PublisherType]publisher.Publisher, error) {
	return f(ctx, nodeConfig)
>>>>>>> main
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
	nodeConfig NodeConfig) (map[model.Engine]executor.Executor, error) {
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
<<<<<<< HEAD
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.Verifier]verifier.Verifier, error) {
||||||| 5d1cca3e
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.VerifierType]verifier.Verifier, error) {
=======
	nodeConfig NodeConfig) (map[model.VerifierType]verifier.Verifier, error) {
>>>>>>> main
	return verifier_util.NewStandardVerifiers(
		ctx,
		nodeConfig.CleanupManager,
		localdb.GetStateResolver(nodeConfig.LocalDB),
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
<<<<<<< HEAD
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.Publisher]publisher.Publisher, error) {
||||||| 5d1cca3e
	nodeConfig NodeConfig,
	controller *controller.Controller) (map[model.PublisherType]publisher.Publisher, error) {
=======
	nodeConfig NodeConfig) (map[model.PublisherType]publisher.Publisher, error) {
>>>>>>> main
	return publisher_util.NewIPFSPublishers(
		ctx,
		nodeConfig.CleanupManager,
		localdb.GetStateResolver(nodeConfig.LocalDB),
		nodeConfig.IPFSClient.APIAddress(),
		nodeConfig.EstuaryAPIKey,
	)
}

func NewStandardPublishersFactory() *StandardPublishersFactory {
	return &StandardPublishersFactory{}
}
