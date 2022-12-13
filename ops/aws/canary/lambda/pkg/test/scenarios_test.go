package test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/filecoin-project/bacalhau/ops/aws/canary/pkg/router"
	"github.com/filecoin-project/bacalhau/pkg/node"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
)

func TestScenarios(t *testing.T) {
	stack, _ := testutils.SetupTest(context.Background(), t, 1, 0, false, node.NewComputeConfigWithDefaults(), node.NewRequesterConfigWithDefaults())

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
