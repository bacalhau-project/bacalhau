//go:build integration || !unit

package bacalhau

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestGetSuite(t *testing.T) {
	suite.Run(t, new(GetSuite))
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type GetSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// Before each test
func (suite *GetSuite) SetupTest() {
	testutils.MustHaveDocker(suite.T())

	logger.ConfigureTestLogging(suite.T())
	require.NoError(suite.T(), system.InitConfigForTesting(suite.T()))
	suite.rootCmd = RootCmd
}

func testResultsFolderStructure(t *testing.T, baseFolder, hostID string) {
	files := []string{}
	err := filepath.Walk(baseFolder, func(path string, _ os.FileInfo, _ error) error {
		usePath := strings.Replace(path, baseFolder, "", 1)
		if usePath != "" {
			files = append(files, usePath)
		}
		return nil
	})
	require.NoError(t, err, "Error walking results directory")

	shortID := system.GetShortID(hostID)

	// if you change the docker run command in any way we need to change this
	resultsCID := "QmR92HM96X3seZEaRWXfRJDDFHKcprqbmQsEQ9uhbrA7MQ"

	expected := []string{
		"/" + ipfs.DownloadVolumesFolderName,
		"/" + ipfs.DownloadVolumesFolderName + "/data",
		"/" + ipfs.DownloadVolumesFolderName + "/data/apples",
		"/" + ipfs.DownloadVolumesFolderName + "/data/apples/file.txt",
		"/" + ipfs.DownloadVolumesFolderName + "/data/file.txt",
		"/" + ipfs.DownloadVolumesFolderName + "/outputs",
		"/" + ipfs.DownloadVolumesFolderName + "/" + ipfs.DownloadFilenameStderr,
		"/" + ipfs.DownloadVolumesFolderName + "/" + ipfs.DownloadFilenameStdout,
		"/" + ipfs.DownloadShardsFolderName,
		"/" + ipfs.DownloadShardsFolderName + "/0_node_" + shortID,
		"/" + ipfs.DownloadShardsFolderName + "/0_node_" + shortID + "/data",
		"/" + ipfs.DownloadShardsFolderName + "/0_node_" + shortID + "/data/apples",
		"/" + ipfs.DownloadShardsFolderName + "/0_node_" + shortID + "/data/apples/file.txt",
		"/" + ipfs.DownloadShardsFolderName + "/0_node_" + shortID + "/data/file.txt",
		"/" + ipfs.DownloadShardsFolderName + "/0_node_" + shortID + "/exitCode",
		"/" + ipfs.DownloadShardsFolderName + "/0_node_" + shortID + "/outputs",
		"/" + ipfs.DownloadShardsFolderName + "/0_node_" + shortID + "/stderr",
		"/" + ipfs.DownloadShardsFolderName + "/0_node_" + shortID + "/stdout",
		"/" + ipfs.DownloadCIDsFolderName,
		"/" + ipfs.DownloadCIDsFolderName + "/" + resultsCID,
		"/" + ipfs.DownloadCIDsFolderName + "/" + resultsCID + "/data",
		"/" + ipfs.DownloadCIDsFolderName + "/" + resultsCID + "/data/apples",
		"/" + ipfs.DownloadCIDsFolderName + "/" + resultsCID + "/data/apples/file.txt",
		"/" + ipfs.DownloadCIDsFolderName + "/" + resultsCID + "/data/file.txt",
		"/" + ipfs.DownloadCIDsFolderName + "/" + resultsCID + "/exitCode",
		"/" + ipfs.DownloadCIDsFolderName + "/" + resultsCID + "/outputs",
		"/" + ipfs.DownloadCIDsFolderName + "/" + resultsCID + "/stderr",
		"/" + ipfs.DownloadCIDsFolderName + "/" + resultsCID + "/stdout",
	}

	require.Equal(t, strings.Join(expected, "\n"), strings.Join(files, "\n"), "The discovered results output structure was not correct")
}

func testDownloadOutput(t *testing.T, cmdOutput, jobID, outputDir string) {
	require.True(t, strings.Contains(
		cmdOutput,
		fmt.Sprintf("Results for job '%s'", jobID),
	), "Job ID not found in output")
	require.True(t, strings.Contains(
		cmdOutput,
		fmt.Sprintf("%s", outputDir),
	), "Download location not found in output")

}

func setupTempWorkingDir(t *testing.T) (string, func()) {
	// switch wd to a temp dir so we are not writing folders to the current directory
	// (the point of this test is to see what happens when we DONT pass --output-dir)
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	// Below looks redundant but it is necessary on Darwin which does some
	// freaky symlinking of root dirs: https://unix.stackexchange.com/a/63565
	newTempDir, err := os.Getwd()
	require.NoError(t, err)
	return newTempDir, func() {
		os.Chdir(originalWd)
	}
}

func getDockerRunArgs(
	t *testing.T,
	stack *devstack.DevStack,
	extraArgs []string,
) []string {
	swarmAddresses, err := stack.Nodes[0].IPFSClient.SwarmAddresses(context.Background())
	require.NoError(t, err)
	args := []string{
		"docker", "run",
		"--api-host", stack.Nodes[0].APIServer.Host,
		"--api-port", fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		"-o", "data:/data",
		"--wait",
	}
	args = append(args, extraArgs...)
	args = append(args,
		"ubuntu",
		"--",
		"bash", "-c",
		"echo hello > /data/file.txt && echo hello && mkdir /data/apples && echo oranges > /data/apples/file.txt",
	)
	return args
}

// this tests that when we do docker run with no --output-dir
// it makes it's own folder to put the results in and does not splat results
// all over the current directory
func (s *GetSuite) TestDockerRunWriteToJobFolderAutoDownload() {
	ctx := context.Background()
	stack, _ := testutils.SetupTest(ctx, s.T(), 1, 0, false,
		node.NewComputeConfigWithDefaults(),
		requesternode.NewDefaultRequesterNodeConfig(),
	)
	*ODR = *NewDockerRunOptions()

	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	args := getDockerRunArgs(s.T(), stack, []string{
		"--wait",
		"--download",
	})
	_, runOutput, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(runOutput)
	hostID := stack.Nodes[0].HostID
	outputFolder := filepath.Join(tempDir, getDefaultJobFolder(jobID))
	testDownloadOutput(s.T(), runOutput, jobID, tempDir)
	testResultsFolderStructure(s.T(), outputFolder, hostID)

}

// this tests that when we do docker run with an --output-dir
// the results layout adheres to the expected folder layout
func (s *GetSuite) TestDockerRunWriteToJobFolderNamedDownload() {
	ctx := context.Background()
	stack, _ := testutils.SetupTest(ctx, s.T(), 1, 0, false,
		node.NewComputeConfigWithDefaults(),
		requesternode.NewDefaultRequesterNodeConfig(),
	)
	*ODR = *NewDockerRunOptions()

	tempDir, err := os.MkdirTemp("", "docker-run-download-test")
	require.NoError(s.T(), err)

	args := getDockerRunArgs(s.T(), stack, []string{
		"--wait",
		"--download",
		"--output-dir", tempDir,
	})
	_, runOutput, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(runOutput)
	hostID := stack.Nodes[0].HostID
	testDownloadOutput(s.T(), runOutput, jobID, tempDir)
	testResultsFolderStructure(s.T(), tempDir, hostID)
}

// this tests that when we do get with no --output-dir
// it makes it's own folder to put the results in and does not splat results
// all over the current directory
func (s *GetSuite) TestGetWriteToJobFolderAutoDownload() {
	ctx := context.Background()
	stack, _ := testutils.SetupTest(ctx, s.T(), 1, 0, false,
		node.NewComputeConfigWithDefaults(),
		requesternode.NewDefaultRequesterNodeConfig(),
	)
	*ODR = *NewDockerRunOptions()
	*OG = *NewGetOptions()

	swarmAddresses, err := stack.Nodes[0].IPFSClient.SwarmAddresses(context.Background())
	require.NoError(s.T(), err)
	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	args := getDockerRunArgs(s.T(), stack, []string{
		"--wait",
	})
	_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)
	hostID := stack.Nodes[0].HostID

	_, getOutput, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "get",
		"--api-host", stack.Nodes[0].APIServer.Host,
		"--api-port", fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		jobID,
	)
	require.NoError(s.T(), err, "Error getting results")

	testDownloadOutput(s.T(), getOutput, jobID, filepath.Join(tempDir, getDefaultJobFolder(jobID)))
	testResultsFolderStructure(s.T(), filepath.Join(tempDir, getDefaultJobFolder(jobID)), hostID)
}

