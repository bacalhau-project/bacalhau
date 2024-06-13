//go:build unit || !integration

package test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/router"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
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
	fsr, c := setup.SetupBacalhauRepoForTesting(t)
	stack := teststack.Setup(context.TODO(), t, fsr, c,
		devstack.WithNumberOfHybridNodes(1),
		devstack.WithNumberOfComputeOnlyNodes(2),
		devstack.WithNodeOverrides(nodeOverrides...),
	)
	// for the requester node to pick up the nodeInfo messages
	nodeutils.WaitForNodeDiscovery(t, stack.Nodes[0].RequesterNode, nodeCount)

	c.Node.ClientAPI.Host = stack.Nodes[0].APIServer.Address
	c.Node.ClientAPI.Port = int(stack.Nodes[0].APIServer.Port)
	t.Log("Host set to", c.Node.ClientAPI.Host)
	t.Log("Port set to", c.Node.ClientAPI.Port)

	os.Setenv("BACALHAU_NODE_SELECTORS", "owner=bacalhau")

	for name := range router.TestcasesMap {
		t.Run(name, func(t *testing.T) {
			event := models.Event{Action: name}
			err := router.RouteWithOpts(context.Background(), event, router.WithConfig(c))
			require.NoError(t, err)
		})
	}
}
