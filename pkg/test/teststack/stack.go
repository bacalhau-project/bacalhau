// Package testutils collects common test utilities.
// Functions here create test stacks meant for integration tests
package teststack

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/system/cleanup"
)

func testDevStackConfig() *devstack.DevStackOptions {
	return &devstack.DevStackOptions{
		NumberOfHybridNodes:        0,
		NumberOfRequesterOnlyNodes: 0,
		NumberOfComputeOnlyNodes:   0,
		NumberOfBadComputeActors:   0,
		NumberOfBadRequesterActors: 0,
		Peer:                       "",
		PublicIPFSMode:             false,
		EstuaryAPIKey:              "",
		CPUProfilingFile:           "",
		MemoryProfilingFile:        "",
		DisabledFeatures:           node.FeatureConfig{},
		AllowListedLocalPaths:      nil,
		NodeInfoPublisherInterval:  routing.NodeInfoPublisherIntervalConfig{},
		ExecutorPlugins:            false,
	}
}

func Setup(
	ctx context.Context,
	t testing.TB,
	opts ...devstack.ConfigOption,
) *devstack.DevStack {
	fsRepo := system.SetupBacalhauRepoForTesting(t)
	cm := cleanup.NewCleanupManager()
	t.Cleanup(func() {
		cm.Cleanup(ctx)
	})
	stack, err := devstack.Setup(ctx, cm, fsRepo, append(testDevStackConfig().Options(), opts...)...)
	if err != nil {
		t.Fatalf("creating teststack: %s", err)
	}

	// Wait for nodes to have announced their presence.
	//nolint:gomnd
	assert.Eventually(t, func() bool {
		return allNodesDiscovered(t, stack)
	}, 10*time.Second, 100*time.Millisecond, "failed to discover all nodes")

	return stack
}

func WithNoopExecutor(noopConfig noop_executor.ExecutorConfig) devstack.ConfigOption {
	return devstack.WithDependencyInjector(node.NodeDependencyInjector{
		ExecutorsFactory: &mixedExecutorFactory{
			standardFactory: node.NewStandardExecutorsFactory(),
			noopFactory:     devstack.NewNoopExecutorsFactoryWithConfig(noopConfig),
		},
	})
}

func allNodesDiscovered(t testing.TB, stack *devstack.DevStack) bool {
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

type mixedExecutorFactory struct {
	standardFactory, noopFactory node.ExecutorsFactory
}

// Get implements node.ExecutorsFactory
func (m *mixedExecutorFactory) Get(
	ctx context.Context,
	nodeConfig node.NodeConfig,
) (executor.ExecutorProvider, error) {
	stdProvider, err := m.standardFactory.Get(ctx, nodeConfig)
	if err != nil {
		return nil, err
	}

	noopProvider, err := m.noopFactory.Get(ctx, nodeConfig)
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

// Returns whether the requester node(s) in the stack have discovered all of the
// other nodes in the stack and have complete information for them (i.e. each
// node has actually announced itself.)
