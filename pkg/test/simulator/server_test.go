package simulator

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
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

	nodeCount := 3
	s := scenario.Scenario{
		JobCheckers: scenario.WaitUntilSuccessful(3),
		Spec: model.Spec{
			Engine: model.EngineWasm,
			Wasm: model.JobSpecWasm{
				EntryPoint:  scenario.WasmHelloWorld.Spec.Wasm.EntryPoint,
				EntryModule: scenario.WasmHelloWorld.Spec.Wasm.EntryModule,
				Parameters:  []string{},
			},
		},
		Deal: model.Deal{
			Concurrency: 3,
		},
	}

	s.Stack = &scenario.StackConfig{
		DevStackOptions: &devstack.DevStackOptions{
			NumberOfHybridNodes: nodeCount,
			SimulatorMode:       true,
		},
	}

	suite.RunScenario(s)
}
