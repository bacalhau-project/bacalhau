//go:build !(unit && (windows || darwin))

package devstack

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ComboDriverSuite struct {
	scenario.ScenarioRunner
}

var _ scenario.ScenarioTestSuite = (*ComboDriverSuite)(nil)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestComboDriverSuite(t *testing.T) {
	suite.Run(t, new(ComboDriverSuite))
}

const exampleText = "hello world"

var testcase scenario.TestCase = scenario.TestCase{
	ResultsChecker: scenario.FileEquals(ipfs.DownloadFilenameStdout, exampleText),
	Spec: model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"bash", "-c",
				`cat /inputs/file.txt`,
			},
		},
	},
	Outputs: []model.StorageSpec{
		{
			Name: "outputs",
			Path: "/outputs/",
		},
	},
	JobCheckers: []job.CheckStatesFunction{
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: 1,
		}),
	},
}

// Test that the combo driver gives preference to the filecoin unsealed driver
// also that this does not affect normal jobs where the CID resides on the IPFS driver
func (suite *ComboDriverSuite) TestComboDriverSealed() {
	testcase.Inputs = scenario.StoredText(exampleText, "/inputs/file.txt")
	suite.RunScenario(testcase)
}

func (suite *ComboDriverSuite) TestComboDriverUnsealed() {
	cid := "apples"
	basePath := suite.T().TempDir()
	err := os.MkdirAll(filepath.Join(basePath, cid), os.ModePerm)
	require.NoError(suite.T(), err)

	filePath := filepath.Join(basePath, cid, "file.txt")
	err = os.WriteFile(filePath, []byte(fmt.Sprintf(exampleText)), 0644)
	require.NoError(suite.T(), err)

	suite.SetupStack(&devstack.DevStackOptions{
		NumberOfNodes:        1,
		PublicIPFSMode:       true,
		FilecoinUnsealedPath: fmt.Sprintf("%s/{{.CID}}", basePath),
	}, computenode.NewDefaultComputeNodeConfig())

	suite.RunScenario(testcase)
}
