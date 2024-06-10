//go:build integration || !unit

package get_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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
	cmdtesting.BaseSuite
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
			"/" + downloader.DownloadFilenameStderr,
			"/" + downloader.DownloadFilenameStdout,
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
	// (the point of this test is to see what happens when we DON'T pass --output-dir)
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
	args := []string{
		"docker", "run",
		"--publisher", "local",
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
	s.T().Skip("--download not supported in v2")
	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	args := s.getDockerRunArgs([]string{
		"--wait",
		"--download",
	})
	_, runOutput, err := s.ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")

	jobID := system.FindJobIDInTestOutputLegacy(runOutput)
	hostID := s.Node.ID
	outputFolder := filepath.Join(tempDir, util.GetDefaultJobFolder(jobID))
	testDownloadOutput(s.T(), runOutput, jobID, tempDir)
	testResultsFolderStructure(s.T(), outputFolder, hostID, nil)

}

// this tests that when we do docker run with an --output-dir
// the results layout adheres to the expected folder layout
func (s *GetSuite) TestDockerRunWriteToJobFolderNamedDownload() {
	s.T().Skip("--download not supported in v2")
	tempDir, err := os.MkdirTemp("", "docker-run-download-test")
	require.NoError(s.T(), err)

	args := s.getDockerRunArgs([]string{
		"--wait",
		"--download",
		"--output-dir", tempDir,
	})
	_, runOutput, err := s.ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutputLegacy(runOutput)
	hostID := s.Node.ID
	testDownloadOutput(s.T(), runOutput, jobID, tempDir)
	testResultsFolderStructure(s.T(), tempDir, hostID, nil)
}

// this tests that when we do get with no --output-dir
// it makes it's own folder to put the results in and does not splat results
// all over the current directory
func (s *GetSuite) TestGetWriteToJobFolderAutoDownload() {
	s.T().Skip("--download not supported in v2")
	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	args := s.getDockerRunArgs([]string{
		"--wait",
		"--publisher", "local",
	})
	_, out, err := s.ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutputLegacy(out)
	hostID := s.Node.ID

	_, getOutput, err := s.ExecuteTestCobraCommand("get",
		jobID,
	)
	require.NoError(s.T(), err, "Error getting results")

	testDownloadOutput(s.T(), getOutput, jobID, filepath.Join(tempDir, util.GetDefaultJobFolder(jobID)))
	testResultsFolderStructure(s.T(), filepath.Join(tempDir, util.GetDefaultJobFolder(jobID)), hostID, nil)
}

func (s *GetSuite) TestGetSingleFileFromOutputBadChoice() {
	testutils.MustHaveIPFS(s.T(), s.Config.Node.IPFS.Connect)
	args := s.getDockerRunArgs([]string{
		"--wait",
		"--publisher", "ipfs",
	})
	_, out, err := s.ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutputLegacy(out)

	_, getoutput, err := s.ExecuteTestCobraCommand("get",
		"--ipfs-connect", s.Config.Node.IPFS.Connect,
		fmt.Sprintf("%s/missing", jobID),
	)

	require.Error(s.T(), err, "expected error but it wasn't returned")
	require.Contains(s.T(), getoutput, "error downloading job")
}

func (s *GetSuite) TestGetSingleFileFromOutput() {
	testutils.MustHaveIPFS(s.T(), s.Config.Node.IPFS.Connect)
	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	args := s.getDockerRunArgs([]string{
		"--wait",
		"--publisher", "ipfs",
	})
	_, out, err := s.ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutputLegacy(out)
	hostID := s.Node.ID

	_, getOutput, err := s.ExecuteTestCobraCommand("get",
		"--ipfs-connect", s.Config.Node.IPFS.Connect,
		fmt.Sprintf("%s/stdout", jobID),
	)
	require.NoError(s.T(), err, "Error getting results")

	testDownloadOutput(s.T(), getOutput, jobID, filepath.Join(tempDir, util.GetDefaultJobFolder(jobID)))
	testResultsFolderStructure(s.T(), filepath.Join(tempDir, util.GetDefaultJobFolder(jobID)), hostID, []string{"/stdout"})
}

func (s *GetSuite) TestGetSingleNestedFileFromOutput() {
	testutils.MustHaveIPFS(s.T(), s.Config.Node.IPFS.Connect)
	tempDir, cleanup := setupTempWorkingDir(s.T())
	defer cleanup()

	args := s.getDockerRunArgs([]string{
		"--wait",
		"--publisher", "ipfs",
	})
	_, out, err := s.ExecuteTestCobraCommand(args...)
	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutputLegacy(out)
	hostID := s.Node.ID

	_, getOutput, err := s.ExecuteTestCobraCommand("get",
		"--ipfs-connect", s.Config.Node.IPFS.Connect,
		"--api-host", s.Node.APIServer.Address,
		"--api-port", fmt.Sprintf("%d", s.Node.APIServer.Port),
		fmt.Sprintf("%s/data/apples/file.txt", jobID),
	)
	require.NoError(s.T(), err, "Error getting results")

	testDownloadOutput(s.T(), getOutput, jobID, filepath.Join(tempDir, util.GetDefaultJobFolder(jobID)))
	testResultsFolderStructure(s.T(),
		filepath.Join(tempDir, util.GetDefaultJobFolder(jobID)),
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
	tempDir, err := os.MkdirTemp("", "docker-run-download-test")
	require.NoError(s.T(), err)

	args := s.getDockerRunArgs([]string{
		"--wait",
		"-o outputs:/outputs",
		"--publisher", "local",
	})
	_, out, err := s.ExecuteTestCobraCommand(args...)

	require.NoError(s.T(), err, "Error submitting job")
	jobID := system.FindJobIDInTestOutputLegacy(out)
	hostID := s.Node.ID

	_, getOutput, err := s.ExecuteTestCobraCommand("get",
		"--output-dir", tempDir,
		jobID,
	)
	require.NoError(s.T(), err, "Error getting results")
	testDownloadOutput(s.T(), getOutput, jobID, tempDir)
	testResultsFolderStructure(s.T(), tempDir, hostID, nil)
}
