//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type DevstackConcurrencySuite struct {
	scenario.ScenarioRunner
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDevstackConcurrencySuite(t *testing.T) {
	suite.Run(t, new(DevstackConcurrencySuite))
}

func (suite *DevstackConcurrencySuite) TestConcurrencyLimit() {

	testCase := scenario.WasmHelloWorld(suite.T())
	testCase.Stack = &scenario.StackConfig{
		DevStackOptions: &devstack.DevStackOptions{NumberOfHybridNodes: 3},
	}
	testCase.Deal = model.Deal{Concurrency: 2}
	testCase.ResultsChecker = scenario.FileEquals(
		model.DownloadFilenameStdout,
		"Hello, world!\n",
	)
	testCase.JobCheckers = []job.CheckStatesFunction{
		job.WaitForExecutionStates(map[model.ExecutionStateType]int{
			model.ExecutionStateCompleted: testCase.Deal.Concurrency,
		}),
	}

	suite.RunScenario(testCase)
}
