//go:build unit || !integration

package test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/router"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
	nodeutils "github.com/bacalhau-project/bacalhau/pkg/test/utils/node"
)

func TestScenariosAgainstDevstack(t *testing.T) {
	nodeOverride := node.NodeConfig{
		Labels: map[string]string{
			"owner": "bacalhau",
		},
	}
	nodeCount := 3
	nodeOverrides := make([]node.NodeConfig, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodeOverrides[i] = nodeOverride
	}
	stack := teststack.Setup(context.TODO(), t,
		devstack.WithNumberOfHybridNodes(nodeCount),
		devstack.WithNodeOverrides(nodeOverrides...),
	)
	// for the requester node to pick up the nodeInfo messages
	nodeutils.WaitForNodeDiscovery(t, stack.Nodes[0], nodeCount)

	var swarmAddresses []string
	for _, n := range stack.Nodes {
		nodeSwarmAddresses, err := n.IPFSClient.SwarmAddresses(context.Background())
		require.NoError(t, err)
		swarmAddresses = append(swarmAddresses, nodeSwarmAddresses...)
	}
	// Need to set the swarm addresses for getIPFSDownloadSettings() to work in test
	swarmenv := config.KeyAsEnvVar(types.NodeIPFSSwarmAddresses)
	os.Setenv(swarmenv, strings.Join(swarmAddresses, ","))
	t.Logf("%s: %s", swarmenv, os.Getenv(swarmenv))

	// Add data to devstack IPFS
	testString := "This is a test string"
	cid, err := ipfs.AddTextToNodes(context.Background(), []byte(testString), stack.IPFSClients()...)
	require.NoError(t, err)
	// Need to set the local ipfs CID for SubmitDockerIPFSJobAndGet() to work in test
	os.Setenv("BACALHAU_CANARY_TEST_CID", cid)

	host := stack.Nodes[0].APIServer.Address
	port := stack.Nodes[0].APIServer.Port
	t.Log("Host set to", host)
	t.Log("Port set to", port)

	viper.Set(types.NodeClientAPIHost, host)
	viper.Set(types.NodeClientAPIPort, port)
	os.Setenv("BACALHAU_NODE_SELECTORS", "owner=bacalhau")

	for name := range router.TestcasesMap {
		t.Run(name, func(t *testing.T) {
			event := models.Event{Action: name}
			err := router.Route(context.Background(), event)
			require.NoError(t, err)
		})
	}
}
