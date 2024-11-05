package test_integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"bacalhau/integration_tests/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type JobsBasicRunsScenariosSuite struct {
	BaseDockerComposeTestSuite
}

func NewJobsBasicRunsScenariosTestSuite() *JobsBasicRunsScenariosSuite {
	s := &JobsBasicRunsScenariosSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *JobsBasicRunsScenariosSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/deployment-with-minio-and-registry.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/6_orchestrator_config.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *JobsBasicRunsScenariosSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in JobsBasicRunsScenariosSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *JobsBasicRunsScenariosSuite) TestDockerRunHelloWorld() {
	result, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "docker", "run", "hello-world"})

	s.Require().NoError(err, "Error running hello world")

	jobID := utils.ExtractJobIDFromOutput(result, &s.Suite)

	completedIn, err := s.waitForJobToComplete(jobID, 30*time.Second)
	s.Require().NoErrorf(err, "Error waiting for job to complete, waited %s: %q", completedIn, err)

	resultDescription, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobID})

	s.Require().NoError(err, "Error running job description")
	s.Require().Contains(resultDescription, "Hello from Docker", resultDescription)
}

func (s *JobsBasicRunsScenariosSuite) TestHelloWorldJob() {
	result, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"--wait=false",
			"--id-only",
			"/bacalhau_integration_tests/common_assets/job_specs/hello_world.yml",
		})
	s.Require().NoError(err, "Error running hello world from job spec")

	jobID, err := utils.ExtractJobIDFromShortOutput(result)
	s.Require().NoErrorf(err, "error extracting Job ID after running it: %q", err)

	completedIn, err := s.waitForJobToComplete(jobID, 30*time.Second)
	s.Require().NoErrorf(err, "Error waiting for job to complete, waited %s: %q", completedIn, err)

	resultDescription, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobID})
	s.Require().NoError(err, "Error getting job description")

	s.Require().Contains(resultDescription, "hello bacalhau world", resultDescription)
}

func (s *JobsBasicRunsScenariosSuite) TestUnsupportedTaskEngineType() {
	result, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"/bacalhau_integration_tests/common_assets/job_specs/unsupported_engine_type.yml",
		})
	s.Require().NoErrorf(err, "Error running job: %q", err)
	s.Require().Contains(result, "not enough nodes to run job")
	s.Require().Contains(result, "does not support vroomvroom")
}

func TestJobsBasicRunsScenariosSuite(t *testing.T) {
	suite.Run(t, NewJobsBasicRunsScenariosTestSuite())
}
