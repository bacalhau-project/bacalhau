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

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/bacalhau-project/bacalhau/pkg/util/targzip"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//When the entrypoint flag is used on the CLI, eg "docker run --entrypoint /bin/echo image hello world" docker will ignore the CMD
//Stored in the dockerfile. Any paramaters used after the image will be interpreted as the new CMD.
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

type dockerfilePermutation struct {
	entrypoint bool
	cmd        bool
}

type userOverwrites struct {
	entrypoint []string
	cmd        []string
}

var defaultUserOverwrites = userOverwrites{
	entrypoint: []string{"/bin/ls"},
	cmd:        []string{"media"},
}

func createDockerfile(d dockerfilePermutation) string {
	ep := "\nENTRYPOINT [\"/bin/echo\"]"
	cmd := "\nCMD [\"echo\", \"This is from CMD\"]"
	baseDockerFile := "FROM alpine:latest"
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
	ctx := context.Background()
	suite.testName = strings.ToLower(suite.Suite.T().Name())

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
		defer response.Body.Close()
	}
}

func (suite *DockerEntrypointTestSuite) TearDownSuite() {
	// delete the test docker images here!
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(suite.T(), err)

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	require.NoError(suite.T(), err, "Error listing containers")

	for _, tag := range suite.imageNames {
		for _, runningContainer := range containers {
			if strings.Contains(runningContainer.Image, tag) {

				err := cli.ContainerStop(ctx, runningContainer.ID, container.StopOptions{})
				require.NoError(suite.T(), err, fmt.Sprintf("Error stopping container %q", runningContainer.ID))

				err = cli.ContainerRemove(ctx, runningContainer.ID, types.ContainerRemoveOptions{})
				require.NoError(suite.T(), err, fmt.Sprintf("Error removing container %q", runningContainer.ID))
			}
		}

		_, err = cli.ImageRemove(ctx, tag+":latest", types.ImageRemoveOptions{Force: true})
		require.NoError(suite.T(), err, "Error removing image")
	}

}

type tests []struct {
	expectedStderr string
	expectedStdout string
	image          string
	entrypoint     []string
	parameters     []string
	expectError    bool
}

func createTestScenario(expectedStderr, expectedStdout, image string, entrypoint, parameters []string, expectError bool) scenario.Scenario {
	testScenario := scenario.Scenario{
		ResultsChecker: scenario.ManyChecks(
			scenario.FileEquals(model.DownloadFilenameStderr, expectedStderr),
			scenario.FileEquals(model.DownloadFilenameStdout, expectedStdout),
		),
		Spec: model.Spec{
			Engine: model.EngineDocker,
			Docker: model.JobSpecDocker{
				Image:      image,
				Entrypoint: entrypoint,
				Parameters: parameters,
			},
		},
		SubmitChecker: scenario.SubmitJobSuccess(),
	}
	if expectError == true {
		testScenario.SubmitChecker = scenario.SubmitJobErrorContains(`"media": executable file not found in $PATH`)
		testScenario.ResultsChecker = scenario.ManyChecks(nil)
	}
	return testScenario
}
func (suite *DockerEntrypointTestSuite) TestCaseImageTrueTrue() {
	//CMD is set AND Entrypoint is set:
	//Entrypoint ["bin/echo"]
	//Cmd ["echo","This is from CMD"]
	image := suite.testName + "-image-true-true"
	stderr := ""
	newTests := tests{

		{stderr, "cdrom\nfloppy\nusb\n", image, defaultUserOverwrites.entrypoint, defaultUserOverwrites.cmd, false},
		{stderr, "media\n", image, nil, defaultUserOverwrites.cmd, false},
		{stderr, "bin\ndev\netc\nhome\nlib\nmedia\nmnt\nopt\nproc\nroot\nrun\nsbin\nsrv\nsys\ntmp\nusr\nvar\n",
			image, defaultUserOverwrites.entrypoint, nil, false},
		{stderr, "echo This is from CMD\n", image, nil, nil, false},
	}
	for _, test := range newTests {
		testScenario := createTestScenario(test.expectedStderr, test.expectedStdout, test.image, test.entrypoint, test.parameters, test.expectError)
		RunTestCase(suite.T(), testScenario)
	}

}

func (suite *DockerEntrypointTestSuite) TestCaseImageTrueFalse() {
	//Entrypoint is set to bin/echo
	//CMD is empty

	image := suite.testName + "-image-true-false"
	stderr := ""
	newTests := tests{
		{stderr, "media\n", image, nil, defaultUserOverwrites.cmd, false},
		{stderr, "cdrom\nfloppy\nusb\n", image, defaultUserOverwrites.entrypoint, defaultUserOverwrites.cmd, false},
		{stderr, "bin\ndev\netc\nhome\nlib\nmedia\nmnt\nopt\nproc\nroot\nrun\nsbin\nsrv\nsys\ntmp\nusr\nvar\n",
			image, defaultUserOverwrites.entrypoint, nil, false},
		{stderr, "\n", image, nil, nil, false},
	}
	for _, test := range newTests {
		testScenario := createTestScenario(test.expectedStderr, test.expectedStdout, test.image, test.entrypoint, test.parameters, test.expectError)
		RunTestCase(suite.T(), testScenario)
	}
}

func (suite *DockerEntrypointTestSuite) TestCaseImageFalseTrue() {
	//Entrypoint is empty
	//CMD is set to ["echo","This is from CMD"]
	image := suite.testName + "-image-false-true"
	stderr := ""
	newTests := tests{
		//override Cmd expect: error.
		{stderr, "", image, nil, defaultUserOverwrites.cmd, true},
		{stderr, "bin\ndev\netc\nhome\nlib\nmedia\nmnt\nopt\nproc\nroot\nrun\nsbin\nsrv\nsys\ntmp\nusr\nvar\n",
			image, defaultUserOverwrites.entrypoint, nil, false},
		{stderr, "cdrom\nfloppy\nusb\n", image, defaultUserOverwrites.entrypoint, defaultUserOverwrites.cmd, false},
		{stderr, "This is from CMD\n", image, nil, nil, false},
	}
	for _, test := range newTests {
		testScenario := createTestScenario(test.expectedStderr, test.expectedStdout, test.image, test.entrypoint, test.parameters, test.expectError)
		RunTestCase(suite.T(), testScenario)
	}

}
func (suite *DockerEntrypointTestSuite) TestCaseImageFalseFalse() {
	//Entrypoint is empty
	//CMD is empty, so dockerfile will use Alpine default of /bin/sh
	image := suite.testName + "-image-false-false"
	stderr := ""
	newTests := tests{
		//override Cmd expect: error.
		{stderr, "", image, nil, defaultUserOverwrites.cmd, true},
		{stderr, "bin\ndev\netc\nhome\nlib\nmedia\nmnt\nopt\nproc\nroot\nrun\nsbin\nsrv\nsys\ntmp\nusr\nvar\n",
			image, defaultUserOverwrites.entrypoint, nil, false},
		{stderr, "cdrom\nfloppy\nusb\n", image, defaultUserOverwrites.entrypoint, defaultUserOverwrites.cmd, false},
		{stderr, "", image, nil, nil, false},
	}
	for _, test := range newTests {
		testScenario := createTestScenario(test.expectedStderr, test.expectedStdout, test.image, test.entrypoint, test.parameters, test.expectError)
		RunTestCase(suite.T(), testScenario)
	}
}
