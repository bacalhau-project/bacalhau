// Package testutils collects common test utilities.
// Functions here create test stacks meant for integration tests
package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func SetupTestWithDefaultConfigs(
	ctx context.Context,
	t *testing.T,
	nodes int,
	nodeOverrides ...node.NodeConfig,
) (*devstack.DevStack, *system.CleanupManager) {
	return SetupTest(
		ctx,
		t,
		nodes,
		node.NewComputeConfigWithDefaults(),
		node.NewRequesterConfigWithDefaults(),
		nodeOverrides...,
	)
}

func SetupTest(
	ctx context.Context,
	t *testing.T,
	nodes int,
	computeConfig node.ComputeConfig, //nolint:gocritic
	requesterConfig node.RequesterConfig,
	nodeOverrides ...node.NodeConfig,
) (*devstack.DevStack, *system.CleanupManager) {
	cm := system.NewCleanupManager()
	t.Cleanup(func() {
		cm.Cleanup(ctx)
	})

	options := devstack.DevStackOptions{
		NumberOfHybridNodes: nodes,
	}
	stack := SetupTestWithNoopExecutor(ctx, t, options, computeConfig, requesterConfig, noop_executor.ExecutorConfig{}, nodeOverrides...)
	return stack, cm
}

type mixedExecutorFactory struct {
	standardFactory, noopFactory node.ExecutorsFactory
}

// Get implements node.ExecutorsFactory
func (m *mixedExecutorFactory) Get(
	ctx context.Context,
	nodeConfig node.NodeConfig,
	storages storage.StorageProvider,
) (executor.ExecutorProvider, error) {
	stdProvider, err := m.standardFactory.Get(ctx, nodeConfig, storages)
	if err != nil {
		return nil, err
	}

	noopProvider, err := m.noopFactory.Get(ctx, nodeConfig, storages)
	if err != nil {
		return nil, err
	}

	noopExecutor, err := noopProvider.Get(ctx, model.EngineNoop)
	if err != nil {
		return nil, err
	}

	return &model.ChainedProvider[model.Engine, executor.Executor]{
		Providers: []model.Provider[model.Engine, executor.Executor]{
			stdProvider,
			model.NewMappedProvider(map[model.Engine]executor.Executor{
				model.EngineNoop: noopExecutor,
			}),
		},
	}, nil
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
	system.InitConfigForTesting(t)
	// We will take the standard executors and add in the noop executor
	executorFactory := &mixedExecutorFactory{
		standardFactory: node.NewStandardExecutorsFactory(),
		noopFactory:     devstack.NewNoopExecutorsFactoryWithConfig(executorConfig),
	}

	injector := node.NodeDependencyInjector{
		StorageProvidersFactory: node.NewStandardStorageProvidersFactory(),
		ExecutorsFactory:        executorFactory,
		PublishersFactory:       node.NewStandardPublishersFactory(),
	}

	cm := system.NewCleanupManager()
	t.Cleanup(func() {
		cm.Cleanup(ctx)
	})

	stack, err := devstack.NewDevStack(ctx, cm, options, computeConfig, requesterConfig, injector, nodeOverrides...)
	require.NoError(t, err)

	// Wait for nodes to have announced their presence.
	assert.Eventually(t, func() bool {
		return allNodesDiscovered(t, stack)
	}, 10*time.Second, 100*time.Millisecond, "failed to discover all nodes")

	return stack
}

// Returns whether the requester node(s) in the stack have discovered all of the
// other nodes in the stack and have complete information for them (i.e. each
// node has actually announced itself.)
func allNodesDiscovered(t *testing.T, stack *devstack.DevStack) bool {
	for _, node := range stack.Nodes {
		ctx := logger.ContextWithNodeIDLogger(context.Background(), node.Host.ID().String())

		if !node.IsRequesterNode() || node.RequesterNode == nil {
			continue
		}

		expectedNodes := stack.GetNodeIds()
		discoveredNodes, err := node.RequesterNode.NodeDiscoverer.ListNodes(ctx)
		require.NoError(t, err)

		if len(discoveredNodes) < len(expectedNodes) {
			t.Logf("Only discovered %d nodes, expected %d. Retrying", len(discoveredNodes), len(expectedNodes))
			return false
		}

		discoveredNodeIDs := make([]string, len(discoveredNodes))
		for i, discoveredNode := range discoveredNodes {
			discoveredNodeIDs[i] = discoveredNode.PeerInfo.ID.String()
		}
		require.ElementsMatch(t, expectedNodes, discoveredNodeIDs)
	}

	return true
}
