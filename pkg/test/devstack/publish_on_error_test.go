//go:build integration

package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
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
			Engine:    model.EngineWasm,
			Verifier:  model.VerifierNoop,
			Publisher: model.PublisherIpfs,
			Wasm: model.JobSpecWasm{
				EntryPoint:  scenario.CatFileToStdout.Spec.Wasm.EntryPoint,
				EntryModule: scenario.CatFileToStdout.Spec.Wasm.EntryModule,
				Parameters: []string{
					"data/hello.txt",
					"does/not/exist.txt",
				},
			},
		},
		ResultsChecker: scenario.FileEquals(model.DownloadFilenameStdout, stdoutText),
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testcase)
}
