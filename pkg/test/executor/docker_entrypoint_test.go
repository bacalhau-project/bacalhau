//go:build integration || !unit

package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	dockmodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/bacalhau-project/bacalhau/pkg/util/targzip"
)

//When the entrypoint flag is used on the CLI, eg "docker run --entrypoint /bin/echo image hello world" docker will ignore the CMD
//Stored in the dockerfile. Any parameters used after the image will be interpreted as the new CMD.
//If no entrypoint is specified in the dockerfile, and the CMD appended to the CLI does not contain an executable that can
//be found in the chosen image's $PATH, the docker daemon will throw an error.
//Please note that if a dockerfile specifies neither the CMD or ENTRYPOINT then docker will use the base image's (if specified).
//Alpine's base image has a CMD of '/bin/sh' and no entrypoint.
//Dockerfiles in this test use the JSON array syntax, or 'exec format', i.e. CMD ["/bin/echo", "hello world"]

func TestDockerEntrypointTestSuite(t *testing.T) {
	suite.Run(t, new(DockerEntrypointTestSuite))
}

type DockerEntrypointTestSuite struct {
	suite.Suite
	testName   string
	imageNames []string
}

func (suite *DockerEntrypointTestSuite) SetupSuite() {
	docker.MustHaveDocker(suite.T())
	ctx := context.Background()
	suite.testName = strings.ToLower(suite.Suite.T().Name())

	tempDir := suite.T().TempDir()
	dockerfilePermutations := []dockerfilePermutation{
		{true, true},
		{true, false},
		{false, true},
		{false, false},
	}

	for _, permutation := range dockerfilePermutations {
		//create dockerfile
		dockerFile := createDockerfile(permutation)
		createUniqueName := func(name string) string {
			return fmt.Sprintf("%s-%s-%s-%s", suite.testName, name,
				strconv.FormatBool(permutation.entrypoint),
				strconv.FormatBool(permutation.cmd))
		}
		name := createUniqueName("Dockerfile")
		filename := filepath.Join(tempDir, name)
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, util.OS_ALL_RWX)
		require.NoError(suite.T(), err, "creating temp file")
		_, err = f.WriteString(dockerFile)
		require.NoError(suite.T(), err, "writing string to file")
		data, err := os.ReadFile(filename)
		require.NoError(suite.T(), err, "reading file")
		suite.T().Log(string(data))

		// TODO our tests will leak files if this statement isn't reached
		f.Close()
		//build image
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		require.NoError(suite.T(), err, "Creating client")
		buildContext := &bytes.Buffer{}
		err = targzip.Compress(ctx, tempDir, buildContext)
		require.NoError(suite.T(), err, "Creating build context")
		tag := createUniqueName("image")
		buildOptions := types.ImageBuildOptions{
			Dockerfile: filepath.Join(tempDir, name),
			Tags:       []string{tag + ":latest"},
		}

		response, err := cli.ImageBuild(ctx, buildContext, buildOptions)
		require.NoError(suite.T(), err, "Error building image: ")
		suite.imageNames = append(suite.imageNames, tag)
		output, err := io.ReadAll(response.Body)
		require.NoError(suite.T(), err, "building image")
		suite.T().Logf("Image %q built successfully: %s", tag, string(output))
		defer func() { _ = response.Body.Close() }()
	}
}

func (suite *DockerEntrypointTestSuite) TearDownSuite() {
	// delete the test docker images here!
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(suite.T(), err)

	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	require.NoError(suite.T(), err, "Error listing containers")

	for _, tag := range suite.imageNames {
		for _, runningContainer := range containers {
			if strings.Contains(runningContainer.Image, tag) {

				err := cli.ContainerStop(ctx, runningContainer.ID, container.StopOptions{})
				require.NoError(suite.T(), err, fmt.Sprintf("Error stopping container %q", runningContainer.ID))

				err = cli.ContainerRemove(ctx, runningContainer.ID, container.RemoveOptions{})
				require.NoError(suite.T(), err, fmt.Sprintf("Error removing container %q", runningContainer.ID))
			}
		}

		_, err = cli.ImageRemove(ctx, tag+":latest", image.RemoveOptions{Force: true})
		require.NoError(suite.T(), err, "Error removing image")
	}

}

