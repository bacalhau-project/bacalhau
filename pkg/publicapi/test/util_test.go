package test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func setupNodeForTest(t *testing.T) (*node.Node, types.BacalhauConfig) {
	// blank config should result in using defaults in node.Node constructor
	return setupNodeForTestWithConfig(t, publicapi.Config{})
}

func setupNodeForTestWithConfig(t *testing.T, apiCfg publicapi.Config) (*node.Node, types.BacalhauConfig) {
	repo, c := setup.SetupBacalhauRepoForTesting(t)
	ctx := context.Background()

	repoPath, err := repo.Path()
	require.NoError(t, err)

	cm := system.NewCleanupManager()
	t.Cleanup(func() { cm.Cleanup(context.Background()) })

	networkPort, err := network.GetFreePort()
	require.NoError(t, err)

	networkConfig := node.NetworkConfig{
		AuthSecret: "test",
		Port:       networkPort,
	}

	jobStore, err := boltjobstore.NewBoltJobStore(filepath.Join(repoPath, "jobs.db"))
	require.NoError(t, err)

	executionStore, err := boltdb.NewStore(ctx, filepath.Join(repoPath, "executions.db"))
	require.NoError(t, err)

	executionDir, err := c.ExecutionDir()
	require.NoError(t, err)

	computeConfig, err := node.NewComputeConfigWith(executionDir, node.ComputeConfigParams{
		ExecutionStore: executionStore,
	})
	require.NoError(t, err)
	requesterConfig, err := node.NewRequesterConfigWith(node.RequesterConfigParams{
		JobStore: jobStore,
	})
	require.NoError(t, err)

	nodeConfig := node.NodeConfig{
		NodeID:              "node-0",
		CleanupManager:      cm,
		HostAddress:         "0.0.0.0",
		APIPort:             0,
		ComputeConfig:       computeConfig,
		RequesterNodeConfig: requesterConfig,
		APIServerConfig:     apiCfg,
		IsRequesterNode:     true,
		IsComputeNode:       true,
		DependencyInjector:  devstack.NewNoopNodeDependencyInjector(),
		NetworkConfig:       networkConfig,
	}

	n, err := node.NewNode(ctx, c, nodeConfig, repo)
	require.NoError(t, err)

	err = n.Start(ctx)
	require.NoError(t, err)

	apiClient := client.New(n.APIServer.GetURI().String())
	require.NoError(t, WaitForNodes(ctx, apiClient))
	return n, c
}
