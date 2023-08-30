package publicapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

const TimeToWaitForServerReply = 10
const TimeToWaitForHealthy = 50

func setupNodeForTest(t *testing.T, cm *system.CleanupManager) *APIClient {
	// blank config should result in using defaults in node.Node constructor
	return setupNodeForTestWithConfig(t, cm, APIServerConfig{})
}

func setupNodeForTestWithConfig(t *testing.T, cm *system.CleanupManager, serverConfig APIServerConfig) *APIClient {
	setup.SetupBacalhauRepoForTesting(t)
	ctx := context.Background()

	libp2pPort, err := freeport.GetFreePort()
	require.NoError(t, err)

	// TODO(forrest) [config] generate a key or get it from testing repo
	libp2pHost, err := libp2p.NewHost(libp2pPort)
	require.NoError(t, err)

	apiServer, err := NewAPIServer(APIServerParams{
		Host:    libp2pHost,
		Address: "0.0.0.0",
		Port:    0,
		Config:  serverConfig,
	})
	require.NoError(t, err)

	require.NoError(t, apiServer.ListenAndServe(ctx, cm))

	client := NewAPIClient(apiServer.Address, apiServer.Port)
	require.NoError(t, waitForHealthy(ctx, client))
	return client
}

func waitForHealthy(ctx context.Context, c *APIClient) error {
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
