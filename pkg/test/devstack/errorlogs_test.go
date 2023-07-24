//go:build integration || !unit

package devstack

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type DevstackErrorLogsSuite struct {
	scenario.ScenarioRunner
}

func TestDevstackErrorLogsSuite(t *testing.T) {
	suite.Run(t, new(DevstackErrorLogsSuite))
}

var executorTestCases = []model.Spec{
	{
		Engine: model.EngineNoop,
		PublisherSpec: model.PublisherSpec{
			Type: model.PublisherIpfs,
		},
	},
	{
		Engine: model.EngineDocker,
		PublisherSpec: model.PublisherSpec{
			Type: model.PublisherIpfs,
		},
		Docker: model.JobSpecDocker{
			Image:      "ubuntu",
			Entrypoint: []string{"bash", "-c", "echo -n 'apples' >&1; echo -n 'oranges' >&2; exit 19;"},
		},
	},
}

var errorLogsTestCase = scenario.Scenario{
	Stack: &scenario.StackConfig{
		ExecutorConfig: noop.ExecutorConfig{
			ExternalHooks: noop.ExecutorConfigExternalHooks{
				JobHandler: func(ctx context.Context, _ string, resultsDir string) (*model.RunCommandResult, error) {
					return executor.WriteJobResults(resultsDir, strings.NewReader("apples"), strings.NewReader("oranges"), 19, nil)
				},
			},
		},
	},
	ResultsChecker: scenario.ManyChecks(
		scenario.FileEquals(model.DownloadFilenameStdout, "apples"),
		scenario.FileEquals(model.DownloadFilenameStderr, "oranges"),
	),
	JobCheckers: []job.CheckStatesFunction{
		job.WaitForSuccessfulCompletion(),
	},
}

func (suite *DevstackErrorLogsSuite) TestCanGetResultsFromErroredJob() {
	for _, testCase := range executorTestCases {
		suite.Run(testCase.Engine.String(), func() {
			docker.MaybeNeedDocker(suite.T(), testCase.Engine == model.EngineDocker)

			scenario := errorLogsTestCase
			scenario.Spec = testCase
			suite.RunScenario(scenario)
		})
	}
}
