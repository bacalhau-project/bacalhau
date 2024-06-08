//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
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
				AllowListedLocalPaths: []string{rootSourceDir + "/*"},
			},
		},
		Inputs: scenario.StoredText(rootSourceDir, stdoutText, "data/hello.txt"),
		Spec: testutils.MakeSpecWithOpts(s.T(),
			legacy_job.WithPublisher(
				model.PublisherSpec{
					Type: model.PublisherLocal,
				},
			),
			legacy_job.WithEngineSpec(
				model.NewWasmEngineBuilder(scenario.InlineData(cat.Program())).
					WithEntrypoint("_start").
					WithParameters(
						"data/hello.txt",
						"does/not/exist.txt",
					).
					Build(),
			),
		),
		ResultsChecker: scenario.FileEquals(downloader.DownloadFilenameStdout, stdoutText),
		JobCheckers: []legacy_job.CheckStatesFunction{
			legacy_job.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}
