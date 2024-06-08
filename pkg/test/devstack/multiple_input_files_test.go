//go:build integration || !unit

package devstack

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/cat"
)

type MultipleInputFilesSuite struct {
	scenario.ScenarioRunner
}

func TestMultipleCIDSuite(t *testing.T) {
	suite.Run(t, new(MultipleInputFilesSuite))
}

func (s *MultipleInputFilesSuite) TestMultipleFiles() {
	dirCID1 := "/input-1"
	dirCID2 := "/input-2"

	fileName1 := "hello-cid-1.txt"
	fileName2 := "hello-cid-2.txt"

	rootSourceDir := s.T().TempDir()

	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: &devstack.DevStackOptions{
				AllowListedLocalPaths: []string{rootSourceDir + "/*"},
			},
		},
		Inputs: scenario.ManyStores(
			scenario.StoredText(rootSourceDir, "file1\n", filepath.Join(dirCID1, fileName1)),
			scenario.StoredText(rootSourceDir, "file2\n", filepath.Join(dirCID2, fileName2)),
		),
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
						filepath.Join(dirCID1, fileName1),
						filepath.Join(dirCID2, fileName2),
					).
					Build(),
			),
		),
		ResultsChecker: scenario.ManyChecks(
			scenario.FileEquals(downloader.DownloadFilenameStdout, "file1\nfile2\n"),
		),
		JobCheckers: []legacy_job.CheckStatesFunction{
			legacy_job.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testCase)
}
