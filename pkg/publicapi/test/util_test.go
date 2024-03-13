package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func setupNodeForTest(t *testing.T) (*node.Node, client.API) {
	// blank config should result in using defaults in node.Node constructor
	return setupNodeForTestWithConfig(t, publicapi.Config{})
}

func setupNodeForTestWithConfig(t *testing.T, apiCfg publicapi.Config) (*node.Node, client.API) {
	repo := setup.SetupBacalhauRepoForTesting(t)
	ctx := context.Background()

	repoPath, err := repo.Path()
	require.NoError(t, err)

	cm := system.NewCleanupManager()
	t.Cleanup(func() { cm.Cleanup(context.Background()) })

	libp2pPort, err := network.GetFreePort()
	require.NoError(t, err)

	privKey, err := config.GetLibp2pPrivKey()
	require.NoError(t, err)

	peerID, err := peer.IDFromPrivateKey(privKey)
	require.NoError(t, err)
	nodeID := peerID.String()

	var libp2pHost host.Host
	networkType, ok := os.LookupEnv("BACALHAU_NODE_NETWORK_TYPE")
	if !ok {
		networkType = models.NetworkTypeNATS
	}

	if networkType == models.NetworkTypeLibp2p {
		libp2pHost, err = libp2p.NewHost(libp2pPort, privKey)
		require.NoError(t, err)
	}

	jobStore, err := boltjobstore.NewBoltJobStore(filepath.Join(repoPath, "jobs.db"))
	require.NoError(t, err)

	executionStore, err := boltdb.NewStore(ctx, filepath.Join(repoPath, "executions.db"))
	require.NoError(t, err)

	computeConfig, err := node.NewComputeConfigWith(node.ComputeConfigParams{
		ExecutionStore: executionStore,
	})
	require.NoError(t, err)
	requesterConfig, err := node.NewRequesterConfigWith(node.RequesterConfigParams{
		JobStore: jobStore,
	})
	require.NoError(t, err)

	nodeConfig := node.NodeConfig{
		NodeID:                    nodeID,
		CleanupManager:            cm,
		HostAddress:               "0.0.0.0",
		APIPort:                   0,
		ComputeConfig:             computeConfig,
		RequesterNodeConfig:       requesterConfig,
		APIServerConfig:           apiCfg,
		IsRequesterNode:           true,
		IsComputeNode:             true,
		DependencyInjector:        devstack.NewNoopNodeDependencyInjector(),
		NodeInfoPublisherInterval: node.TestNodeInfoPublishConfig,
		NodeInfoStoreTTL:          10 * time.Minute,
		NetworkConfig: node.NetworkConfig{
			Type:       networkType,
			Libp2pHost: libp2pHost,
		},
	}

	n, err := node.NewNode(ctx, nodeConfig)
	require.NoError(t, err)

	err = n.Start(ctx)
	require.NoError(t, err)

	apiClient := client.New(n.APIServer.GetURI().String())
	require.NoError(t, WaitForNodes(ctx, apiClient))
	return n, apiClient
}
