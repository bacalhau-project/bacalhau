package publicapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
)

const TimeToWaitForServerReply = 10
const TimeToWaitForHealthy = 50

//nolint:unused // used in tests
func setupNodeForTest(t *testing.T) (*node.Node, *publicapi.APIClient) {
	// blank config should result in using defaults in node.Node constructor
	return setupNodeForTestWithConfig(t, publicapi.APIServerConfig{})
}

//nolint:unused // used in tests
func setupNodeForTestWithConfig(t *testing.T, serverConfig publicapi.APIServerConfig) (*node.Node, *publicapi.APIClient) {
	require.NoError(t, system.InitConfigForTesting(t))
	ctx := context.Background()

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	libp2pPort, err := freeport.GetFreePort()
	require.NoError(t, err)

	apiPort, err := freeport.GetFreePort()
	require.NoError(t, err)

	libp2pHost, err := libp2p.NewHost(libp2pPort)
	require.NoError(t, err)

	nodeConfig := node.NodeConfig{
		CleanupManager:      system.NewCleanupManager(),
		Host:                libp2pHost,
		HostAddress:         "0.0.0.0",
		APIPort:             apiPort,
		LocalDB:             datastore,
		ComputeConfig:       node.NewComputeConfigWithDefaults(),
		RequesterNodeConfig: node.NewRequesterConfigWithDefaults(),
		APIServerConfig:     serverConfig,
	}

	n, err := node.NewNode(ctx, nodeConfig, devstack.NewNoopNodeDependencyInjector())
	require.NoError(t, err)

	err = n.Start(ctx)
	require.NoError(t, err)

	client := publicapi.NewAPIClient(n.APIServer.GetURI())
	require.NoError(t, waitForHealthy(ctx, client))
	return n, client
}

//nolint:unused // used in tests
func waitForHealthy(ctx context.Context, c *publicapi.APIClient) error {
	ch := make(chan bool)
	go func() {
		for {
			alive, err := c.Alive(ctx)
			if err == nil && alive {
				ch <- true
				return
			}

			time.Sleep(time.Duration(TimeToWaitForHealthy) * time.Millisecond)
		}
	}()

	select {
	case <-ch:
		return nil
	case <-time.After(time.Duration(TimeToWaitForServerReply) * time.Second):
		return fmt.Errorf("server did not reply after %ss", time.Duration(TimeToWaitForServerReply)*time.Second)
	}
}
