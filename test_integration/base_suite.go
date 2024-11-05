package test_integration

import (
	"bacalhau/integration_tests/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/compose/v2/pkg/api"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
)

type BaseDockerComposeTestSuite struct {
	suite.Suite
	ComposeStack           interface{}
	AdditionalSetupEnvVars map[string]string
	Context                context.Context
	Cancel                 context.CancelFunc
	SuiteRunIdentifier     string
	GlobalRunIdentifier    string
}

func (s *BaseDockerComposeTestSuite) SetupSuite(dockerComposeFilePath string, renderingData map[string]interface{}) {
	s.T().Log("Setting up [Test Suite] in base suite......")
	s.Context, s.Cancel = context.WithCancel(context.Background())
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])

	// Default Rendered Data
	defaultRenderingData := map[string]interface{}{
		"RegistryImageName":     fmt.Sprintf("bacalhau-test-registry-%s:%s", s.GlobalRunIdentifier, s.GlobalRunIdentifier),
		"OrchestratorImageName": fmt.Sprintf("bacalhau-test-orchestrator-%s:%s", s.GlobalRunIdentifier, s.GlobalRunIdentifier),
		"ComputeImageName":      fmt.Sprintf("bacalhau-test-compute-%s:%s", s.GlobalRunIdentifier, s.GlobalRunIdentifier),
		"JumpboxImageName":      fmt.Sprintf("bacalhau-test-jumpbox-%s:%s", s.GlobalRunIdentifier, s.GlobalRunIdentifier),
	}

	// Merge Rendering Data
	if renderingData != nil {
		for key, value := range renderingData {
			defaultRenderingData[key] = value
		}
	}

	// Render Docker Compose File
	tmpDir, err := os.MkdirTemp("", "bacalhau-test-"+s.SuiteRunIdentifier)
	s.Require().NoErrorf(err, "Error creating tmp dir: %q", err)

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			fmt.Printf("Error removing temporary directory: %v\n", err)
		}
	}()

	renderedDockerComposeFile := s.renderDockerComposeFile(dockerComposeFilePath, tmpDir, defaultRenderingData)

	// Standup docker compose stack
	s.standUpDockerComposeStack(renderedDockerComposeFile)

	// Run a basic command to init the ~/.bacalhau directory, or else the output will unfortunately
	// mess with tests results, even if just a json format is requested
	_, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "node", "list"})
	s.Require().NoErrorf(err, "Error running a basic command to init the '.bacalhau' directory: %q", err)
}

