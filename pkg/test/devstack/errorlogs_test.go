//go:build integration || !unit

package devstack

import (
	"context"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
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
	Stack: &scenario.StackConfig{
		ExecutorConfig: &noop.ExecutorConfig{
			ExternalHooks: noop.ExecutorConfigExternalHooks{
				JobHandler: func(ctx context.Context, shard model.JobShard, resultsDir string) (*model.RunCommandResult, error) {
					return executor.WriteJobResults(resultsDir, strings.NewReader("apples"), strings.NewReader("oranges"), 19, nil)
				},
			},
		},
	},
	ResultsChecker: scenario.ManyChecks(
		scenario.FileEquals(ipfs.DownloadFilenameStdout, "apples"),
		scenario.FileEquals(ipfs.DownloadFilenameStderr, "oranges"),
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
		Engine:    model.EngineNoop,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
	},
}

func (suite *DevstackErrorLogsSuite) TestErrorContainer() {
	suite.RunScenario(errorLogsTestCase)
}
