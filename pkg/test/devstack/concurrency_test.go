//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
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
		DevStackOptions: &devstack.DevStackOptions{
			NumberOfHybridNodes:      1,
			NumberOfComputeOnlyNodes: 2,
		},
	}
	testCase.Job.Count = 2
	testCase.Job.Task().Publisher = publisher_local.NewSpecConfig()
	testCase.ResultsChecker = scenario.FileEquals(
		downloader.DownloadFilenameStdout,
		"Hello, world!\nHello, world!\n",
	)
	testCase.JobCheckers = []scenario.StateChecks{
		scenario.WaitForExecutionStates(map[models.ExecutionStateType]int{
			models.ExecutionStateCompleted: testCase.Job.Count,
		}),
	}

	suite.RunScenario(testCase)
}
