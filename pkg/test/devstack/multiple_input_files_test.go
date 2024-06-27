//go:build integration || !unit

package devstack

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/cat"
)

type MultipleInputFilesSuite struct {
	scenario.ScenarioRunner
}

func TestMultipleInputFilesSuite(t *testing.T) {
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
				AllowListedLocalPaths: []string{rootSourceDir + scenario.AllowedListedLocalPathsSuffix},
			},
		},
		Inputs: scenario.ManyStores(
			scenario.StoredText(rootSourceDir, "file1\n", filepath.Join(dirCID1, fileName1)),
			scenario.StoredText(rootSourceDir, "file2\n", filepath.Join(dirCID2, fileName2)),
		),
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name:      s.T().Name(),
					Publisher: publisher_local.NewSpecConfig(),
					Engine: wasmmodels.NewWasmEngineBuilder(scenario.InlineData(cat.Program())).
						WithEntrypoint("_start").
						WithParameters(
							filepath.Join(dirCID1, fileName1),
							filepath.Join(dirCID2, fileName2),
						).
						MustBuild(),
				},
			},
		},
		ResultsChecker: scenario.ManyChecks(
			scenario.FileEquals(downloader.DownloadFilenameStdout, "file1\nfile2\n"),
		),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
		},
	}

	s.RunScenario(testCase)
}
