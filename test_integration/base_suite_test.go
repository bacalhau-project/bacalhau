package test_integration

import (
	"context"
	"fmt"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var globalTestExecutionId string

func TestMain(m *testing.M) {
	globalTestExecutionId = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	log.Println("====> Starting the whole test flow: ", globalTestExecutionId)

	err := setTestGlobalEnvVariables(map[string]string{})
	if err != nil {
		log.Println("Error Setting up Test Env Variables: ", err.Error())
		os.Exit(1)
	}

	ctx := context.Background()
	err = compileBacalhau(ctx, "../main.go")
	if err != nil {
		log.Println("Error compiling the bacalhau binary: ", err.Error())
		os.Exit(1)
	}

	// TODO: Maybe we do not need to created images, but just inject
	// TODO: them with artifacts before container starts the starts (certs and binary and configs)
	err = buildBaseImages(globalTestExecutionId)
	if err != nil {
		log.Println("Error building base images: ", err.Error())
		os.Exit(1)
	}

	exitCode := m.Run()

	err = deleteDockerTestImagesAndPrune(globalTestExecutionId)
	if err != nil {
		log.Println("Error cleaning up base images: ", err.Error())
		os.Exit(1)
	}

	// TODO: Better cleaning
	os.Remove("./assets/dockerfiles/bacalhau_bin")
	//Exit with the same code as the test run
	os.Exit(exitCode)
}

type BaseDockerComposeTestSuite struct {
	suite.Suite
	ComposeStack           interface{}
	AdditionalSetupEnvVars map[string]string
	Context                context.Context
	Cancel                 context.CancelFunc
	suiteRunIdentifier     string
	globalRunIdentifier    string
}

func (s *BaseDockerComposeTestSuite) SetupSuite(dockerComposeFilePath string, adhocImagesNames map[string]string) {
	s.T().Log("Setting up [Test Suite] in base suite......")

	// Merge Docker images name
	initialImagesNames := map[string]string{
		"placeholder-registry-image":  fmt.Sprintf("bacalhau-test-registry-%s:%s", s.globalRunIdentifier, s.globalRunIdentifier),
		"placeholder-requester-image": fmt.Sprintf("bacalhau-test-requester-%s:%s", s.globalRunIdentifier, s.globalRunIdentifier),
		"placeholder-compute-image":   fmt.Sprintf("bacalhau-test-compute-%s:%s", s.globalRunIdentifier, s.globalRunIdentifier),
		"placeholder-jumpbox-image":   fmt.Sprintf("bacalhau-test-jumpbox-%s:%s", s.globalRunIdentifier, s.globalRunIdentifier),
	}

	if adhocImagesNames != nil {
		for key, value := range adhocImagesNames {
			initialImagesNames[key] = value
		}
	}

	// Render Docker Compose File
	tmpDir, err := os.MkdirTemp("", "bacalhau-test-"+s.suiteRunIdentifier)
	s.Require().NoErrorf(err, "Error creating tmp dir: %q", err)

	renderedDockerComposeFilePath, err := s.renderDockerComposeFile(
		dockerComposeFilePath,
		tmpDir,
		initialImagesNames,
	)
	s.Require().NoErrorf(err, "Error rendering Docker Compose file: %q", err)

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Printf("Error removing temporary directory: %v\n", err)
		}
	}()

	s.standUpDockerComposeStack(renderedDockerComposeFilePath)
	s.copyTestsAssetsToJumpbox()
}

func (s *BaseDockerComposeTestSuite) TearDownSuite() {
	// Runs once after all tests
	s.T().Log("Tearing down [Test Suite] in base suite...")
	s.downWithDockerCompose()

	// Delete any docker images that were created for this specific suite, but leave out the common global images
	err := deleteDockerTestImagesAndPrune(s.suiteRunIdentifier)
	s.Require().NoError(err, "Failed to delete images")
}

func (s *BaseDockerComposeTestSuite) SetupTest() {
	// Runs before each test
	s.T().Log("Setting up [Test] in base suite...")
}

func (s *BaseDockerComposeTestSuite) TearDownTest() {
	// Runs after each test
	s.T().Log("Tearing down [Test] in base suite......")
}

func (s *BaseDockerComposeTestSuite) copyTestsAssetsToJumpbox() {
	s.T().Log("Copying ./assets to jumpbox container......")

	// TODO: make this name configurable
	jumpboxServiceName := "bacalhau-jumpbox-node"
	typeAssertedComposeStack, typeAssertionOk := s.ComposeStack.(interface {
		ServiceContainer(context.Context, string) (*testcontainers.DockerContainer, error)
	})
	s.Require().True(typeAssertionOk, "copying test assets: ComposeStack does not implement ServiceContainer method")

	jumpBoxContainer, err := typeAssertedComposeStack.ServiceContainer(s.Context, jumpboxServiceName)
	s.Require().NoErrorf(err, "cannot find jumpbox container: %q", err)

	testAssetsDirectory, err := filepath.Abs(filepath.Join(".", "assets/"))
	s.Require().NoErrorf(err, "cannot find assets tes directory: %q", err)

	// Now you can use CopyDirToContainer on the service container
	err = jumpBoxContainer.CopyDirToContainer(s.Context, testAssetsDirectory, "/app/", 0o755)
	s.Require().NoErrorf(err, "cannot copy tests assets to jumpbox: %q", err)
}

