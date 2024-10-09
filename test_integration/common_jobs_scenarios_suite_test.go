package test_integration

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
	"time"
)

type HappyPathTestSuite struct {
	BaseDockerComposeTestSuite
}

func NewHappyPathTestSuite() *HappyPathTestSuite {
	s := &HappyPathTestSuite{}
	s.globalRunIdentifier = globalTestExecutionId
	s.suiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *HappyPathTestSuite) SetupSuite() {
	// Suite specific Images can be created here if needed
	rawDockerComposeFilePath := "./assets/docker_compose_files/docker-compose-with-minio-and-registry.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, map[string]string{})
}

func (s *HappyPathTestSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in HappyPathTestSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *HappyPathTestSuite) TestDockerRunHelloWorld() {
	result, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "docker", "run", "hello-world"})

	s.Require().NoError(err, "Error running hello world")

	jobID := extractJobIDFromOutput(result, &s.Suite)

	completedIn, err := s.waitForJobToComplete(jobID, 30*time.Second)
	s.Require().NoErrorf(err, "Error waiting for job to complete, waited %s: %q", completedIn, err)

	resultDescription, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobID})

	s.Require().NoError(err, "Error running job description")
	s.Require().Contains(resultDescription, "Hello from Docker", resultDescription)
}

func (s *HappyPathTestSuite) TestHelloWorldJob() {
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

func TestHappyPathTestSuite(t *testing.T) {
	suite.Run(t, NewHappyPathTestSuite())
}
