//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/spec"
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
	stdoutText := "I am a miserable failure\n"

	engineSpec, err := spec.MutateWasmEngineSpec(scenario.CatFileToStdout.Spec.EngineSpec,
		spec.WithParameters("data/hello.txt", "does/not/exist.txt"),
	)
	require.NoError(s.T(), err)

	testcase := scenario.Scenario{
		Inputs: scenario.StoredText(stdoutText, "data/hello.txt"),
		Spec: model.Spec{
			EngineSpec: engineSpec,
			Verifier:   model.VerifierNoop,
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
