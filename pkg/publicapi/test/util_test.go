package test

import (
	"context"
	"net/http"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/go-chi/chi/v5"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
)

func setupServer(t *testing.T) (*publicapi.Server, *client.APIClient) {
	// blank config should result in using defaults in node.Node constructor
	return setupServerWithConfig(t, publicapi.NewConfig())
}

func setupServerWithConfig(t *testing.T, serverConfig *publicapi.Config) (*publicapi.Server, *client.APIClient) {
	return setupServerWithHandlers(t, serverConfig, map[string]http.Handler{})
}

func setupServerWithHandlers(
	t *testing.T, serverConfig *publicapi.Config, handlers map[string]http.Handler) (*publicapi.Server, *client.APIClient) {
	setup.SetupBacalhauRepoForTesting(t)
	ctx := context.Background()

	apiServer, err := publicapi.NewAPIServer(publicapi.ServerParams{
		Router:  chi.NewRouter(),
		Address: "0.0.0.0",
		Port:    0,
		Config:  *serverConfig,
	})
	require.NoError(t, err)

	// Register core endpoints
	shared.NewEndpoint(shared.EndpointParams{
		Router: apiServer.Router,
		NodeID: "test-node-id",
	})

	for path, handler := range handlers {
		apiServer.Router.Mount(path, handler)
	}
	require.NoError(t, apiServer.ListenAndServe(ctx))

	apiClient := client.NewAPIClient(apiServer.Address, apiServer.Port)
	require.NoError(t, WaitForAlive(ctx, apiClient))
	return apiServer, apiClient
}

func setupNodeForTest(t *testing.T) (*node.Node, *client.APIClient) {
	// blank config should result in using defaults in node.Node constructor
	return setupNodeForTestWithConfig(t, publicapi.Config{})
}

func setupNodeForTestWithConfig(t *testing.T, apiCfg publicapi.Config) (*node.Node, *client.APIClient) {
	fsRepo := setup.SetupBacalhauRepoForTesting(t)
	ctx := context.Background()

	cm := system.NewCleanupManager()
	t.Cleanup(func() { cm.Cleanup(context.Background()) })

	libp2pPort, err := freeport.GetFreePort()
	require.NoError(t, err)

	privKey, err := config.GetLibp2pPrivKey()
	require.NoError(t, err)
	libp2pHost, err := libp2p.NewHost(libp2pPort, privKey)
	require.NoError(t, err)

	nodeConfig := node.NodeConfig{
		CleanupManager:            cm,
		Host:                      libp2pHost,
		HostAddress:               "0.0.0.0",
		APIPort:                   0,
		ComputeConfig:             node.NewComputeConfigWithDefaults(),
		RequesterNodeConfig:       node.NewRequesterConfigWithDefaults(),
		APIServerConfig:           apiCfg,
		IsRequesterNode:           true,
		IsComputeNode:             true,
		DependencyInjector:        devstack.NewNoopNodeDependencyInjector(),
		NodeInfoPublisherInterval: node.TestNodeInfoPublishConfig,
		FsRepo:                    fsRepo,
	}

	n, err := node.NewNode(ctx, nodeConfig)
	require.NoError(t, err)

	err = n.Start(ctx)
	require.NoError(t, err)

	apiClient := client.NewAPIClient(n.APIServer.Address, n.APIServer.Port)
	require.NoError(t, WaitForNodes(ctx, apiClient))
	return n, apiClient
}
