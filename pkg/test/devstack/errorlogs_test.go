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
	spec_docker "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
	enginetesting "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/testing"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type DevstackErrorLogsSuite struct {
	scenario.ScenarioRunner
}

func TestDevstackErrorLogsSuite(t *testing.T) {
	suite.Run(t, new(DevstackErrorLogsSuite))
}

func executorTestCases(t testing.TB) []model.Spec {
	return []model.Spec{
		{
			Engine: enginetesting.NoopMakeEngine(t, "noop"),
			PublisherSpec: model.PublisherSpec{
				Type: model.PublisherIpfs,
			},
		},
		{
			Engine: enginetesting.DockerMakeEngine(t,
				enginetesting.DockerWithImage("ubuntu"),
				enginetesting.DockerWithEntrypoint("bash", "-c", "echo -n 'apples' >&1; echo -n 'oranges' >&2; exit 19;"),
			),
			PublisherSpec: model.PublisherSpec{
				Type: model.PublisherIpfs,
			},
		},
	}
}

var errorLogsTestCase = scenario.Scenario{
	Stack: &scenario.StackConfig{
		ExecutorConfig: noop.ExecutorConfig{
			ExternalHooks: noop.ExecutorConfigExternalHooks{
				JobHandler: func(ctx context.Context, job model.Job, resultsDir string) (*model.RunCommandResult, error) {
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
	for _, testCase := range executorTestCases(suite.T()) {
		suite.Run(testCase.Engine.String(), func() {
			docker.MaybeNeedDocker(suite.T(), testCase.Engine.Schema == spec_docker.EngineType)

			scenario := errorLogsTestCase
			scenario.Spec = testCase
			suite.RunScenario(scenario)
		})
	}
}
