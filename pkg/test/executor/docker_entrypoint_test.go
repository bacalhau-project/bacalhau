//go:build integration || !unit

package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// cases:
// 	1. Dockerfile specifies ENTRYPOINT, no CMD
// 	2. Dockerfile specifies no ENTRYPOINT, but CMD
// 	3. Dockerfile specifies ENTRYPOINT and CMD
// 	4. Dockerfile specifies neither
// for each of these cases:
//  1. what happens when we only set CMD
//  2. what happens when we set ENTRYPOINT and CMD
//  3. what happens when we set neither

func TestDockerEntrypointTestSuite(t *testing.T) {
	suite.Run(t, new(DockerEntrypointTestSuite))
}

type DockerEntrypointTestSuite struct {
	suite.Suite
}

type dockerfilePermutation struct {
	entrypoint bool
	cmd        bool
}

func createDockerfile(d dockerfilePermutation) string {
	//olgibbons: Use pkg template for this:
	ep := "\nENTRYPOINT [\"/bin/echo\"]"
	cmd := "\nCMD [\"echo\", \"This is from CMD\"]"
	baseDockerFile := "FROM ubuntu:latest"
	if d.entrypoint == true {
		baseDockerFile += ep
	}
	if d.cmd == true {
		baseDockerFile += cmd
	}
	return baseDockerFile
}

func (suite *DockerEntrypointTestSuite) SetupSuite() {
	docker.MustHaveDocker(suite.T())
	tempDir := suite.T().TempDir()
	dockerfilePermutations := []struct {
		entrypoint bool
		cmd        bool
	}{
		{true, true},
		{true, false},
		{false, true},
		{false, false},
	}

	for _, permutation := range dockerfilePermutations {
		//create dockerfile
		dockerFile := createDockerfile(permutation)
		name := fmt.Sprintf("Dockerfile-%s-%s",
			strconv.FormatBool(permutation.entrypoint),
			strconv.FormatBool(permutation.cmd))
		filename := filepath.Join(tempDir, name)
		f, err := os.OpenFile(filename, os.O_CREATE, 0600)
		require.NoError(suite.T(), err, "creating temp file")
		f.WriteString(dockerFile)
		//build image
		cli, err := client.NewClientWithOpts()
		ctx := context.Background()
		cli.ImageBuild()
		f.Close()
	}
	// build the test docker images here!
}

func (suite *DockerEntrypointTestSuite) TearDownSuite() {
	// delete the test docker images here!
}

func (suite *DockerEntrypointTestSuite) TestDockerfileEntryPointSetNoCmd() {
	DockerFileNoEntryPointYesCommand := scenario.Scenario{
		ResultsChecker: scenario.ManyChecks(
			scenario.FileEquals(model.DownloadFilenameStderr, ""),
			scenario.FileEquals(model.DownloadFilenameStdout, "I am overriding the ubuntu image's default command (bash)\n"),
		),
		Spec: model.Spec{
			Engine: model.EngineDocker,
			Docker: model.JobSpecDocker{
				Image:      "ubuntu:latest",
				Entrypoint: nil,
				Parameters: []string{"echo", "I am overriding the ubuntu image's default command (bash)"},
			},
		}, // Docker will always run ENTRYPOINT + CMD. If entrypoint = [], this is the same as just running CMD
	}

	RunTestCase(suite.T(), DockerFileNoEntryPointYesCommand)
}

//in cmd/docker_run_test.go: make sure flag is hooked up correctly
