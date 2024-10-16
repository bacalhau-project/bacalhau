package test_integration

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
	"time"
)

type CommonJobsScenariosSuite struct {
	BaseDockerComposeTestSuite
}

func NewCommonJobsScenariosTestSuite() *CommonJobsScenariosSuite {
	s := &CommonJobsScenariosSuite{}
	s.globalRunIdentifier = globalTestExecutionId
	s.suiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *CommonJobsScenariosSuite) SetupSuite() {
	// Suite specific Images can be created here if needed
	rawDockerComposeFilePath := "./assets/docker_compose_files/docker-compose-with-minio-and-registry.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, map[string]string{})
}

func (s *CommonJobsScenariosSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in CommonJobsScenariosSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *CommonJobsScenariosSuite) TestDockerRunHelloWorld() {
	result, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "docker", "run", "hello-world"})

	s.Require().NoError(err, "Error running hello world")

	jobID := extractJobIDFromOutput(result, &s.Suite)

	completedIn, err := s.waitForJobToComplete(jobID, 30*time.Second)
	s.Require().NoErrorf(err, "Error waiting for job to complete, waited %s: %q", completedIn, err)

	resultDescription, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobID})

	s.Require().NoError(err, "Error running job description")
	s.Require().Contains(resultDescription, "Hello from Docker", resultDescription)
}

func (s *CommonJobsScenariosSuite) TestHelloWorldJob() {
	result, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"--wait=false",
			"--id-only",
			"/app/assets/job_specs/hello_world.yml",
		})
	s.Require().NoError(err, "Error running hello world from job spec")

	jobID, err := extractJobIDFromShortOutput(result)
	s.Require().NoErrorf(err, "error extracting Job ID after running it: %q", err)

	completedIn, err := s.waitForJobToComplete(jobID, 30*time.Second)
	s.Require().NoErrorf(err, "Error waiting for job to complete, waited %s: %q", completedIn, err)

	resultDescription, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobID})
	s.Require().NoError(err, "Error running job description")

	s.Require().Contains(resultDescription, "hello bacalhau world", resultDescription)
}

func TestCommonJobsScenariosSuite(t *testing.T) {
	suite.Run(t, NewCommonJobsScenariosTestSuite())
}
