//go:build integration

package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/suite"
)

type DevstackErrorLogsSuite struct {
	scenario.ScenarioRunner
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDevstackErrorLogsSuite(t *testing.T) {
	suite.Run(t, new(DevstackErrorLogsSuite))
}

var errorLogsTestCase = scenario.Scenario{
	ResultsChecker: scenario.ManyChecks(
		scenario.FileEquals(ipfs.DownloadFilenameStdout, "apples\n"),
		scenario.FileEquals(ipfs.DownloadFilenameStderr, "oranges\n"),
	),
	JobCheckers: []job.CheckStatesFunction{
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: 1,
		}),
	},
	Spec: model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash",
				"-c",
				"echo apples && echo oranges >&2 && exit 19",
			},
		},
	},
}

func (suite *DevstackErrorLogsSuite) TestErrorContainer() {
	testutils.MustHaveDocker(suite.T())
	suite.RunScenario(errorLogsTestCase)
}
