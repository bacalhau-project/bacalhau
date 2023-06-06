//go:build integration || !unit

package simulator

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type SimulatorSuite struct {
	scenario.ScenarioRunner
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSimulatorSuite(t *testing.T) {
	suite.Run(t, new(SimulatorSuite))
}

func (suite *SimulatorSuite) TestSimulatorSanity() {
	system.InitConfigForTesting(suite.T())

	nodeCount := 3
	s := scenario.Scenario{
		JobCheckers: scenario.WaitUntilSuccessful(3),
		Spec:        scenario.WasmHelloWorld.Spec,
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
