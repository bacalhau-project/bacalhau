package devstack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cmd "github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DevstackPythonWASMSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDevstackPythonWASMSuite(t *testing.T) {
	suite.Run(t, new(DevstackPythonWASMSuite))
}

// Before all suite
func (suite *DevstackPythonWASMSuite) SetupAllSuite() {

}

// Before each test
func (suite *DevstackPythonWASMSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *DevstackPythonWASMSuite) TearDownTest() {
}

func (suite *DevstackPythonWASMSuite) TearDownAllSuite() {

}

// full end-to-end test of python/wasm:
//
// * use CLI to submit a python job with --deterministic set
// * context (python file and requirements.txt) should be pinned to ipfs by
//   requester node
// * docker executor downloads context and starts wasm container image with the
//   context mounted in

func (suite *DevstackPythonWASMSuite) TestPythonWasmVolumes() {
	nodeCount := 1
	inputPath := "/input"
	outputPath := "/output"
	fileContents := "pineapples"

	ctx, span := newSpan("TestPythonWasmVolumes")
	defer span.End()
	stack, cm := SetupTest(suite.T(), nodeCount, 0, computenode.NewDefaultComputeNodeConfig())
	defer TeardownTest(stack, cm)

	nodeIds, err := stack.GetNodeIds()
	require.NoError(suite.T(), err)
	tmpDir, err := ioutil.TempDir("", "devstack_test")
	require.NoError(suite.T(), err)
	defer func() {
		err := os.RemoveAll(tmpDir)
		require.NoError(suite.T(), err)
	}()

	oldDir, err := os.Getwd()
	require.NoError(suite.T(), err)
	err = os.Chdir(tmpDir)
	require.NoError(suite.T(), err)
	defer func() {
		err := os.Chdir(oldDir)
		require.NoError(suite.T(), err)
	}()

	fileCid, err := stack.AddTextToNodes(nodeCount, []byte(fileContents))
	require.NoError(suite.T(), err)

	// write bytes to main.py
	mainPy := []byte(fmt.Sprintf(`
import os
print("LIST /")
print(os.listdir("/"))
#print("LIST /input")
#print(os.listdir("/input"))
print("LIST /output")
print(os.listdir("/output"))
print("LIST /job")
print(os.listdir("/job"))
open("%s/test.txt", "w").write(open("%s").read())
`, outputPath, inputPath))

	err = ioutil.WriteFile("main.py", mainPy, 0644)
	require.NoError(suite.T(), err)

	_, out, err := cmd.ExecuteTestCobraCommand(suite.T(), cmd.RootCmd,
		fmt.Sprintf("--api-port=%d", stack.Nodes[0].APIServer.Port),
		"--api-host=localhost",
		"run",
		"-v", fmt.Sprintf("%s:%s", fileCid, inputPath),
		"-o", fmt.Sprintf("%s:%s", "output", outputPath),
		"python",
		"--deterministic",
		"main.py",
	)
	require.NoError(suite.T(), err)
	jobId := strings.TrimSpace(out)
	log.Debug().Msgf("jobId=%s", jobId)
	time.Sleep(time.Second * 5)
	err = stack.WaitForJob(ctx, jobId,
		devstack.WaitForJobThrowErrors([]executor.JobStateType{
			executor.JobStateCancelled,
			executor.JobStateError,
		}),
		devstack.WaitForJobAllHaveState(nodeIds, executor.JobStateComplete),
	)
	require.NoError(suite.T(), err)

	nodeID := nodeIds[0]
	node, err := stack.GetNode(ctx, nodeID)
	require.NoError(suite.T(), err)

	apiUri := node.APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	jobStates, err := apiClient.GetExecutionStates(ctx, jobId)
	require.NoError(suite.T(), err)

	state, ok := jobStates[nodeID]
	require.True(suite.T(), ok)

	outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-devstack-test")
	require.NoError(suite.T(), err)

	outputPath = filepath.Join(outputDir, state.ResultsID)
	err = node.IpfsClient.Get(ctx, state.ResultsID, outputPath)
	require.NoError(suite.T(), err)

	filePath := fmt.Sprintf("%s/output/test.txt", outputPath)
	outputData, err := os.ReadFile(filePath)
	require.NoError(suite.T(), err)

	require.Equal(suite.T(), fileContents, strings.TrimSpace(string(outputData)))
}
func (suite *DevstackPythonWASMSuite) TestSimplestPythonWasmDashC() {
	suite.T().Skip("This test fails when run directly after TestPythonWasmVolumes :-(")

	ctx, span := newSpan("TestSimplestPythonWasmDashC")
	defer span.End()
	stack, cm := SetupTest(suite.T(), 1, 0, computenode.NewDefaultComputeNodeConfig())
	defer TeardownTest(stack, cm)

	nodeIds, err := stack.GetNodeIds()
	require.NoError(suite.T(), err)

	// TODO: see also list_test.go, maybe factor out a common way to do this cli
	// setup
	_, out, err := cmd.ExecuteTestCobraCommand(suite.T(), cmd.RootCmd,
		fmt.Sprintf("--api-port=%d", stack.Nodes[0].APIServer.Port),
		"--api-host=localhost",
		"run",
		"python",
		"--deterministic",
		"-c",
		"print(1+1)",
	)
	require.NoError(suite.T(), err)

	jobId := strings.TrimSpace(out)
	log.Debug().Msgf("jobId=%s", jobId)
	// wait for the job to complete across all nodes
	err = stack.WaitForJob(ctx, jobId,
		devstack.WaitForJobThrowErrors([]executor.JobStateType{
			executor.JobStateCancelled,
			executor.JobStateError,
		}),
		devstack.WaitForJobAllHaveState(nodeIds, executor.JobStateComplete),
	)
	require.NoError(suite.T(), err)

	// load result from ipfs and check it
	// TODO: see devStackDockerStorageTest for how to do this

}