func (s *BaseDockerComposeTestSuite) standUpDockerComposeStack(composeFilePath string) {
	s.Require().FileExistsf(composeFilePath, "Rendered Docker compose File does not exist: %s", composeFilePath)

	// Testify project names must consist only of lowercase alphanumeric characters, hyphens, and underscores
	normalizedTestName := strings.ToLower(strings.ReplaceAll(s.T().Name(), "/", "-"))
	identifier := fmt.Sprintf("test-bacalhau-%s-%s", s.suiteRunIdentifier, normalizedTestName)

	composeStack, err := tc.NewDockerComposeWith(
		tc.StackIdentifier(identifier),
		tc.WithStackFiles(composeFilePath),
	)
	s.Require().NoErrorf(err, "Error preparing Docker Compose Stack: %q\n", err)

	s.ComposeStack = composeStack

	err = composeStack.Up(s.Context, tc.Wait(true), tc.WithRecreate(api.RecreateForce))
	s.Require().NoErrorf(err, "Error creating Docker Compose Stack: %q\n", err)
}

func (s *BaseDockerComposeTestSuite) downWithDockerCompose() {
	err := s.ComposeStack.(interface {
		Down(context.Context, ...tc.StackDownOption) error
	}).Down(
		context.Background(),
		tc.RemoveImagesLocal,
		tc.RemoveOrphans(true),
		tc.RemoveVolumes(true),
	)
	s.Require().NoErrorf(err, "Error tearing down docker compose stack: %q\n", err)
	s.Cancel()
}

func (s *BaseDockerComposeTestSuite) executeCommandInContainer(containerName string, cmd []string) (string, error) {
	typeAssertedComposeStack, ok := s.ComposeStack.(interface {
		ServiceContainer(context.Context, string) (*testcontainers.DockerContainer, error)
	})
	if !ok {
		return "", fmt.Errorf("executing command inside container: ComposeStack does not implement ServiceContainer method")
	}

	container, err := typeAssertedComposeStack.ServiceContainer(s.Context, containerName)
	if err != nil {
		return "", fmt.Errorf("executing command inside container: failed to get service container: %w", err)
	}

	exitCode, reader, err := container.Exec(s.Context, cmd)
	if err != nil {
		return "", fmt.Errorf("executing command inside container: failed to execute command: %w", err)
	}

	output, err2 := readContainerExecOutput(err, reader)
	if err2 != nil {
		return "", err2
	}

	if exitCode != 0 {
		return output, fmt.Errorf("executing command inside container: command exited with status %d: %s", exitCode, output)
	}

	return output, nil
}

func (s *BaseDockerComposeTestSuite) executeCommandInDefaultJumpbox(cmd []string) (string, error) {
	return s.executeCommandInContainer("bacalhau-jumpbox-node", cmd)
}

func (s *BaseDockerComposeTestSuite) waitForJobToComplete(jobID string, timeout time.Duration) (time.Duration, error) {
	startTime := time.Now()
	endTime := startTime.Add(timeout)

	for time.Now().Before(endTime) {
		jobDescriptionResultJson, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", "--output=json", jobID})
		if err != nil {
			return time.Since(startTime), err
		}

		jobState, err := extractJobStateType(jobDescriptionResultJson)
		if err != nil {
			return time.Since(startTime), err
		}

		if jobState == "Completed" {
			return time.Since(startTime), nil
		}

		time.Sleep(1000 * time.Millisecond)
	}

	return timeout, fmt.Errorf("job did not finish within allowed time limit %s", timeout.String())
}

func (s *BaseDockerComposeTestSuite) renderDockerComposeFile(inputFilePath string, tmpDir string, imageReplacements map[string]string) (string, error) {
	data, err := os.ReadFile(inputFilePath)
	if err != nil {
		return "", fmt.Errorf("error reading file: %v", err)
	}

	var node yaml.Node
	err = yaml.Unmarshal(data, &node)
	if err != nil {
		return "", fmt.Errorf("error parsing YAML: %v", err)
	}

	err = modifyImageNamesInDockerComposeFile(&node, imageReplacements)
	if err != nil {
		return "", fmt.Errorf("error modifying image names: %v", err)
	}

	modifiedYAML, err := yaml.Marshal(&node)
	if err != nil {
		return "", fmt.Errorf("error marshaling YAML: %v", err)
	}

	tmpFile := filepath.Join(tmpDir, "rendered-docker-compose.yml")
	err = os.WriteFile(tmpFile, modifiedYAML, 0644)
	if err != nil {
		return "", fmt.Errorf("error writing file: %v", err)
	}

	return tmpFile, nil
}

func readContainerExecOutput(err error, reader io.Reader) (string, error) {
	var buf strings.Builder
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return "", fmt.Errorf(
			"failed to read command output: %w",
			err,
		)
	}

	output := buf.String()
	return output, nil
}
