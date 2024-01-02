//go:build integration || !unit

package devstack

import (
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/cat"
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
						filepath.Join(dirCID1, fileName1),
						filepath.Join(dirCID2, fileName2),
					).
					Build(),
			),
		),
		ResultsChecker: scenario.ManyChecks(
			scenario.FileEquals(downloader.DownloadFilenameStdout, "file1\nfile2\n"),
		),
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testCase)
}
