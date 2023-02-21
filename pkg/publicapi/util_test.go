package publicapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
)

const TimeToWaitForServerReply = 10
const TimeToWaitForHealthy = 50

//nolint:unused // used in tests
func setupNodeForTest(t *testing.T, cm *system.CleanupManager) *APIClient {
	// blank config should result in using defaults in node.Node constructor
	return setupNodeForTestWithConfig(t, cm, APIServerConfig{})
}

//nolint:unused // used in tests
func setupNodeForTestWithConfig(t *testing.T, cm *system.CleanupManager, serverConfig APIServerConfig) *APIClient {
	require.NoError(t, system.InitConfigForTesting(t))
	ctx := context.Background()

	libp2pPort, err := freeport.GetFreePort()
	require.NoError(t, err)

	apiPort, err := freeport.GetFreePort()
	require.NoError(t, err)

	libp2pHost, err := libp2p.NewHost(libp2pPort)
	require.NoError(t, err)

	apiServer, err := NewAPIServer(APIServerParams{
		Host:    libp2pHost,
		Address: "0.0.0.0",
		Port:    apiPort,
		Config:  serverConfig,
	})
	require.NoError(t, err)

	go func() {
		err := apiServer.ListenAndServe(ctx, cm)
		require.NoError(t, err)
	}()

	client := NewAPIClient(apiServer.GetURI())
	require.NoError(t, waitForHealthy(ctx, client))
	return client
}

//nolint:unused // used in tests
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
