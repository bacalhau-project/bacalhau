//go:build integration || !unit

package devstack

import (
	"context"
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type DevstackErrorLogsSuite struct {
	scenario.ScenarioRunner
}

func TestDevstackErrorLogsSuite(t *testing.T) {
	suite.Run(t, new(DevstackErrorLogsSuite))
}

func executorTestCases(t testing.TB) []model.Spec {
	return []model.Spec{
		testutils.MakeSpecWithOpts(t,
			job.WithPublisher(
				model.PublisherSpec{Type: model.PublisherIpfs},
			),
		),
		testutils.MakeSpecWithOpts(t,
			job.WithEngineSpec(
				model.NewDockerEngineBuilder("ubuntu").
					WithEntrypoint("bash", "-c", "echo -n 'apples' >&1; echo -n 'oranges' >&2; exit 19;").
					Build(),
			),
			job.WithPublisher(
				model.PublisherSpec{Type: model.PublisherIpfs},
			),
		),
	}
}

var errorLogsTestCase = scenario.Scenario{
	Stack: &scenario.StackConfig{
		ExecutorConfig: noop.ExecutorConfig{
			ExternalHooks: noop.ExecutorConfigExternalHooks{
				JobHandler: func(ctx context.Context, _ string, resultsDir string) (*models.RunCommandResult, error) {
					return executor.WriteJobResults(resultsDir, strings.NewReader("apples"), strings.NewReader("oranges"), 19, nil, executor.OutputLimits{
						MaxStdoutFileLength:   system.MaxStdoutFileLength,
						MaxStdoutReturnLength: system.MaxStdoutReturnLength,
						MaxStderrFileLength:   system.MaxStderrFileLength,
						MaxStderrReturnLength: system.MaxStderrReturnLength,
					}), nil
				},
			},
		},
	},
	ResultsChecker: scenario.ManyChecks(
		scenario.FileEquals(downloader.DownloadFilenameStdout, "apples"),
		scenario.FileEquals(downloader.DownloadFilenameStderr, "oranges"),
	),
	JobCheckers: []job.CheckStatesFunction{
		job.WaitForSuccessfulCompletion(),
	},
}

func (suite *DevstackErrorLogsSuite) TestCanGetResultsFromErroredJob() {
	for _, testCase := range executorTestCases(suite.T()) {
		suite.Run(testCase.EngineSpec.String(), func() {
			docker.EngineSpecRequiresDocker(suite.T(), testCase.EngineSpec)

			scenario := errorLogsTestCase
			scenario.Spec = testCase
			suite.RunScenario(scenario)
		})
	}
}
