//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
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
	// TODO(forrest): [feels] stay positive!
	stdoutText := "I am a miserable failure\n"

	catFileToStdOutWasmEngine, err := model.WasmEngineSpecFromEngineSpec(scenario.CatFileToStdout.Spec.EngineSpec)
	s.Require().NoError(err)
	testcase := scenario.Scenario{
		Inputs: scenario.StoredText(stdoutText, "data/hello.txt"),
		Spec: model.Spec{
			EngineDeprecated: model.EngineWasm,
			Verifier:         model.VerifierNoop,
			PublisherSpec: model.PublisherSpec{
				Type: model.PublisherIpfs,
			},
			EngineSpec: model.WasmEngineSpec{
				Entrypoint:  catFileToStdOutWasmEngine.Entrypoint,
				EntryModule: catFileToStdOutWasmEngine.EntryModule,
				Parameters: []string{
					"data/hello.txt",
					"does/not/exist.txt",
				},
			}.AsEngineSpec(),
		},
		ResultsChecker: scenario.FileEquals(model.DownloadFilenameStdout, stdoutText),
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}
