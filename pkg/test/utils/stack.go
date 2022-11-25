// Package testutils collects common test utilities.
// Functions here create test stacks meant for integration tests
package testutils

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/stretchr/testify/require"
)

func SetupTest(
	ctx context.Context,
	t *testing.T,
	nodes int, badActors int,
	lotusNode bool,
	computeConfig node.ComputeConfig, //nolint:gocritic
	requesterNodeConfig requesternode.RequesterNodeConfig,
) (*devstack.DevStack, *system.CleanupManager) {
	require.NoError(t, system.InitConfigForTesting(t))

	cm := system.NewCleanupManager()
	t.Cleanup(cm.Cleanup)

	options := devstack.DevStackOptions{
		NumberOfNodes:     nodes,
		NumberOfBadActors: badActors,
		LocalNetworkLotus: lotusNode,
	}

	stack, err := devstack.NewStandardDevStack(ctx, cm, options, computeConfig, requesterNodeConfig)
	require.NoError(t, err)

	return stack, cm
}

type mixedExecutorFactory struct {
	*node.StandardExecutorsFactory
	*devstack.NoopExecutorsFactory
}

// Get implements node.ExecutorsFactory
func (m *mixedExecutorFactory) Get(ctx context.Context, nodeConfig node.NodeConfig) (executor.ExecutorProvider, error) {
	stdProvider, err := m.StandardExecutorsFactory.Get(ctx, nodeConfig)
	if err != nil {
		return nil, err
	}

	noopProvider, err := m.NoopExecutorsFactory.Get(ctx, nodeConfig)
	if err != nil {
		return nil, err
	}

	noopExecutor, err := noopProvider.GetExecutor(ctx, model.EngineNoop)
	if err != nil {
		return nil, err
	}

	err = stdProvider.AddExecutor(ctx, model.EngineNoop, noopExecutor)
	return stdProvider, err
}

var _ node.ExecutorsFactory = (*mixedExecutorFactory)(nil)

func SetupTestWithNoopExecutor(
	ctx context.Context,
	t *testing.T,
	options devstack.DevStackOptions,
	computeConfig node.ComputeConfig, //nolint:gocritic
	requesterNodeConfig requesternode.RequesterNodeConfig,
	executorConfig *noop_executor.ExecutorConfig,
) *devstack.DevStack {
	require.NoError(t, system.InitConfigForTesting(t))

	var executorFactory node.ExecutorsFactory
	if executorConfig != nil {
		// We will take the standard executors and add in the noop executor
		executorFactory = &mixedExecutorFactory{
			StandardExecutorsFactory: node.NewStandardExecutorsFactory(),
			NoopExecutorsFactory:     devstack.NewNoopExecutorsFactoryWithConfig(*executorConfig),
		}
	} else {
		executorFactory = node.NewStandardExecutorsFactory()
	}

	injector := node.NodeDependencyInjector{
		StorageProvidersFactory: node.NewStandardStorageProvidersFactory(),
		ExecutorsFactory:        executorFactory,
		VerifiersFactory:        node.NewStandardVerifiersFactory(),
		PublishersFactory:       node.NewStandardPublishersFactory(),
	}

	cm := system.NewCleanupManager()
	t.Cleanup(cm.Cleanup)

	stack, err := devstack.NewDevStack(ctx, cm, options, computeConfig, requesterNodeConfig, injector)
	require.NoError(t, err)

	return stack
}

// same as n
func NewNoopStackMultinode(
	ctx context.Context,
	t *testing.T,
	count int,
	computeConfig node.ComputeConfig,
	noopExecutorConfig noop_executor.ExecutorConfig,
	inprocessTransportConfig inprocess.InProcessTransportClusterConfig,
) ([]*node.Node, *system.CleanupManager) {
	cm := system.NewCleanupManager()

	nodes := []*node.Node{}

	inprocessTransportConfig.Count = count
	cluster, err := inprocess.NewInProcessTransportCluster(inprocessTransportConfig)
	require.NoError(t, err)

	for i := 0; i < count; i++ {
		datastore, err := inmemory.NewInMemoryDatastore()
		require.NoError(t, err)

		transport := cluster.GetTransport(i)
		nodeConfig := node.NodeConfig{
			CleanupManager:      cm,
			LocalDB:             datastore,
			Transport:           transport,
			ComputeConfig:       computeConfig,
			RequesterNodeConfig: requesternode.NewDefaultRequesterNodeConfig(),
		}

		injector := devstack.NewNoopNodeDependencyInjector()
		injector.ExecutorsFactory = devstack.NewNoopExecutorsFactoryWithConfig(noopExecutorConfig)
		node, err := node.NewNode(ctx, nodeConfig, injector)
		require.NoError(t, err)
		err = transport.Start(ctx)
		require.NoError(t, err)

		nodes = append(nodes, node)
	}

	return nodes, cm
}
