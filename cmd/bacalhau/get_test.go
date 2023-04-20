//go:build integration || !unit

package bacalhau

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/model"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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
	BaseSuite
}

// Before each test
func (s *GetSuite) SetupTest() {
	docker.MustHaveDocker(s.T())
	s.BaseSuite.SetupTest()
}

func testResultsFolderStructure(t *testing.T, baseFolder, hostID string, expectedFiles []string) {
	var files []string
	err := filepath.Walk(baseFolder, func(path string, _ os.FileInfo, _ error) error {
		usePath := strings.Replace(path, baseFolder, "", 1)
		if usePath != "" {
			files = append(files, usePath)
		}
		return nil
	})
	require.NoError(t, err, "Error walking results directory")

	var expected []string
	if expectedFiles != nil {
		expected = expectedFiles
	} else {
		// Default folder structure if nothing was provided
		expected = []string{
			"/data",
			"/data/apples",
			"/data/apples/file.txt",
			"/data/file.txt",
			"/exitCode",
			"/outputs",
			"/" + model.DownloadFilenameStderr,
			"/" + model.DownloadFilenameStdout,
		}
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
		outputDir,
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
		assert.NoError(t, os.Chdir(originalWd))
	}
}

func (s *GetSuite) getDockerRunArgs(extraArgs []string) []string {
	swarmAddresses, err := s.node.IPFSClient.SwarmAddresses(context.Background())
	require.NoError(s.T(), err)
	args := []string{
		"docker", "run",
		"--api-host", s.host,
		"--api-port", fmt.Sprint(s.port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		"-o", "data:/data",
		"--wait",
	}
	args = append(args, extraArgs...)
	args = append(args,
		"ubuntu:kinetic",
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
	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	args := s.getDockerRunArgs([]string{
		"--wait",
		"--download",
	})
	_, runOutput, err := ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(runOutput)
	hostID := s.node.Host.ID().String()
	outputFolder := filepath.Join(tempDir, getDefaultJobFolder(jobID))
	testDownloadOutput(s.T(), runOutput, jobID, tempDir)
	testResultsFolderStructure(s.T(), outputFolder, hostID, nil)

}

// this tests that when we do docker run with an --output-dir
// the results layout adheres to the expected folder layout
func (s *GetSuite) TestDockerRunWriteToJobFolderNamedDownload() {
	tempDir, err := os.MkdirTemp("", "docker-run-download-test")
	require.NoError(s.T(), err)

	args := s.getDockerRunArgs([]string{
		"--wait",
		"--download",
		"--output-dir", tempDir,
	})
	_, runOutput, err := ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(runOutput)
	hostID := s.node.Host.ID().String()
	testDownloadOutput(s.T(), runOutput, jobID, tempDir)
	testResultsFolderStructure(s.T(), tempDir, hostID, nil)
}

// this tests that when we do get with no --output-dir
// it makes it's own folder to put the results in and does not splat results
// all over the current directory
func (s *GetSuite) TestGetWriteToJobFolderAutoDownload() {
	swarmAddresses, err := s.node.IPFSClient.SwarmAddresses(context.Background())
	require.NoError(s.T(), err)
	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	args := s.getDockerRunArgs([]string{
		"--wait",
	})
	_, out, err := ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)
	hostID := s.node.Host.ID().String()

	_, getOutput, err := ExecuteTestCobraCommand("get",
		"--api-host", s.node.APIServer.Address,
		"--api-port", fmt.Sprintf("%d", s.node.APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		jobID,
	)
	require.NoError(s.T(), err, "Error getting results")

	testDownloadOutput(s.T(), getOutput, jobID, filepath.Join(tempDir, getDefaultJobFolder(jobID)))
	testResultsFolderStructure(s.T(), filepath.Join(tempDir, getDefaultJobFolder(jobID)), hostID, nil)
}

func (s *GetSuite) TestGetSingleFileFromOutputBadChoice() {
	swarmAddresses, err := s.node.IPFSClient.SwarmAddresses(context.Background())
	require.NoError(s.T(), err)

	args := s.getDockerRunArgs([]string{
		"--wait",
	})
	_, out, err := ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)

	_, _, err = ExecuteTestCobraCommand("get",
		"--api-host", s.node.APIServer.Address,
		"--api-port", fmt.Sprintf("%d", s.node.APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		fmt.Sprintf("%s/missing", jobID),
	)
	require.Error(s.T(), err, "Error getting results")
}

func (s *GetSuite) TestGetSingleFileFromOutput() {
	swarmAddresses, err := s.node.IPFSClient.SwarmAddresses(context.Background())
	require.NoError(s.T(), err)
	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	args := s.getDockerRunArgs([]string{
		"--wait",
	})
	_, out, err := ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)
	hostID := s.node.Host.ID().String()

	_, getOutput, err := ExecuteTestCobraCommand("get",
		"--api-host", s.node.APIServer.Address,
		"--api-port", fmt.Sprintf("%d", s.node.APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		fmt.Sprintf("%s/stdout", jobID),
	)
	require.NoError(s.T(), err, "Error getting results")

	testDownloadOutput(s.T(), getOutput, jobID, filepath.Join(tempDir, getDefaultJobFolder(jobID)))
	testResultsFolderStructure(s.T(), filepath.Join(tempDir, getDefaultJobFolder(jobID)), hostID, []string{"/stdout"})
}

func (s *GetSuite) TestGetSingleNestedFileFromOutput() {
	swarmAddresses, err := s.node.IPFSClient.SwarmAddresses(context.Background())
	require.NoError(s.T(), err)
	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	args := s.getDockerRunArgs([]string{
		"--wait",
	})
	_, out, err := ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)
	hostID := s.node.Host.ID().String()

	_, getOutput, err := ExecuteTestCobraCommand("get",
		"--api-host", s.node.APIServer.Address,
		"--api-port", fmt.Sprintf("%d", s.node.APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		fmt.Sprintf("%s/data/apples/file.txt", jobID),
	)
	require.NoError(s.T(), err, "Error getting results")

	testDownloadOutput(s.T(), getOutput, jobID, filepath.Join(tempDir, getDefaultJobFolder(jobID)))
	testResultsFolderStructure(s.T(),
		filepath.Join(tempDir, getDefaultJobFolder(jobID)),
		hostID,
		[]string{
			"/data",
			"/data/apples",
			"/data/apples/file.txt",
		})
}

// this tests that when we do get with an --output-dir
// the results layout adheres to the expected folder layout
func (s *GetSuite) TestGetWriteToJobFolderNamedDownload() {
	ctx := context.Background()
	swarmAddresses, err := s.node.IPFSClient.SwarmAddresses(ctx)
	require.NoError(s.T(), err)

	tempDir, err := os.MkdirTemp("", "docker-run-download-test")
	require.NoError(s.T(), err)

	args := s.getDockerRunArgs([]string{
		"--wait",
	})
	_, out, err := ExecuteTestCobraCommand(args...)

	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutput(out)
	hostID := s.node.Host.ID().String()

	_, getOutput, err := ExecuteTestCobraCommand("get",
		"--api-host", s.node.APIServer.Address,
		"--api-port", fmt.Sprintf("%d", s.node.APIServer.Port),
		"--ipfs-swarm-addrs", strings.Join(swarmAddresses, ","),
		"--output-dir", tempDir,
		jobID,
	)
	require.NoError(s.T(), err, "Error getting results")
	testDownloadOutput(s.T(), getOutput, jobID, tempDir)
	testResultsFolderStructure(s.T(), tempDir, hostID, nil)
}