// TODO: test that > 10MB context is rejected

func (suite *DevstackPythonWASMSuite) TestSimplePythonWasm() {
	suite.T().Skip("This test fails when run directly after TestPythonWasmVolumes :-(")

	ctx, span := newSpan("TestSimplePythonWasm")
	defer span.End()
	stack, cm := SetupTest(suite.T(), 1, 0, computenode.NewDefaultComputeNodeConfig())
	defer TeardownTest(stack, cm)

	nodeIds, err := stack.GetNodeIds()
	require.NoError(suite.T(), err)
	tmpDir, err := ioutil.TempDir("", "devstack_test")
	require.NoError(suite.T(), err)
	defer func() {
		err := os.RemoveAll(tmpDir)
		require.NoError(suite.T(), err)
	}()

	oldDir, err := os.Getwd()
	require.NoError(suite.T(), err)
	err = os.Chdir(tmpDir)
	require.NoError(suite.T(), err)
	defer func() {
		err := os.Chdir(oldDir)
		require.NoError(suite.T(), err)
	}()

	// write bytes to main.py
	mainPy := []byte("print(1+1)")
	err = ioutil.WriteFile("main.py", mainPy, 0644)
	require.NoError(suite.T(), err)

	_, out, err := cmd.ExecuteTestCobraCommand(suite.T(), cmd.RootCmd,
		fmt.Sprintf("--api-port=%d", stack.Nodes[0].APIServer.Port),
		"--api-host=localhost",
		"run",
		"python",
		"--deterministic",
		"main.py",
	)
	require.NoError(suite.T(), err)
	jobId := strings.TrimSpace(out)
	log.Debug().Msgf("jobId=%s", jobId)
	time.Sleep(time.Second * 5)
	err = stack.WaitForJob(ctx, jobId,
		devstack.WaitForJobThrowErrors([]executor.JobStateType{
			executor.JobStateCancelled,
			executor.JobStateError,
		}),
		devstack.WaitForJobAllHaveState(nodeIds, executor.JobStateComplete),
	)
	require.NoError(suite.T(), err)

}

// func TestPythonWasmWithRequirements(t *testing.T) {
// 	tmpDir, err := ioutil.TempDir("", "devstack_test")
// 	require.NoError(t, err)
// 	defer func() {
// 		err := os.RemoveAll(tmpDir)
// 		require.NoError(t, err)
// 	}()

// 	oldDir, err := os.Getwd()
// 	require.NoError(t, err)
// 	err = os.Chdir(tmpDir)
// 	require.NoError(t, err)
// 	defer func() {
// 		err := os.Chdir(oldDir)
// 		require.NoError(t, err)
// 	}()

// 	_, out, err := ExecuteTestCobraCommand(t, RootCmd,
// 		"run",
// 		"python",
// 		"--deterministic",
// 		"main.py",
// 	)
// 	require.NoError(t, err)

// }