// this tests that when we do get with an --output-dir
// the results layout adheres to the expected folder layout
func (s *GetSuite) TestGetWriteToJobFolderNamedDownload() {
	ctx := context.Background()
	stack, _ := testutils.SetupTest(ctx, s.T(), 1, 0, false,
		node.NewComputeConfigWithDefaults(),
		requesternode.NewDefaultRequesterNodeConfig(),
	)
	*ODR = *NewDockerRunOptions()
	*OG = *NewGetOptions()

	swarmAddresses, err := stack.Nodes[0].IPFSClient.SwarmAddresses(ctx)
	require.NoError(s.T(), err)

	tempDir, err := os.MkdirTemp("", "docker-run-download-test")
	require.NoError(s.T(), err)

	args := getDockerRunArgs(s.T(), stack, []string{
		"--wait",
	})
	_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, args...)

	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)
	hostID := stack.Nodes[0].HostID

	_, getOutput, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "get",
		"--api-host", stack.Nodes[0].APIServer.Host,
		"--api-port", fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		"--output-dir", tempDir,
		jobID,
	)
	require.NoError(s.T(), err, "Error getting results")
	testDownloadOutput(s.T(), getOutput, jobID, tempDir)
	testResultsFolderStructure(s.T(), tempDir, hostID)
}
