//go:build integration

package devstack

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
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

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestComboDriverSuite(t *testing.T) {
	suite.Run(t, new(ComboDriverSuite))
}

const exampleText = "hello world"

var testcase = scenario.Scenario{
	ResultsChecker: scenario.FileEquals(model.DownloadFilenameStdout, exampleText),
	Spec: model.Spec{
		Engine:    model.EngineWasm,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Wasm: model.JobSpecWasm{
			EntryPoint:  scenario.CatFileToStdout.Spec.Wasm.EntryPoint,
			EntryModule: scenario.CatFileToStdout.Spec.Wasm.EntryModule,
			Parameters: []string{
				`/inputs/file.txt`,
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
		job.WaitForSuccessfulCompletion(),
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

	testcase.Stack = &scenario.StackConfig{
		DevStackOptions: &devstack.DevStackOptions{
			NumberOfHybridNodes:  1,
			PublicIPFSMode:       false,
			FilecoinUnsealedPath: fmt.Sprintf("%s/{{.CID}}", basePath),
		},
	}

	suite.RunScenario(testcase)
}
