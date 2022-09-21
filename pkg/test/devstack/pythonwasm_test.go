package devstack

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/devstack"

	cmd "github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
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
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
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
	inputPath := "/inputs"
	inWASMOutputPath := "/outputs"
	fileContents := "pineapples"

	ctx := context.Background()
	stack, cm := SetupTest(ctx, suite.T(), nodeCount, 0, computenode.NewDefaultComputeNodeConfig())
	defer TeardownTest(stack, cm)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/pythonwasmtest/pythonwasmvolumes")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	tmpOutputDir, err := ioutil.TempDir("", "bacalhau-test-python-wasm-volumes")
	require.NoError(suite.T(), err)
	defer func() {
		err := os.RemoveAll(tmpOutputDir)
		require.NoError(suite.T(), err)
	}()

	oldDir, err := os.Getwd()
	require.NoError(suite.T(), err)
	err = os.Chdir(tmpOutputDir)
	require.NoError(suite.T(), err)
	defer func() {
		err := os.Chdir(oldDir)
		require.NoError(suite.T(), err)
	}()

	fileCid, err := devstack.AddTextToNodesForTests(ctx, []byte(fileContents), devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
	require.NoError(suite.T(), err)

	// write bytes to main.py
	mainPy := []byte(fmt.Sprintf(`
import os
print("LIST /")
print(os.listdir("/"))
print("LIST %s")
print(os.listdir("%s"))
open("%s/%s", "w").write(open("%s").read())
`, inWASMOutputPath, inWASMOutputPath, inWASMOutputPath, devstack.TmpFileName, inputPath))

	err = ioutil.WriteFile("main.py", mainPy, 0644)
	require.NoError(suite.T(), err)

	_, out, err := cmd.ExecuteTestCobraCommand(suite.T(), cmd.RootCmd,
		fmt.Sprintf("--api-port=%d", stack.Nodes[0].APIServer.Port),
		"--api-host=localhost",
		"run",
		"-v", fmt.Sprintf("%s:%s", fileCid, inputPath),
		"python",
		"--deterministic",
		"main.py",
	)
	require.NoError(suite.T(), err)
	jobID := strings.TrimSpace(out)
	log.Debug().Msgf("jobId=%s", jobID)
	time.Sleep(time.Second * 5)

	node := stack.Nodes[0]
	apiUri := node.APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	resolver := apiClient.GetJobStateResolver()
	require.NoError(suite.T(), err)
	err = resolver.WaitUntilComplete(ctx, jobID)
	require.NoError(suite.T(), err)

	shards, err := resolver.GetShards(ctx, jobID)
	require.NoError(suite.T(), err)
	require.True(suite.T(), len(shards) > 0)

	shard := shards[0]

	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), shard.PublishedResult.CID)

	outputPath := filepath.Join(tmpOutputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
	require.NoError(suite.T(), err)

	filePath := fmt.Sprintf("%s/outputs/%s", outputPath, devstack.TmpFileName)
	stdoutContents, _ := ioutil.ReadFile(outputPath + "/outputs/stdout")
	require.NotEmpty(suite.T(), stdoutContents)
	stderrContents, _ := ioutil.ReadFile(filePath + "/outputs/stderr")
	require.NotEmpty(suite.T(), stderrContents)
	exitCodeContents, _ := ioutil.ReadFile(filePath + "/outputs/exitCode")
	require.Equal(suite.T(), exitCodeContents, "0")

	outputData, err := os.ReadFile(filePath)
	require.NoError(suite.T(), err)

	require.Equal(suite.T(), fileContents, strings.TrimSpace(string(outputData)))
}
func (suite *DevstackPythonWASMSuite) TestSimplestPythonWasmDashC() {
	suite.T().Skip("This test fails when run directly after TestPythonWasmVolumes :-(")

	ctx := context.Background()
	stack, cm := SetupTest(ctx, suite.T(), 1, 0, computenode.NewDefaultComputeNodeConfig())
	defer TeardownTest(stack, cm)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/pythonwasmtest/simplestpythonwasmdashc")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

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

	node := stack.Nodes[0]
	apiUri := node.APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	resolver := apiClient.GetJobStateResolver()
	require.NoError(suite.T(), err)
	err = resolver.WaitUntilComplete(ctx, jobId)
	require.NoError(suite.T(), err)

}

// TODO: test that > 10MB context is rejected

func (suite *DevstackPythonWASMSuite) TestSimplePythonWasm() {
	suite.T().Skip("This test fails when run directly after TestPythonWasmVolumes :-(")

	ctx := context.Background()
	stack, cm := SetupTest(ctx, suite.T(), 1, 0, computenode.NewDefaultComputeNodeConfig())
	defer TeardownTest(stack, cm)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/pythonwasmtest/simplepythonwasm")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

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

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	resolver := apiClient.GetJobStateResolver()
	require.NoError(suite.T(), err)
	err = resolver.WaitUntilComplete(ctx, jobId)
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
