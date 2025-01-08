//go:build integration || !unit

package devstack

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	dockmodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type DevstackErrorLogsSuite struct {
	scenario.ScenarioRunner
}

func TestDevstackErrorLogsSuite(t *testing.T) {
	suite.Run(t, new(DevstackErrorLogsSuite))
}

func executorTestCases(t testing.TB) []*models.Job {
	return []*models.Job{
		{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Engine: &models.SpecConfig{
						Type:   models.EngineNoop,
						Params: make(map[string]interface{}),
					},
					Publisher: publisher_local.NewSpecConfig(),
				},
			},
		},
		{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Engine: dockmodels.NewDockerEngineBuilder("busybox:1.37.0").
						WithEntrypoint("sh", "-c", "echo -n 'apples' >&1; echo -n 'oranges' >&2; exit 19;").
						MustBuild(),
					Publisher: publisher_local.NewSpecConfig(),
				},
			},
		},
	}
}

var errorLogsTestCase = scenario.Scenario{
	Stack: &scenario.StackConfig{
		ExecutorConfig: noop.ExecutorConfig{
			ExternalHooks: noop.ExecutorConfigExternalHooks{
				JobHandler: func(ctx context.Context, execContext noop.ExecutionContext) (*models.RunCommandResult, error) {
					return executor.WriteJobResults(execContext.ResultsDir, strings.NewReader("apples"), strings.NewReader("oranges"), 19, nil, executor.OutputLimits{
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
	JobCheckers: []scenario.StateChecks{
		scenario.WaitForSuccessfulCompletion(),
	},
}

func (suite *DevstackErrorLogsSuite) TestCanGetResultsFromErroredJob() {
	for _, testCase := range executorTestCases(suite.T()) {
		suite.Run(testCase.Task().Engine.Type, func() {
			docker.EngineSpecRequiresDocker(suite.T(), testCase.Task().Engine)

			s := errorLogsTestCase
			s.Job = testCase
			suite.RunScenario(s)
		})
	}
}