func (s *BaseDockerComposeTestSuite) TearDownSuite() {
	// Runs once after all tests
	s.T().Log("Tearing down [Test Suite] in base suite...")
	s.downWithDockerCompose()

	// Delete any docker images that were created for this specific suite, but leave out the common global images
	err := utils.DeleteDockerTestImagesAndPrune(s.SuiteRunIdentifier)
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

func (s *BaseDockerComposeTestSuite) standUpDockerComposeStack(composeFilePath string) {
	s.Require().FileExistsf(composeFilePath, "Rendered Docker compose File does not exist: %s", composeFilePath)

	// Testify project names must consist only of lowercase alphanumeric characters, hyphens, and underscores
	normalizedTestName := strings.ToLower(strings.ReplaceAll(s.T().Name(), "/", "-"))
	identifier := fmt.Sprintf("test-bacalhau-%s-%s", s.SuiteRunIdentifier, normalizedTestName)

	composeStack, err := tc.NewDockerComposeWith(
		tc.StackIdentifier(identifier),
		tc.WithStackFiles(composeFilePath),
	)
	s.Require().NoErrorf(err, "Error preparing Docker Compose Stack: %q\n", err)

	s.ComposeStack = composeStack

	err = composeStack.Up(s.Context, tc.Wait(true), tc.WithRecreate(api.RecreateForce), tc.RemoveOrphans(true))
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

func (s *BaseDockerComposeTestSuite) executeCommandInContainer(containerName string, cmd []string, execOptions ...exec.ProcessOption) (string, error) {
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

	exitCode, reader, err := container.Exec(s.Context, cmd, execOptions...)
	if err != nil {
		return "", fmt.Errorf("executing command inside container: failed to execute command: %w", err)
	}

	output, err2 := readContainerExecOutput(err, reader)
	if err2 != nil {
		return "", err2
	}

	if exitCode != 0 {
		return "", fmt.Errorf("executing command inside container: command exited with status %d: %s", exitCode, output)
	}

	return output, nil
}

func (s *BaseDockerComposeTestSuite) executeCommandInDefaultJumpbox(cmd []string, execOptions ...exec.ProcessOption) (string, error) {
	return s.executeCommandInContainer("bacalhau-jumpbox-node", cmd, execOptions...)
}

func (s *BaseDockerComposeTestSuite) waitForJobToComplete(jobID string, timeout time.Duration) (time.Duration, error) {
	startTime := time.Now()
	endTime := startTime.Add(timeout)

	for time.Now().Before(endTime) {
		jobDescriptionResultJson, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", "--output=json", jobID})
		if err != nil {
			return time.Since(startTime), err
		}

		jobState, err := utils.ExtractJobStateType(jobDescriptionResultJson)
		if err != nil {
			return time.Since(startTime), err
		}

		if jobState == "Completed" {
			return time.Since(startTime), nil
		}

		time.Sleep(time.Second)
	}

	return timeout, fmt.Errorf("job did not finish within allowed time limit %s", timeout.String())
}

func (s *BaseDockerComposeTestSuite) unmarshalJSONString(jsonString string, expectedType JSONResponseType) (interface{}, error) {
	// Cleanup the Json output. Unfortunate that the CLI prints extra
	// characters at the beginning and at the end
	cleanedJsonString := strings.TrimLeftFunc(jsonString, func(r rune) bool {
		return r != '{' && r != '['
	})
	cleanedJsonString = strings.TrimRightFunc(cleanedJsonString, func(r rune) bool {
		return r != '}' && r != ']'
	})

	var data interface{}
	err := json.Unmarshal([]byte(cleanedJsonString), &data)
	if err != nil {
		s.Require().NoErrorf(err, "Error unmarshalling json string: %q. JSON input received: %q", err, jsonString)
		return nil, err
	}

	var actualType JSONResponseType
	switch data.(type) {
	case map[string]interface{}:
		actualType = JSONObject
	case []interface{}:
		actualType = JSONArray
	default:
		s.Require().NoErrorf(err, "unexpected JSON type in Response: neither object nor list. Got: %q", jsonString)
		return nil, fmt.Errorf("unexpected JSON type in Response. Got: %q", jsonString)
	}

	if actualType != expectedType {
		s.Require().NoErrorf(err, "JSON type mismatch in: expected %s, got %s. Input String: %q", expectedType, actualType, jsonString)
		return nil, fmt.Errorf("JSON type mismatch in: expected %s, got %s", expectedType, actualType)
	}

	return data, nil
}

func (s *BaseDockerComposeTestSuite) convertStringToDynamicJSON(jsonString string) (*utils.DynamicJSON, error) {
	// Cleanup the Json output. Unfortunate that the CLI prints extra
	// characters at the beginning and at the end
	cleanedJsonString := strings.TrimLeftFunc(jsonString, func(r rune) bool {
		return r != '{' && r != '['
	})
	cleanedJsonString = strings.TrimRightFunc(cleanedJsonString, func(r rune) bool {
		return r != '}' && r != ']'
	})

	parsedDynamicJSON, err := utils.ParseToDynamicJSON(cleanedJsonString)
	if err != nil {
		return nil, err
	}

	return parsedDynamicJSON, nil
}

func (s *BaseDockerComposeTestSuite) renderDockerComposeFile(inputFilePath string, tmpDir string, imageReplacements map[string]interface{}) string {
	renderedDockerComposeFile, err := os.CreateTemp(tmpDir, "docker-compose-")
	s.Require().NoErrorf(err, "Error creating tmp file: %q", err)

	err = renderedDockerComposeFile.Close()
	s.Require().NoErrorf(err, "Error closing tmp file: %q", err)

	err = utils.ProcessYAMLTemplate(inputFilePath, renderedDockerComposeFile.Name(), imageReplacements)
	s.Require().NoErrorf(err, "Error rendering docker compose file: %q", err)

	return renderedDockerComposeFile.Name()
}

func (s *BaseDockerComposeTestSuite) commonAssets(inputFilePath string) string {
	return fmt.Sprintf("/bacalhau_integration_tests/common_assets/%s", inputFilePath)
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

// JSONResponseType Used for API response validation
type JSONResponseType string

const (
	JSONObject JSONResponseType = "JSONObject"
	JSONArray  JSONResponseType = "JSONArray"
)
