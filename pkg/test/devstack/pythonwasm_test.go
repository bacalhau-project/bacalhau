//go:build !(unit && (windows || darwin))

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
func (s *DevstackPythonWASMSuite) SetupSuite() {

}

// Before each test
func (s *DevstackPythonWASMSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(s.T(), err)
}

func (s *DevstackPythonWASMSuite) TearDownTest() {
}

func (s *DevstackPythonWASMSuite) TearDownSuite() {

}

// full end-to-end test of python/wasm:
//
// * use CLI to submit a python job with --deterministic set
// * context (python file and requirements.txt) should be pinned to ipfs by
//   requester node
// * docker executor downloads context and starts wasm container image with the
//   context mounted in

func (s *DevstackPythonWASMSuite) TestPythonWasmVolumes() {
	cmd.Fatal = cmd.FakeFatalErrorHandler

	nodeCount := 1
	inputPath := "/input"
	outputPath := "/output"
	fileContents := "pineapples"

	ctx := context.Background()
	stack, cm := SetupTest(ctx, s.T(), nodeCount, 0, false, computenode.NewDefaultComputeNodeConfig())

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack.TestPythonWasmVolumes")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	tmpDir := s.T().TempDir()

	oldDir, err := os.Getwd()
	require.NoError(s.T(), err)
	err = os.Chdir(tmpDir)
	require.NoError(s.T(), err)
	defer func() {
		err := os.Chdir(oldDir)
		require.NoError(s.T(), err)
	}()

	fileCid, err := devstack.AddTextToNodes(ctx, []byte(fileContents), devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
	require.NoError(s.T(), err)

	// write bytes to main.py
	mainPy := []byte(fmt.Sprintf(`
	import os
	print("LIST /")
	print(os.listdir("/"))
	print("LIST %s")
	print(os.listdir("%s"))
	open("%s/test.txt", "w").write(open("%s").read())
`, outputPath, outputPath, outputPath, inputPath))

	err = ioutil.WriteFile("main.py", mainPy, 0644)
	require.NoError(s.T(), err)

	_, out, err := cmd.ExecuteTestCobraCommand(s.T(), cmd.RootCmd,
		fmt.Sprintf("--api-port=%d", stack.Nodes[0].APIServer.Port),
		"--api-host=localhost",
		"run",
		"-v", fmt.Sprintf("%s:%s", fileCid, inputPath),
		"-o", fmt.Sprintf("%s:%s", "output", outputPath),
		"python",
		"--deterministic",
		"main.py",
	)
	jobID := system.FindJobIDInTestOutput(out)
	require.NoError(s.T(), err)
	log.Debug().Msgf("jobId=%s", jobID)
	time.Sleep(time.Second * 5)

	node := stack.Nodes[0]
	apiUri := node.APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	resolver := apiClient.GetJobStateResolver()
	require.NoError(s.T(), err)
	err = resolver.WaitUntilComplete(ctx, jobID)
	require.NoError(s.T(), err)

	shards, err := resolver.GetShards(ctx, jobID)
	require.NoError(s.T(), err)
	require.True(s.T(), len(shards) > 0)

	shard := shards[0]

	outputDir := s.T().TempDir()
	require.NotEmpty(s.T(), shard.PublishedResult.CID)

	finalOutputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
	err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, finalOutputPath)
	require.NoError(s.T(), err)

	err = filepath.Walk(finalOutputPath,
		func(path string, info os.FileInfo, err error) error {
			require.NoError(s.T(), err)
			log.Debug().Msgf("%s - %d", path, info.Size())
			return err
		})
	require.NoError(s.T(), err)

	stdoutContents, err := ioutil.ReadFile(filepath.Join(finalOutputPath, "stdout"))
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), stdoutContents)

	log.Debug().Msgf("stdoutContents=> %s", stdoutContents)

	// stderrContents, err := ioutil.ReadFile(filepath.Join(finalOutputPath, "stderr"))
	// require.NoError(s.T(), err)
	// require.Empty(s.T(), stderrContents, "stderr should be empty: %s", stderrContents)

	filePath := fmt.Sprintf("%s/output/test.txt", finalOutputPath)
	outputData, err := os.ReadFile(filePath)
	require.NoError(s.T(), err)

	require.Equal(s.T(), fileContents, strings.TrimSpace(string(outputData)))
}
func (s *DevstackPythonWASMSuite) TestSimplestPythonWasmDashC() {
	s.T().Skip("This test fails when run directly after TestPythonWasmVolumes :-(")

	ctx := context.Background()
	stack, cm := SetupTest(ctx, s.T(), 1, 0, false, computenode.NewDefaultComputeNodeConfig())

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/pythonwasmtest/simplestpythonwasmdashc")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	// TODO: see also list_test.go, maybe factor out a common way to do this cli
	// setup
	_, out, err := cmd.ExecuteTestCobraCommand(s.T(), cmd.RootCmd,
		fmt.Sprintf("--api-port=%d", stack.Nodes[0].APIServer.Port),
		"--api-host=localhost",
		"run",
		"python",
		"--deterministic",
		"-c",
		"print(1+1)",
	)
	require.NoError(s.T(), err)

	jobId := strings.TrimSpace(out)
	log.Debug().Msgf("jobId=%s", jobId)

	node := stack.Nodes[0]
	apiUri := node.APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	resolver := apiClient.GetJobStateResolver()
	require.NoError(s.T(), err)
	err = resolver.WaitUntilComplete(ctx, jobId)
	require.NoError(s.T(), err)

}

// TODO: test that > 10MB context is rejected

func (s *DevstackPythonWASMSuite) TestSimplePythonWasm() {
	s.T().Skip("This test fails when run directly after TestPythonWasmVolumes :-(")

	ctx := context.Background()
	stack, cm := SetupTest(ctx, s.T(), 1, 0, false, computenode.NewDefaultComputeNodeConfig())

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/pythonwasmtest/simplepythonwasm")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	tmpDir := s.T().TempDir()

	oldDir, err := os.Getwd()
	require.NoError(s.T(), err)
	err = os.Chdir(tmpDir)
	require.NoError(s.T(), err)
	defer func() {
		err := os.Chdir(oldDir)
		require.NoError(s.T(), err)
	}()

	// write bytes to main.py
	mainPy := []byte("print(1+1)")
	err = ioutil.WriteFile("main.py", mainPy, 0644)
	require.NoError(s.T(), err)

	_, out, err := cmd.ExecuteTestCobraCommand(s.T(), cmd.RootCmd,
		fmt.Sprintf("--api-port=%d", stack.Nodes[0].APIServer.Port),
		"--api-host=localhost",
		"run",
		"python",
		"--deterministic",
		"main.py",
	)
	require.NoError(s.T(), err)
	jobId := strings.TrimSpace(out)
	log.Debug().Msgf("jobId=%s", jobId)
	time.Sleep(time.Second * 5)

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	resolver := apiClient.GetJobStateResolver()
	require.NoError(s.T(), err)
	err = resolver.WaitUntilComplete(ctx, jobId)
	require.NoError(s.T(), err)
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
