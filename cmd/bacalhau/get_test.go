package bacalhau

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	devstack_tests "github.com/filecoin-project/bacalhau/pkg/test/devstack"
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

// Before all suite
func (suite *GetSuite) SetupAllSuite() {

}

// Before each test
func (suite *GetSuite) SetupTest() {
	require.NoError(suite.T(), system.InitConfigForTesting())
	suite.rootCmd = RootCmd
}

func (suite *GetSuite) TearDownTest() {
}

func (suite *GetSuite) TearDownAllSuite() {

}

func testResultsFolderStructure(t *testing.T, baseFolder, hostID string) {
	files := []string{}
	err := filepath.Walk(baseFolder, func(path string, info os.FileInfo, err error) error {
		usePath := strings.Replace(path, baseFolder, "", 1)
		if usePath != "" {
			files = append(files, usePath)
		}
		return nil
	})
	require.NoError(t, err, "Error walking results directory")

	require.Equal(t, strings.Join([]string{
		fmt.Sprintf("/%s", ipfs.DownloadShardsFolderName),
		fmt.Sprintf("/%s/0", ipfs.DownloadShardsFolderName),
		fmt.Sprintf("/%s/0/node_%s_exitCode", ipfs.DownloadShardsFolderName, system.GetShortID(hostID)),
		fmt.Sprintf("/%s/0/node_%s_stderr", ipfs.DownloadShardsFolderName, system.GetShortID(hostID)),
		fmt.Sprintf("/%s/0/node_%s_stdout", ipfs.DownloadShardsFolderName, system.GetShortID(hostID)),
		fmt.Sprintf("/stderr"),
		fmt.Sprintf("/stdout"),
		fmt.Sprintf("/%s", ipfs.DownloadVolumesFolderName),
		fmt.Sprintf("/%s/outputs", ipfs.DownloadVolumesFolderName),
	}, ","), strings.Join(files, ","), "The discovered results output structure was not correct")
}

func setupTempWorkingDir(t *testing.T) (string, func()) {
	// switch wd to a temp dir so we are not writing folders to the current directory
	// (the point of this test is to see what happens when we DONT pass --output-dir)
	tempDir, err := os.MkdirTemp("", "docker-run-download-test")
	require.NoError(t, err)
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	return tempDir, func() {
		os.Chdir(originalWd)
	}
}

// this tests that when we do docker run with no --output-dir
// it makes it's own folder to put the results in and does not splat results
// all over the current directory
func (s *GetSuite) TestDockerRunWriteToJobFolderAutoDownload() {
	ctx := context.Background()
	stack, _ := devstack_tests.SetupTest(ctx, s.T(), 1, 0, false, computenode.ComputeNodeConfig{})
	*ODR = *NewDockerRunOptions()

	swarmAddresses, err := stack.Nodes[0].IPFSClient.SwarmAddresses(ctx)
	require.NoError(s.T(), err)

	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "docker", "run",
		"--api-host", stack.Nodes[0].APIServer.Host,
		"--api-port", fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		"--wait",
		"--download",
		"ubuntu",
		"--",
		"echo", "hello from docker submit wait",
	)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)
	hostID := stack.Nodes[0].HostID

	testResultsFolderStructure(s.T(), filepath.Join(tempDir, getDefaultJobFolder(jobID)), hostID)
}

// this tests that when we do docker run with an --output-dir
// the results layout adheres to the expected folder layout
func (s *GetSuite) TestDockerRunWriteToJobFolderNamedDownload() {
	ctx := context.Background()
	stack, _ := devstack_tests.SetupTest(ctx, s.T(), 1, 0, false, computenode.ComputeNodeConfig{})
	*ODR = *NewDockerRunOptions()

	swarmAddresses, err := stack.Nodes[0].IPFSClient.SwarmAddresses(ctx)
	require.NoError(s.T(), err)

	tempDir, err := os.MkdirTemp("", "docker-run-download-test")
	require.NoError(s.T(), err)

	_, _, err = ExecuteTestCobraCommand(s.T(), s.rootCmd, "docker", "run",
		"--api-host", stack.Nodes[0].APIServer.Host,
		"--api-port", fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		"--wait",
		"--download",
		"--output-dir", tempDir,
		"ubuntu",
		"--",
		"echo", "hello from docker submit wait",
	)
	require.NoError(s.T(), err, "Error submitting job")
	hostID := stack.Nodes[0].HostID
	testResultsFolderStructure(s.T(), tempDir, hostID)
}

// this tests that when we do get with no --output-dir
// it makes it's own folder to put the results in and does not splat results
// all over the current directory
func (s *GetSuite) TestGetWriteToJobFolderAutoDownload() {
	ctx := context.Background()
	stack, _ := devstack_tests.SetupTest(ctx, s.T(), 1, 0, false, computenode.ComputeNodeConfig{})
	*ODR = *NewDockerRunOptions()
	*OG = *NewGetOptions()

	swarmAddresses, err := stack.Nodes[0].IPFSClient.SwarmAddresses(ctx)
	require.NoError(s.T(), err)

	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "docker", "run",
		"--api-host", stack.Nodes[0].APIServer.Host,
		"--api-port", fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		"--wait",
		"ubuntu",
		"--",
		"echo", "hello from docker submit wait",
	)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)
	hostID := stack.Nodes[0].HostID

	_, _, err = ExecuteTestCobraCommand(s.T(), s.rootCmd, "get",
		"--api-host", stack.Nodes[0].APIServer.Host,
		"--api-port", fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		jobID,
	)
	require.NoError(s.T(), err, "Error getting results")

	testResultsFolderStructure(s.T(), filepath.Join(tempDir, getDefaultJobFolder(jobID)), hostID)
}

// this tests that when we do get with an --output-dir
// the results layout adheres to the expected folder layout
func (s *GetSuite) TestGetWriteToJobFolderNamedDownload() {
	ctx := context.Background()
	stack, _ := devstack_tests.SetupTest(ctx, s.T(), 1, 0, false, computenode.ComputeNodeConfig{})
	*ODR = *NewDockerRunOptions()
	*OG = *NewGetOptions()

	swarmAddresses, err := stack.Nodes[0].IPFSClient.SwarmAddresses(ctx)
	require.NoError(s.T(), err)

	tempDir, err := os.MkdirTemp("", "docker-run-download-test")
	require.NoError(s.T(), err)

	_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "docker", "run",
		"--api-host", stack.Nodes[0].APIServer.Host,
		"--api-port", fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		"--wait",
		"ubuntu",
		"--",
		"echo", "hello from docker submit wait",
	)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)
	hostID := stack.Nodes[0].HostID

	_, _, err = ExecuteTestCobraCommand(s.T(), s.rootCmd, "get",
		"--api-host", stack.Nodes[0].APIServer.Host,
		"--api-port", fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		"--output-dir", tempDir,
		jobID,
	)
	require.NoError(s.T(), err, "Error getting results")

	testResultsFolderStructure(s.T(), tempDir, hostID)
}
