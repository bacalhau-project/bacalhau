// Package testutils collects common test utilities.
// Functions here create test stacks meant for integration tests
package testutils

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

func SetupTestDevStack(
	ctx context.Context,
	t testing.TB,
	opts ...devstack.ConfigOption,
) *devstack.DevStack {
	system.InitConfigForTesting(t)
	cm := system.NewCleanupManager()
	t.Cleanup(func() {
		cm.Cleanup(ctx)
	})
	stack, err := devstack.NewDevStack(ctx, cm, append(testDevStackConfig().Options(), opts...)...)
	if err != nil {
		t.Fatalf("creating devstack: %s", err)
	}

	// Wait for nodes to have announced their presence.
	//nolint:gomnd
	assert.Eventually(t, func() bool {
		return allNodesDiscovered(t, stack)
	}, 10*time.Second, 100*time.Millisecond, "failed to discover all nodes")

	return stack
}

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

type MixedExecutorFactory struct {
	StandardFactory, NoopFactory node.ExecutorsFactory
}

// Get implements node.ExecutorsFactory
func (m *MixedExecutorFactory) Get(
	ctx context.Context,
	nodeConfig node.NodeConfig,
) (executor.ExecutorProvider, error) {
	stdProvider, err := m.StandardFactory.Get(ctx, nodeConfig)
	if err != nil {
		return nil, err
	}

	noopProvider, err := m.NoopFactory.Get(ctx, nodeConfig)
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

var _ node.ExecutorsFactory = (*MixedExecutorFactory)(nil)

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
	executorFactory := &MixedExecutorFactory{
		StandardFactory: node.NewStandardExecutorsFactory(),
		NoopFactory:     devstack.NewNoopExecutorsFactoryWithConfig(executorConfig),
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

	stack, err := devstack.NewDevStack(ctx, cm,
		append(
			options.Options(),
			devstack.WithComputeConfig(computeConfig),
			devstack.WithRequesterConfig(requesterConfig),
			devstack.WithDependencyInjector(injector),
			devstack.WithNodeOverrides(nodeOverrides...),
		)...)
	require.NoError(t, err)

	// Wait for nodes to have announced their presence.
	//nolint:gomnd
	assert.Eventually(t, func() bool {
		return allNodesDiscovered(t, stack)
	}, 10*time.Second, 100*time.Millisecond, "failed to discover all nodes")

	return stack
}

// Returns whether the requester node(s) in the stack have discovered all of the
// other nodes in the stack and have complete information for them (i.e. each
// node has actually announced itself.)
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
