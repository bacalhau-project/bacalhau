// Package testutils collects common test utilities.
// Functions here create test stacks meant for integration tests
package testutils

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
)

func SetupTest(
	ctx context.Context,
	t *testing.T,
	nodes int, badActors int,
	lotusNode bool,
	computeConfig node.ComputeConfig, //nolint:gocritic
	requesterConfig node.RequesterConfig,
	nodeOverrides ...node.NodeConfig,
) (*devstack.DevStack, *system.CleanupManager) {
	cm := system.NewCleanupManager()
	t.Cleanup(func() {
		cm.Cleanup(ctx)
	})

	options := devstack.DevStackOptions{
		NumberOfHybridNodes:      nodes,
		NumberOfBadComputeActors: badActors,
		LocalNetworkLotus:        lotusNode,
	}
	stack := SetupTestWithNoopExecutor(ctx, t, options, computeConfig, requesterConfig, noop_executor.ExecutorConfig{}, nodeOverrides...)
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

	noopExecutor, err := noopProvider.Get(ctx, model.EngineNoop)
	if err != nil {
		return nil, err
	}

	stdProvider.(*model.MappedProvider[model.Engine, executor.Executor]).Add(model.EngineNoop, noopExecutor)
	return stdProvider, nil
}

var _ node.ExecutorsFactory = (*mixedExecutorFactory)(nil)

func SetupTestWithNoopExecutor(
	ctx context.Context,
	t *testing.T,
	options devstack.DevStackOptions,
	computeConfig node.ComputeConfig, //nolint:gocritic
	requesterConfig node.RequesterConfig,
	executorConfig noop_executor.ExecutorConfig,
	nodeOverrides ...node.NodeConfig,
) *devstack.DevStack {
	require.NoError(t, system.InitConfigForTesting(t))
	// We will take the standard executors and add in the noop executor
	executorFactory := &mixedExecutorFactory{
		StandardExecutorsFactory: node.NewStandardExecutorsFactory(),
		NoopExecutorsFactory:     devstack.NewNoopExecutorsFactoryWithConfig(executorConfig),
	}

	injector := node.NodeDependencyInjector{
		StorageProvidersFactory: node.NewStandardStorageProvidersFactory(),
		ExecutorsFactory:        executorFactory,
		VerifiersFactory:        node.NewStandardVerifiersFactory(),
		PublishersFactory:       node.NewStandardPublishersFactory(),
	}

	cm := system.NewCleanupManager()
	t.Cleanup(func() {
		cm.Cleanup(ctx)
	})

	stack, err := devstack.NewDevStack(ctx, cm, options, computeConfig, requesterConfig, injector, nodeOverrides...)
	require.NoError(t, err)

	return stack
}
