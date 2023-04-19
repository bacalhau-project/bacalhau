//go:build integration || !unit

package devstack

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/spec"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type MultipleCIDSuite struct {
	scenario.ScenarioRunner
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestMultipleCIDSuite(t *testing.T) {
	suite.Run(t, new(MultipleCIDSuite))
}

func (s *MultipleCIDSuite) TestMultipleCIDs() {
	dirCID1 := "/input-1"
	dirCID2 := "/input-2"

	fileName1 := "hello-cid-1.txt"
	fileName2 := "hello-cid-2.txt"

	engineSpec, err := spec.MutateWasmEngineSpec(scenario.CatFileToStdout.Spec.EngineSpec,
		spec.WithParameters(filepath.Join(dirCID1, fileName1), filepath.Join(dirCID2, fileName2)),
	)
	require.NoError(s.T(), err)

	testCase := scenario.Scenario{
		Inputs: scenario.ManyStores(
			scenario.StoredText("file1\n", filepath.Join(dirCID1, fileName1)),
			scenario.StoredText("file2\n", filepath.Join(dirCID2, fileName2)),
		),
		Spec: model.Spec{
			EngineSpec: engineSpec,
			Verifier:   model.VerifierNoop,
			PublisherSpec: model.PublisherSpec{
				Type: model.PublisherIpfs,
			},
		},
		ResultsChecker: scenario.ManyChecks(
			scenario.FileEquals(model.DownloadFilenameStdout, "file1\nfile2\n"),
		),
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testCase)
}
