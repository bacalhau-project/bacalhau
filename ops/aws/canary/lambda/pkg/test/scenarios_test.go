package test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/router"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/node"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
)

func TestScenarios(t *testing.T) {
	stack, _ := testutils.SetupTest(context.Background(), t, 3, 0, false, node.NewComputeConfigWithDefaults(), node.NewRequesterConfigWithDefaults())

	os.Setenv("BACALHAU_ENVIRONMENT", "test")
	t.Logf("BACALHAU_ENVIRONMENT: %s", os.Getenv("BACALHAU_ENVIRONMENT"))

	swarmAddresses, err := stack.Nodes[0].IPFSClient.SwarmAddresses(context.Background())
	require.NoError(t, err)
	// Need to set the swarm addresses for getIPFSDownloadSettings() to work in test
	os.Setenv("BACALHAU_IPFS_SWARM_ADDRESSES", strings.Join(swarmAddresses, ","))
	t.Logf("BACALHAU_IPFS_SWARM_ADDRESSES: %s", os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES"))

	// Add data to devstack IPFS
	testString := "This is a test string"
	cid, err := ipfs.AddTextToNodes(context.Background(), []byte(testString), stack.IPFSClients()...)
	require.NoError(t, err)
	// Need to set the local ipfs CID for SubmitDockerIPFSJobAndGet() to work in test
	os.Setenv("BACALHAU_CANARY_TEST_CID", cid)

	host := stack.Nodes[0].APIServer.Host
	port := stack.Nodes[0].APIServer.Port
	t.Log("Host set to", host)
	t.Log("Port set to", port)

	os.Setenv("BACALHAU_HOST", host)
	os.Setenv("BACALHAU_PORT", fmt.Sprint(port))

	for name := range router.TestcasesMap {
		t.Run(name, func(t *testing.T) {
			event := models.Event{Action: name}
			err := router.Route(context.Background(), event)
			require.NoError(t, err)
		})
	}
}
