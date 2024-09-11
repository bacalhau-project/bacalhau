//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/cat"

	"github.com/stretchr/testify/suite"
)

type PublishOnErrorSuite struct {
	scenario.ScenarioRunner
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestPublishOnErrorSuite(t *testing.T) {
	suite.Run(t, new(PublishOnErrorSuite))
}

func (s *PublishOnErrorSuite) TestPublishOnError() {
	stdoutText := "I am a miserable failure\n"

	rootSourceDir := s.T().TempDir()

	testcase := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: &devstack.DevStackOptions{
				AllowListedLocalPaths: []string{rootSourceDir + scenario.AllowedListedLocalPathsSuffix},
			},
		},
		Inputs: scenario.StoredText(rootSourceDir, stdoutText, "data/hello.txt"),
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name:      s.T().Name(),
					Publisher: publisher_local.NewSpecConfig(),
					Engine: wasmmodels.NewWasmEngineBuilder(scenario.InlineData(cat.Program())).
						WithEntrypoint("_start").
						WithParameters(
							"data/hello.txt",
							"does/not/exist.txt",
						).
						MustBuild(),
				},
			},
		},
		ResultsChecker: scenario.FileEquals(downloader.DownloadFilenameStdout, stdoutText),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}
