package simulator

import (
	"context"
	"fmt"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SimulatorSuite struct {
	scenario.ScenarioRunner
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSimulatorSuite(t *testing.T) {
	suite.Run(t, new(SimulatorSuite))
}

// Test that the combo driver gives preference to the filecoin unsealed driver
// also that this does not affect normal jobs where the CID resides on the IPFS driver
func (suite *SimulatorSuite) TestSimulatorSanity() {
	system.InitConfigForTesting(suite.T())
	ctx := context.Background()
	cm := system.NewCleanupManager()

	simulatorPort, err := freeport.GetFreePort()
	require.NoError(suite.T(), err)

	localDB, err := inmemory.NewInMemoryDatastore()
	require.NoError(suite.T(), err)

	simulatorServer := NewServer(ctx, "0.0.0.0", simulatorPort, localDB)
	go simulatorServer.ListenAndServe(ctx, cm)

	nodeCount := 3
	s := scenario.Scenario{
		Contexts: scenario.StoredFile(
			"../../testdata/wasm/noop/main.wasm",
			"/job",
		),
		JobCheckers: scenario.WaitUntilSuccessful(3),
		Spec: model.Spec{
			Engine: model.EngineWasm,
			Wasm: model.JobSpecWasm{
				EntryPoint: "_start",
				Parameters: []string{},
			},
		},
		Deal: model.Deal{
			Concurrency: 3,
		},
	}

	s.Stack = &scenario.StackConfig{
		DevStackOptions: &devstack.DevStackOptions{
			NumberOfNodes: nodeCount,
			SimulatorURL:  fmt.Sprintf("ws://127.0.0.1:%d/websocket", simulatorPort),
		},
	}

	suite.RunScenario(s)
}
