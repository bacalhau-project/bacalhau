// Package testutils collects common test utilities.
// Functions here create test stacks meant for integration tests
package teststack

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/setup"

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
	_, useNATS := os.LookupEnv("BACALHAU_NODE_NETWORK_USENATS")
	return &devstack.DevStackOptions{
		NumberOfHybridNodes:        0,
		NumberOfRequesterOnlyNodes: 0,
		NumberOfComputeOnlyNodes:   0,
		NumberOfBadComputeActors:   0,
		NumberOfBadRequesterActors: 0,
		Peer:                       "",
		PublicIPFSMode:             false,
		CPUProfilingFile:           "",
		MemoryProfilingFile:        "",
		DisabledFeatures:           node.FeatureConfig{},
		AllowListedLocalPaths:      nil,
		NodeInfoPublisherInterval:  routing.NodeInfoPublisherIntervalConfig{},
		ExecutorPlugins:            false,
		UseNATS:                    useNATS,
	}
}

func Setup(
	ctx context.Context,
	t testing.TB,
	opts ...devstack.ConfigOption,
) *devstack.DevStack {
	// NB: if a test suite has defined a repo use it, otherwise make one.
	repoPath := os.Getenv("BACALHAU_DIR")
	var fsRepo *repo.FsRepo
	if repoPath != "" {
		var err error
		fsRepo, err = repo.NewFS(repoPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := fsRepo.Open(); err != nil {
			t.Fatal(err)
		}
	} else {
		fsRepo = setup.SetupBacalhauRepoForTesting(t)
	}
	cm := system.NewCleanupManager()
	t.Cleanup(func() {
		cm.Cleanup(ctx)
	})
	stack, err := devstack.Setup(ctx, cm, fsRepo, append(testDevStackConfig().Options(), opts...)...)
	if err != nil {
		t.Fatalf("creating teststack: %s", err)
	}

	// Wait for nodes to have announced their presence.
	//nolint:gomnd
	require.Eventually(t,
		func() bool {
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
		ctx := logger.ContextWithNodeIDLogger(context.Background(), node.ID)

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
			discoveredNodeIDs[i] = discoveredNode.ID()
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

	noopExecutor, err := noopProvider.Get(ctx, model.EngineNoop.String())
	if err != nil {
		return nil, err
	}

	return &provider.ChainedProvider[executor.Executor]{
		Providers: []provider.Provider[executor.Executor]{
			stdProvider,
			provider.NewMappedProvider(map[string]executor.Executor{
				models.EngineNoop: noopExecutor,
			}),
		},
	}, nil
}

var _ node.ExecutorsFactory = (*mixedExecutorFactory)(nil)

// Returns whether the requester node(s) in the stack have discovered all of the
// other nodes in the stack and have complete information for them (i.e. each
// node has actually announced itself.)
