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

type JobsUpdateSuite struct {
	BaseDockerComposeTestSuite
}

func NewJobsUpdateTestSuite() *JobsUpdateSuite {
	s := &JobsUpdateSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *JobsUpdateSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/deployment-with-basic-setup.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/19_orchestrator_config.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *JobsUpdateSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in JobsUpdateSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *JobsUpdateSuite) TestDockerRunHelloWorld() {
	result, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "docker", "run", "hello-world"})

	s.Require().NoError(err, "Error running hello world")

	jobID := utils.ExtractJobIDFromOutput(result, &s.Suite)

	completedIn, err := s.waitForJobToComplete(jobID, 30*time.Second)
	s.Require().NoErrorf(err, "Error waiting for job to complete, waited %s: %q", completedIn, err)

	resultDescription, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobID})

	s.Require().NoError(err, "Error running job description")
	s.Require().Contains(resultDescription, "Hello from Docker", resultDescription)
}

func (s *JobsUpdateSuite) TestHelloWorldJob() {
	jobName := "basic-hello-world"

	result, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"--wait=false",
			"--id-only",
			"/bacalhau_integration_tests/common_assets/job_specs/19-basic-hello-world-v1.yml",
		})
	s.Require().NoError(err, "Error running hello world from job spec")

	jobID, err := utils.ExtractJobIDFromShortOutput(result)
	s.Require().NoErrorf(err, "error extracting Job ID after running it: %q", err)

	completedIn, err := s.waitForJobToComplete(jobName, 30*time.Second)
	s.Require().NoErrorf(err, "Error waiting for job to complete, waited %s: %q", completedIn, err)

	resultDescription, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobID})
	s.Require().NoError(err, "Error getting job description by ID")
	s.Require().Contains(resultDescription, "hello bacalhau world1", resultDescription)

	resultDescriptionByName, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobName})
	s.Require().NoError(err, "Error getting job description by name")

	count := strings.Count(resultDescriptionByName, "hello bacalhau world1")
	s.Require().Equal(2, count, "Expected exactly 2 occurrences of 'hello bacalhau world1' "+
		"in job description, but found %d", count)

	// =======================================================================
	// =======================================================================
	// Update the Job and run it
	result, err = s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"--wait=false",
			"--id-only",
			"/bacalhau_integration_tests/common_assets/job_specs/19-basic-hello-world-v2.yml",
		})
	s.Require().NoError(err, "Error running hello world from job spec")

	jobID, err = utils.ExtractJobIDFromShortOutput(result)
	s.Require().NoErrorf(err, "error extracting Job ID after running it: %q", err)

	completedIn, err = s.waitForJobToComplete(jobName, 30*time.Second)
	s.Require().NoErrorf(err, "Error waiting for job to complete, waited %s: %q", completedIn, err)

	resultDescription, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobID})
	s.Require().NoError(err, "Error getting job description by ID")
	s.Require().Contains(resultDescription, "hello bacalhau world2", resultDescription)
	s.Require().NotContains(resultDescription, "hello bacalhau world1", resultDescription)

	resultDescriptionByName, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobName})
	s.Require().NoError(err, "Error getting job description by name")

	count = strings.Count(resultDescriptionByName, "hello bacalhau world2")
	s.Require().Equal(2, count, "Expected exactly 2 occurrences of 'hello bacalhau world2' "+
		"in job description, but found %d", count)

	// =======================================================================
	// =======================================================================
	// Describe version 1 of the job
	resultDescription, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job",
		"describe", jobID, "--version=1"})
	s.Require().NoError(err, "Error getting job description by ID")
	s.Require().Contains(resultDescription, "hello bacalhau world1", resultDescription)
	s.Require().NotContains(resultDescription, "hello bacalhau world2", resultDescription)

	resultDescriptionByName, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job",
		"describe", jobName, "--version=1"})
	s.Require().NoError(err, "Error getting job description by name")

	count = strings.Count(resultDescriptionByName, "hello bacalhau world1")
	s.Require().Equal(2, count, "Expected exactly 2 occurrences of 'hello bacalhau world1' "+
		"in job description, but found %d", count)
	s.Require().NotContains(resultDescriptionByName, "hello bacalhau world2", resultDescription)

	// =======================================================================
	// =======================================================================
	// Describe version 2 of the job
	resultDescription, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job",
		"describe", jobID, "--version=2"})
	s.Require().NoError(err, "Error getting job description by ID")
	s.Require().Contains(resultDescription, "hello bacalhau world2", resultDescription)
	s.Require().NotContains(resultDescription, "hello bacalhau world1", resultDescription)

	resultDescriptionByName, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job",
		"describe", jobName, "--version=2"})
	s.Require().NoError(err, "Error getting job description by name")

	count = strings.Count(resultDescriptionByName, "hello bacalhau world2")
	s.Require().Equal(2, count, "Expected exactly 2 occurrences of 'hello bacalhau world2' "+
		"in job description, but found %d", count)
	s.Require().NotContains(resultDescriptionByName, "hello bacalhau world1", resultDescription)

	// =======================================================================
	// =======================================================================
	// Display Job executions for latest job
	result, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job",
		"executions", jobName})
	s.Require().NoError(err, "Error getting job executions by Name latest version")

	count = strings.Count(result, "Completed")
	s.Require().Equal(2, count, "Expected exactly 2 occurrences of 'Completed  Stopped' "+
		"in list of executions, but found %d", count)

	// =======================================================================
	// =======================================================================
	// Display Job executions version 2
	result, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job",
		"executions", jobName, "--version=1"})
	s.Require().NoError(err, "Error getting job executions by Name version 1")

	count = strings.Count(result, "Completed")
	s.Require().Equal(2, count, "Expected exactly 2 occurrences of 'Completed  Stopped' "+
		"in list of executions, but found %d", count)

	// =======================================================================
	// =======================================================================
	// Display Job executions all-versions
	result, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job",
		"executions", jobName, "--all-versions"})
	s.Require().NoError(err, "Error getting all job executions")

	count = strings.Count(result, "Completed")
	s.Require().Equal(4, count, "Expected exactly 4 occurrences of 'Completed  Stopped' "+
		"in list of executions, but found %d", count)
}

func TestJobsUpdateSuite(t *testing.T) {
	suite.Run(t, NewJobsUpdateTestSuite())
}
