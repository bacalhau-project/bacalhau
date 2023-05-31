//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	enginetesting "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/testing"
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

	testcase := scenario.Scenario{
		Inputs: scenario.StoredText(stdoutText, "data/hello.txt"),
		Spec: model.Spec{
			Engine: enginetesting.WasmMakeEngine(s.T(),
				enginetesting.WasmWithEntrypoint("_start"),
				enginetesting.WasmWithEntryModule(scenario.InlineData(cat.Program())),
				enginetesting.WasmWithParameters(
					"data/hello.txt",
					"does/not/exist.txt",
				),
			),
			Verifier: model.VerifierNoop,
			PublisherSpec: model.PublisherSpec{
				Type: model.PublisherIpfs,
			},
		},
		ResultsChecker: scenario.FileEquals(model.DownloadFilenameStdout, stdoutText),
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}
