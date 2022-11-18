//go:build integration

package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"

	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/suite"
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

	testCase := scenario.WasmHelloWorld
	testCase.Stack = &scenario.StackConfig{
		DevStackOptions: &devstack.DevStackOptions{NumberOfNodes: 3},
	}
	testCase.Deal = model.Deal{Concurrency: 2}
	testCase.ResultsChecker = scenario.FileEquals(
		ipfs.DownloadFilenameStdout,
		"Hello, world!\nHello, world!\n",
	)
	testCase.JobCheckers = []job.CheckStatesFunction{
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: 2,
		}),
	}

	suite.RunScenario(testCase)
}
