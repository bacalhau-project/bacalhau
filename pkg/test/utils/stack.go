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
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func SetupTestWithDefaultConfigs(
	ctx context.Context,
	t *testing.T,
	nodes int, badActors int,
	lotusNode bool,
	nodeOverrides ...node.NodeConfig,
) (*devstack.DevStack, *system.CleanupManager) {
	return SetupTest(
		ctx,
		t,
		nodes, badActors,
		lotusNode,
		node.NewComputeConfigWithDefaults(),
		node.NewRequesterConfigWithDefaults(),
		nodeOverrides...,
	)
}

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

	// Wait for nodes to have announced their presence.
	for !allNodesDiscovered(t, stack) {
		time.Sleep(time.Second)
	}

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

		ids, err := stack.GetNodeIds()
		require.NoError(t, err)

		discoveredNodes, err := node.RequesterNode.NodeDiscoverer.ListNodes(ctx)
		require.NoError(t, err)

		for _, discoveredNode := range discoveredNodes {
			if discoveredNode.NodeType == model.NodeTypeCompute && discoveredNode.ComputeNodeInfo == nil {
				t.Logf("Node %s seen but without required compute node info", discoveredNode.PeerInfo.ID)
				return false
			}

			idx := slices.Index(ids, discoveredNode.PeerInfo.ID.String())
			require.GreaterOrEqualf(t, idx, 0, "Discovered a node not in the devstack?")

			ids = slices.Delete(ids, idx, idx+1)
		}

		if len(ids) > 0 {
			t.Logf("Did not see nodes %v", ids)
			return false
		}
	}

	return true
}
