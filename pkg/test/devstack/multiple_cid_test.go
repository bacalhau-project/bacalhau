//go:build !(unit && (windows || darwin))

package devstack

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
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

	testCase := scenario.TestCase{
		Inputs: scenario.ManyStores(
			scenario.StoredText("file1", filepath.Join(dirCID1, fileName1)),
			scenario.StoredText("file2", filepath.Join(dirCID2, fileName2)),
		),
		Outputs: []model.StorageSpec{},
		Spec: model.Spec{
			Engine:    model.EngineDocker,
			Verifier:  model.VerifierNoop,
			Publisher: model.PublisherIpfs,
			Docker: model.JobSpecDocker{
				Image: "ubuntu",
				Entrypoint: []string{
					"bash",
					"-c",
					fmt.Sprintf("ls && ls %s && ls %s", dirCID1, dirCID2),
				},
			},
		},
		ResultsChecker: scenario.ManyChecks(
			scenario.FileContains(ipfs.DownloadFilenameStdout, fileName1, 23),
			scenario.FileContains(ipfs.DownloadFilenameStdout, fileName2, 23),
		),
		JobCheckers: []job.CheckStatesFunction{
			job.WaitThrowErrors([]model.JobStateType{
				model.JobStateError,
			}),
			job.WaitForJobStates(map[model.JobStateType]int{
				model.JobStateCompleted: 1,
			}),
		},
	}

	s.RunScenario(testCase)
}
