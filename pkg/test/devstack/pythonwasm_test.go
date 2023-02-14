//go:build integration

package devstack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/node"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"

	cmd "github.com/filecoin-project/bacalhau/cmd/bacalhau"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/requester/publicapi"
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

// Before each test
func (s *DevstackPythonWASMSuite) SetupTest() {
	docker.MustHaveDocker(s.T())

	logger.ConfigureTestLogging(s.T())
	err := system.InitConfigForTesting(s.T())
	require.NoError(s.T(), err)
}

// full end-to-end test of python/wasm:
//
// * use CLI to submit a python job with --deterministic set
// * context (python file and requirements.txt) should be pinned to ipfs by
//   requester node
// * docker executor downloads context and starts wasm container image with the
//   context mounted in

func (s *DevstackPythonWASMSuite) TestPythonWasmVolumes() {
	testutils.SkipIfArm(s.T(), "https://github.com/filecoin-project/bacalhau/issues/1268")
	cmd.Fatal = cmd.FakeFatalErrorHandler

	nodeCount := 1
	inputPath := "/input"
	outputPath := "/output"
	fileContents := "pineapples"

	ctx := context.Background()
	stack, _ := testutils.SetupTest(ctx, s.T(), nodeCount, 0, false,
		node.NewComputeConfigWithDefaults(),
		node.NewRequesterConfigWithDefaults())

	tmpDir := s.T().TempDir()

	oldDir, err := os.Getwd()
	require.NoError(s.T(), err)
	err = os.Chdir(tmpDir)
	require.NoError(s.T(), err)
	defer func() {
		err := os.Chdir(oldDir)
		require.NoError(s.T(), err)
	}()

	fileCid, err := ipfs.AddTextToNodes(ctx, []byte(fileContents), devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
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

	err = os.WriteFile("main.py", mainPy, 0644)
	require.NoError(s.T(), err)

	_, out, err := cmd.ExecuteTestCobraCommand(s.T(),
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
	apiClient := publicapi.NewRequesterAPIClient(apiUri)
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

	stdoutContents, err := os.ReadFile(filepath.Join(finalOutputPath, "stdout"))
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
	testutils.SkipIfArm(s.T(), "https://github.com/filecoin-project/bacalhau/issues/1268")
	cmd.Fatal = cmd.FakeFatalErrorHandler

	ctx := context.Background()
	stack, _ := testutils.SetupTest(ctx, s.T(), 1, 0, false,
		node.NewComputeConfigWithDefaults(),
		node.NewRequesterConfigWithDefaults())

	// TODO: see also list_test.go, maybe factor out a common way to do this cli
	// setup
	_, out, err := cmd.ExecuteTestCobraCommand(s.T(),
		fmt.Sprintf("--api-port=%d", stack.Nodes[0].APIServer.Port),
		"--api-host=localhost",
		"run",
		"python",
		"--deterministic",
		"-c",
		"print(1+1)",
	)
	require.NoError(s.T(), err)

	jobId := system.FindJobIDInTestOutput(out)
	require.NoError(s.T(), err)
	log.Debug().Msgf("jobId=%s", jobId)
	time.Sleep(time.Second * 5)

	node := stack.Nodes[0]
	apiUri := node.APIServer.GetURI()
	apiClient := publicapi.NewRequesterAPIClient(apiUri)
	resolver := apiClient.GetJobStateResolver()
	require.NoError(s.T(), err)
	err = resolver.WaitUntilComplete(ctx, jobId)
	require.NoError(s.T(), err)

}

// TODO: test that > 10MB context is rejected

func (s *DevstackPythonWASMSuite) TestSimplePythonWasm() {
	testutils.SkipIfArm(s.T(), "https://github.com/filecoin-project/bacalhau/issues/1268")
	cmd.Fatal = cmd.FakeFatalErrorHandler

	ctx := context.Background()
	stack, _ := testutils.SetupTest(ctx, s.T(), 1, 0, false,
		node.NewComputeConfigWithDefaults(),
		node.NewRequesterConfigWithDefaults())

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
	mainPy := []byte("print(1+1)\n")
	err = os.WriteFile("main.py", mainPy, 0644)
	require.NoError(s.T(), err)

	_, out, err := cmd.ExecuteTestCobraCommand(s.T(),
		fmt.Sprintf("--api-port=%d", stack.Nodes[0].APIServer.Port),
		"--api-host=localhost",
		"run",
		"python",
		"--deterministic",
		"main.py",
	)
	require.NoError(s.T(), err)

	jobId := system.FindJobIDInTestOutput(out)
	require.NotEmpty(s.T(), jobId, "Unable to find Job ID in", out)
	log.Debug().Msgf("jobId=%s", jobId)
	time.Sleep(time.Second * 5)

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewRequesterAPIClient(apiUri)
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
