//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/job"
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

	testcase := scenario.Scenario{
		Inputs: scenario.StoredText(stdoutText, "data/hello.txt"),
		Spec: testutils.MakeSpecWithOpts(s.T(),
			job.WithPublisher(
				model.PublisherSpec{
					Type: model.PublisherIpfs,
				},
			),
			job.WithEngineSpec(
				model.NewWasmEngineBuilder(scenario.InlineData(cat.Program())).
					WithEntrypoint("_start").
					WithParameters(
						"data/hello.txt",
						"does/not/exist.txt",
					).
					Build(),
			),
		),
		ResultsChecker: scenario.FileEquals(model.DownloadFilenameStdout, stdoutText),
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}
