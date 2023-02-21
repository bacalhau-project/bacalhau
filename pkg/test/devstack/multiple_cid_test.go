//go:build integration

package devstack

import (
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/suite"
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

	testCase := scenario.Scenario{
		Inputs: scenario.ManyStores(
			scenario.StoredText("file1\n", filepath.Join(dirCID1, fileName1)),
			scenario.StoredText("file2\n", filepath.Join(dirCID2, fileName2)),
		),
		Spec: model.Spec{
			Engine:    model.EngineWasm,
			Verifier:  model.VerifierNoop,
			Publisher: model.PublisherIpfs,
			Wasm: model.JobSpecWasm{
				EntryPoint:  scenario.CatFileToStdout.Spec.Wasm.EntryPoint,
				EntryModule: scenario.CatFileToStdout.Spec.Wasm.EntryModule,
				Parameters: []string{
					filepath.Join(dirCID1, fileName1),
					filepath.Join(dirCID2, fileName2),
				},
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