func (suite *DockerEntrypointTestSuite) TestTableDriven() {
	var (
		overwriteEntrypoint = []string{"ls"}
		overwriteCmd        = []string{"var"}
	)

	testCases := []struct {
		name            string
		imageSuffix     string
		expectedStderr  string
		expectedStdout  string
		entrypoint      []string
		parameters      []string
		expectError     bool
		expectNoResults bool
	}{
		// Test cases for true-true
		{
			name:           "TrueTrue - Override both entrypoint and cmd",
			imageSuffix:    "true-true",
			expectedStdout: "spool\nwww\n",
			entrypoint:     overwriteEntrypoint,
			parameters:     overwriteCmd,
		},
		{
			name:           "TrueTrue - Override only cmd",
			imageSuffix:    "true-true",
			expectedStdout: "var\n",
			parameters:     overwriteCmd,
		},
		{
			name:           "TrueTrue - Override only entrypoint",
			imageSuffix:    "true-true",
			expectedStdout: "bin\ndev\netc\nhome\nlib\nlib64\nproc\nroot\nsys\ntmp\nusr\nvar\n",
			entrypoint:     overwriteEntrypoint,
		},
		{
			name:           "TrueTrue - Do not override",
			imageSuffix:    "true-true",
			expectedStdout: "echo This is from CMD\n",
		},
		// Test cases for true-false
		{
			name:           "TrueFalse - Override only cmd",
			imageSuffix:    "true-false",
			expectedStdout: "var\n",
			parameters:     overwriteCmd,
		},
		{
			name:           "TrueFalse - Override both entrypoint and cmd",
			imageSuffix:    "true-false",
			expectedStdout: "spool\nwww\n",
			entrypoint:     overwriteEntrypoint,
			parameters:     overwriteCmd,
		},
		{
			name:           "TrueFalse - Override only entrypoint",
			imageSuffix:    "true-false",
			expectedStdout: "bin\ndev\netc\nhome\nlib\nlib64\nproc\nroot\nsys\ntmp\nusr\nvar\n",
			entrypoint:     overwriteEntrypoint,
		},
		{
			name:           "TrueFalse - Do not override",
			imageSuffix:    "true-false",
			expectedStdout: "\n",
		},
		// Test cases for false-true
		{
			name:            "FalseTrue - Override only cmd",
			imageSuffix:     "false-true",
			parameters:      overwriteCmd,
			expectError:     false,
			expectNoResults: true,
		},
		{
			name:           "FalseTrue - Override only entrypoint",
			imageSuffix:    "false-true",
			expectedStdout: "bin\ndev\netc\nhome\nlib\nlib64\nproc\nroot\nsys\ntmp\nusr\nvar\n",
			entrypoint:     overwriteEntrypoint,
		},
		{
			name:           "FalseTrue - Override both entrypoint and cmd",
			imageSuffix:    "false-true",
			expectedStdout: "spool\nwww\n",
			entrypoint:     overwriteEntrypoint,
			parameters:     overwriteCmd,
		},
		{
			name:           "FalseTrue - Do not override",
			imageSuffix:    "false-true",
			expectedStdout: "This is from CMD\n",
		},
		// Test cases for false-false
		{
			name:            "FalseFalse - Override only cmd",
			imageSuffix:     "false-false",
			parameters:      overwriteCmd,
			expectError:     false,
			expectNoResults: true,
		},
		{
			name:           "FalseFalse - Override only entrypoint",
			imageSuffix:    "false-false",
			expectedStdout: "bin\ndev\netc\nhome\nlib\nlib64\nproc\nroot\nsys\ntmp\nusr\nvar\n",
			entrypoint:     overwriteEntrypoint,
		},
		{
			name:           "FalseFalse - Override both entrypoint and cmd",
			imageSuffix:    "false-false",
			expectedStdout: "spool\nwww\n",
			entrypoint:     overwriteEntrypoint,
			parameters:     overwriteCmd,
		},
		{
			name:        "FalseFalse - Do not override",
			imageSuffix: "false-false",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			image := fmt.Sprintf("%s-image-%s", suite.testName, tc.imageSuffix)
			testScenario := createTestScenario(t, tc.expectedStderr, tc.expectedStdout, image, tc.entrypoint, tc.parameters, tc.expectError, tc.expectNoResults)
			RunTestCase(t, testScenario)
		})
	}
}

func createTestScenario(t testing.TB, expectedStderr, expectedStdout, image string, entrypoint, parameters []string, expectError bool, expectNoResults bool) scenario.Scenario {
	var checkResults scenario.CheckResults
	if expectNoResults {
		checkResults = nil
	} else {
		checkResults = scenario.ManyChecks(
			scenario.FileEquals(downloader.DownloadFilenameStderr, expectedStderr),
			scenario.FileEquals(downloader.DownloadFilenameStdout, expectedStdout),
		)
	}
	testScenario := scenario.Scenario{
		ResultsChecker: checkResults,
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Engine: dockmodels.NewDockerEngineBuilder(image).
						WithEntrypoint(entrypoint...).
						WithParameters(parameters...).
						MustBuild(),
				},
			},
		},
		SubmitChecker: scenario.SubmitJobSuccess(),
	}
	if expectError == true {
		testScenario.SubmitChecker = scenario.SubmitJobErrorContains(`"var": executable file not found in $PATH`)
		testScenario.ResultsChecker = scenario.ManyChecks()
	}
	return testScenario
}

type dockerfilePermutation struct {
	entrypoint bool
	cmd        bool
}

func createDockerfile(d dockerfilePermutation) string {
	ep := "\nENTRYPOINT [\"/bin/echo\"]"
	cmd := "\nCMD [\"echo\", \"This is from CMD\"]"
	baseDockerFile := "FROM busybox:1.37.0"
	if d.entrypoint == true {
		baseDockerFile += ep
	}
	if d.cmd == true {
		baseDockerFile += cmd
	}
	return baseDockerFile
}
