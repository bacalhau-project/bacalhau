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
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/bacalhau-project/bacalhau/pkg/util/targzip"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//When the entrypoint flag is used on the CLI, eg "docker run --entrypoint /bin/echo image hello world" docker will ignore the CMD
//Stored in the dockerfile. Any paramaters used after the image will be interpreted as the new CMD.
//If no entrypoint is specified in the dockerfile and the CMD appended to the CLI does not contain an executable that can
//be found in Alpine's $PATH, the docker daemon will throw an error.
//Please note that if a dockerfile specifies neither the CMD or ENTRYPOINT then docker will use the base image's (if specified).
//Alpine's base image has a CMD of '/bin/sh' and no entrypoint.
//Dockerfiles in this test use the JSON array syntax, or 'exec format', i.e. CMD ["/bin/echo", "hello world"]

// cases:
// 	1. Dockerfile specifies ENTRYPOINT, no CMD (error with Alpine)
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

type userOverwrites struct {
	entrypoint []string
	cmd        []string
}

var defaultUserOverwrites = userOverwrites{
	entrypoint: []string{"/bin/ls"},
	cmd:        []string{"media"},
}

func createDockerfile(d dockerfilePermutation) string {
	//olgibbons: Use pkg template for this:
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
			return fmt.Sprintf("%s-%s-%s", name,
				strconv.FormatBool(permutation.entrypoint),
				strconv.FormatBool(permutation.cmd))
		}
		name := createUniqueName("Dockerfile")
		filename := filepath.Join(tempDir, name)
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, util.OS_ALL_RWX)
		require.NoError(suite.T(), err, "creating temp file")
		_, err = f.WriteString(dockerFile)
		require.NoError(suite.T(), err, "writing string to file")
		//delete following line after testing olgibbons
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
		output, err := io.ReadAll(response.Body)
		require.NoError(suite.T(), err, "building image")
		suite.T().Logf("Image %q built successfully: %s", tag, string(output))
		defer response.Body.Close()
	}
}

func (suite *DockerEntrypointTestSuite) TearDownSuite() {
	// delete the test docker images here!
}

type tests []struct {
	expectedStderr string
	expectedStdout string
	image          string
	entrypoint     []string
	parameters     []string
}

func createTestScenario(expectedStderr, expectedStdout, image string, entrypoint, parameters []string) scenario.Scenario {
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
	}
	return testScenario
}
func (suite *DockerEntrypointTestSuite) TestCaseImageTrueTrue() {
	//CMD is set AND Entrypoint is set:
	//Entrypoint ["bin/echo"]
	//Cmd ["echo","This is from CMD"]
	image := "image-true-true"
	stderr := ""
	newTests := tests{

		{stderr, "cdrom\nfloppy\nusb\n", image, defaultUserOverwrites.entrypoint, defaultUserOverwrites.cmd},
		{stderr, "media\n", image, nil, defaultUserOverwrites.cmd},
		{stderr, "bin\ndev\netc\nhome\nlib\nmedia\nmnt\nopt\nproc\nroot\nrun\nsbin\nsrv\nsys\ntmp\nusr\nvar\n",
			image, defaultUserOverwrites.entrypoint, nil},
		{stderr, "echo This is from CMD\n", image, nil, nil},
	}
	for _, test := range newTests {
		testScenario := createTestScenario(test.expectedStderr, test.expectedStdout, test.image, test.entrypoint, test.parameters)
		RunTestCase(suite.T(), testScenario)
	}

}

func (suite *DockerEntrypointTestSuite) TestCaseImageTrueFalse() {
	//Entrypoint is set to bin/echo
	//CMD is empty

	image := "image-true-false"
	stderr := ""
	newTests := tests{
		{stderr, "media\n", image, nil, defaultUserOverwrites.cmd},
		{stderr, "cdrom\nfloppy\nusb\n", image, defaultUserOverwrites.entrypoint, defaultUserOverwrites.cmd},
		{stderr, "bin\ndev\netc\nhome\nlib\nmedia\nmnt\nopt\nproc\nroot\nrun\nsbin\nsrv\nsys\ntmp\nusr\nvar\n",
			image, defaultUserOverwrites.entrypoint, nil},
		{stderr, "\n", image, nil, nil},
	}
	for _, test := range newTests {
		testScenario := createTestScenario(test.expectedStderr, test.expectedStdout, test.image, test.entrypoint, test.parameters)
		RunTestCase(suite.T(), testScenario)
	}
}

func (suite *DockerEntrypointTestSuite) TestCaseImageFalseTrue() {
	//Entrypoint is empty
	//CMD is set to ["echo","This is from CMD"]
	image := "image-false-true"
	stderr := ""
	newTests := tests{
		//override Cmd expect: error.
		{stderr, "", image, nil, defaultUserOverwrites.cmd},
		//override Entrypoint to /bin/ls should throw an error
		{stderr, "bin\ndev\netc\nhome\nlib\nmedia\nmnt\nopt\nproc\nroot\nrun\nsbin\nsrv\nsys\ntmp\nusr\nvar\n",
			image, defaultUserOverwrites.entrypoint, nil},
		{stderr, "cdrom\nfloppy\nusb\n", image, defaultUserOverwrites.entrypoint, defaultUserOverwrites.cmd},
		{stderr, "This is from CMD\n", image, nil, nil},
	}
	for _, test := range newTests {
		testScenario := createTestScenario(test.expectedStderr, test.expectedStdout, test.image, test.entrypoint, test.parameters)
		RunTestCase(suite.T(), testScenario)
	}

}
func (suite *DockerEntrypointTestSuite) TestCaseImageFalseFalse() {
	//Entrypoint is empty
	//CMD is empty, so dockerfile will use Alpine default of /bin/sh
	image := "image-false-false"
	stderr := ""
	newTests := tests{
		//override Cmd expect: error.
		{stderr, "", image, nil, defaultUserOverwrites.cmd},
		{stderr, "bin\ndev\netc\nhome\nlib\nmedia\nmnt\nopt\nproc\nroot\nrun\nsbin\nsrv\nsys\ntmp\nusr\nvar\n",
			image, defaultUserOverwrites.entrypoint, nil},
		{stderr, "cdrom\nfloppy\nusb\n", image, defaultUserOverwrites.entrypoint, defaultUserOverwrites.cmd},
		{stderr, "", image, nil, nil},
	}
	for _, test := range newTests {
		testScenario := createTestScenario(test.expectedStderr, test.expectedStdout, test.image, test.entrypoint, test.parameters)
		RunTestCase(suite.T(), testScenario)
	}
}

func (suite *DockerEntrypointTestSuite) TestDeleteMeAfterOlGibbons() {
	DockerfileOverWriteCmdAndEntrypoint := createTestScenario(
		"",
		"cdrom\nfloppy\nusb\n",
		"image-true-true",
		defaultUserOverwrites.entrypoint,
		defaultUserOverwrites.cmd,
	)
	RunTestCase(suite.T(), DockerfileOverWriteCmdAndEntrypoint)
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
