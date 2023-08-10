package publicapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"

	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	requester_publicapi "github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

//nolint:unused // used in tests
func setupNodeForTest(t *testing.T) (*node.Node, *requester_publicapi.RequesterAPIClient) {
	// blank config should result in using defaults in node.Node constructor
	return setupNodeForTestWithConfig(t, publicapi.APIServerConfig{})
}

//nolint:unused // used in tests
func setupNodeForTestWithConfig(t *testing.T, config publicapi.APIServerConfig) (*node.Node, *requester_publicapi.RequesterAPIClient) {
	system.InitConfigForTesting(t)
	ctx := context.Background()

	cm := system.NewCleanupManager()
	t.Cleanup(func() { cm.Cleanup(context.Background()) })

	dir, _ := os.MkdirTemp("", "bacalhau-jobstore-test")
	dbFile := filepath.Join(dir, "testing.db")
	cm.RegisterCallback(func() error {
		os.Remove(dbFile)
		return nil
	})

	libp2pPort, err := freeport.GetFreePort()
	require.NoError(t, err)

	// TODO(forrest) [config] generate a key for testing
	libp2pHost, err := libp2p.NewHost(libp2pPort)
	require.NoError(t, err)

	nodeConfig := node.NodeConfig{
		CleanupManager:            system.NewCleanupManager(),
		Host:                      libp2pHost,
		HostAddress:               "0.0.0.0",
		APIPort:                   0,
		ComputeConfig:             node.NewComputeConfigWithDefaults(),
		RequesterNodeConfig:       node.NewRequesterConfigWithDefaults(),
		APIServerConfig:           config,
		IsRequesterNode:           true,
		IsComputeNode:             true,
		DependencyInjector:        devstack.NewNoopNodeDependencyInjector(),
		NodeInfoPublisherInterval: node.TestNodeInfoPublishConfig,
	}

	n, err := node.NewNode(ctx, nodeConfig)
	require.NoError(t, err)

	err = n.Start(ctx)
	require.NoError(t, err)

	client := requester_publicapi.NewRequesterAPIClient(n.APIServer.Address, n.APIServer.Port)
	require.NoError(t, requester_publicapi.WaitForHealthy(ctx, client))
	return n, client
}
