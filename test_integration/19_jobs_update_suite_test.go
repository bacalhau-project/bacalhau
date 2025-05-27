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

	resultDescriptionByNameJSON, err := s.executeCommandInDefaultJumpbox([]string{
		"bacalhau",
		"job",
		"describe",
		jobName,
		"--output=json",
	})
	dynamicJSONOutput, err := s.convertStringToDynamicJSON(resultDescriptionByNameJSON)
	s.Require().NoError(err)

	jobVersion, err := dynamicJSONOutput.Query("$.Job.Version")
	s.Require().NoError(err)
	s.Require().Equal(1, jobVersion.Int())

	executionsList, err := dynamicJSONOutput.Query("$.Executions.Items")
	s.Require().NoError(err)
	s.Require().Equal(2, len(executionsList.Array()))

	// Check the first and second execution job version value
	firstExecutionJobVersion, err := dynamicJSONOutput.Query("$.Executions.Items[0].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(1, firstExecutionJobVersion.Int())

	secondExecutionJobVersion, err := dynamicJSONOutput.Query("$.Executions.Items[1].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(1, secondExecutionJobVersion.Int())

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
	s.Require().NotContains(resultDescriptionByName, "hello bacalhau world1", resultDescription)

	resultDescriptionByNameJSON, err = s.executeCommandInDefaultJumpbox([]string{
		"bacalhau",
		"job",
		"describe",
		jobName,
		"--output=json",
	})
	dynamicJSONOutput, err = s.convertStringToDynamicJSON(resultDescriptionByNameJSON)
	s.Require().NoError(err)

	jobVersion, err = dynamicJSONOutput.Query("$.Job.Version")
	s.Require().NoError(err)
	s.Require().Equal(2, jobVersion.Int())

	executionsList, err = dynamicJSONOutput.Query("$.Executions.Items")
	s.Require().NoError(err)
	s.Require().Equal(2, len(executionsList.Array()))

	// Check the first and second execution job version value
	firstExecutionJobVersion, err = dynamicJSONOutput.Query("$.Executions.Items[0].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(2, firstExecutionJobVersion.Int())

	secondExecutionJobVersion, err = dynamicJSONOutput.Query("$.Executions.Items[1].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(2, secondExecutionJobVersion.Int())

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

	resultDescriptionByNameJSON, err = s.executeCommandInDefaultJumpbox([]string{
		"bacalhau",
		"job",
		"describe",
		jobName,
		"--version=1",
		"--output=json",
	})
	dynamicJSONOutput, err = s.convertStringToDynamicJSON(resultDescriptionByNameJSON)
	s.Require().NoError(err)

	jobVersion, err = dynamicJSONOutput.Query("$.Job.Version")
	s.Require().NoError(err)
	s.Require().Equal(1, jobVersion.Int())

	executionsList, err = dynamicJSONOutput.Query("$.Executions.Items")
	s.Require().NoError(err)
	s.Require().Equal(2, len(executionsList.Array()))

	// Check the first and second execution job version value
	firstExecutionJobVersion, err = dynamicJSONOutput.Query("$.Executions.Items[0].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(1, firstExecutionJobVersion.Int())

	secondExecutionJobVersion, err = dynamicJSONOutput.Query("$.Executions.Items[1].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(1, secondExecutionJobVersion.Int())

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

	resultDescriptionByNameJSON, err = s.executeCommandInDefaultJumpbox([]string{
		"bacalhau",
		"job",
		"describe",
		jobName,
		"--version=2",
		"--output=json",
	})
	dynamicJSONOutput, err = s.convertStringToDynamicJSON(resultDescriptionByNameJSON)
	s.Require().NoError(err)

	jobVersion, err = dynamicJSONOutput.Query("$.Job.Version")
	s.Require().NoError(err)
	s.Require().Equal(2, jobVersion.Int())

	executionsList, err = dynamicJSONOutput.Query("$.Executions.Items")
	s.Require().NoError(err)
	s.Require().Equal(2, len(executionsList.Array()))

	// Check the first and second execution job version value
	firstExecutionJobVersion, err = dynamicJSONOutput.Query("$.Executions.Items[0].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(2, firstExecutionJobVersion.Int())

	secondExecutionJobVersion, err = dynamicJSONOutput.Query("$.Executions.Items[1].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(2, secondExecutionJobVersion.Int())

	// =======================================================================
	// =======================================================================
	// Display Job executions for latest job
	result, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job",
		"executions", jobName})
	s.Require().NoError(err, "Error getting job executions by Name latest version")

	count = strings.Count(result, "Completed")
	s.Require().Equal(2, count, "Expected exactly 2 occurrences of 'Completed  Stopped' "+
		"in list of executions, but found %d", count)

	resultExecutionsByNameJSON, err := s.executeCommandInDefaultJumpbox([]string{
		"bacalhau",
		"job",
		"executions",
		jobName,
		"--output=json",
	})
	dynamicJSONOutput, err = s.convertStringToDynamicJSON(resultExecutionsByNameJSON)
	s.Require().NoError(err)

	executionsList, err = dynamicJSONOutput.Query("$")
	s.Require().NoError(err)
	s.Require().Equal(2, len(executionsList.Array()))

	// Check the first and second execution job version value
	firstExecutionJobVersion, err = dynamicJSONOutput.Query("$[0].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(2, firstExecutionJobVersion.Int())

	secondExecutionJobVersion, err = dynamicJSONOutput.Query("$[1].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(2, secondExecutionJobVersion.Int())

	// =======================================================================
	// =======================================================================
	// Display Job executions version 2
	result, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job",
		"executions", jobName, "--version=1"})
	s.Require().NoError(err, "Error getting job executions by Name version 1")

	count = strings.Count(result, "Completed")
	s.Require().Equal(2, count, "Expected exactly 2 occurrences of 'Completed  Stopped' "+
		"in list of executions, but found %d", count)

	resultExecutionsByNameJSON, err = s.executeCommandInDefaultJumpbox([]string{
		"bacalhau",
		"job",
		"executions",
		jobName,
		"--version=1",
		"--output=json",
	})
	dynamicJSONOutput, err = s.convertStringToDynamicJSON(resultExecutionsByNameJSON)
	s.Require().NoError(err)

	executionsList, err = dynamicJSONOutput.Query("$")
	s.Require().NoError(err)
	s.Require().Equal(2, len(executionsList.Array()))

	// Check the first and second execution job version value
	firstExecutionJobVersion, err = dynamicJSONOutput.Query("$[0].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(1, firstExecutionJobVersion.Int())

	secondExecutionJobVersion, err = dynamicJSONOutput.Query("$[1].JobVersion")
	s.Require().NoError(err)
	s.Require().Equal(1, secondExecutionJobVersion.Int())

	// =======================================================================
	// =======================================================================
	// Display Job executions all-versions
	result, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job",
		"executions", jobName, "--all-versions"})
	s.Require().NoError(err, "Error getting all job executions")

	count = strings.Count(result, "Completed")
	s.Require().Equal(4, count, "Expected exactly 4 occurrences of 'Completed  Stopped' "+
		"in list of executions, but found %d", count)

	resultExecutionsByNameJSON, err = s.executeCommandInDefaultJumpbox([]string{
		"bacalhau",
		"job",
		"executions",
		jobName,
		"--all-versions",
		"--output=json",
	})
	dynamicJSONOutput, err = s.convertStringToDynamicJSON(resultExecutionsByNameJSON)
	s.Require().NoError(err)

	executionsList, err = dynamicJSONOutput.Query("$")
	s.Require().NoError(err)
	s.Require().Equal(4, len(executionsList.Array()))

	// check that we have 4 executions with 2 for each version, the order is not guaranteed, do not rely on it
	// Count versions to ensure we have 2 executions with version 1 and 2 with version 2
	// This approach doesn't rely on order of results which isn't guaranteed
	versionCounts := map[int]int{1: 0, 2: 0}

	for i := 0; i < 4; i++ {
		version, err := dynamicJSONOutput.Query(fmt.Sprintf("$[%d].JobVersion", i))
		s.Require().NoError(err)
		versionCounts[version.Int()]++
	}

	s.Require().Equal(2, versionCounts[1], "Expected exactly 2 executions with version 1")
	s.Require().Equal(2, versionCounts[2], "Expected exactly 2 executions with version 2")

	// =======================================================================
	// =======================================================================
	// Display job versions

	resultJobVersionsByNameJSON, err := s.executeCommandInDefaultJumpbox([]string{
		"bacalhau",
		"job",
		"versions",
		jobName,
		"--output=json",
	})
	dynamicJSONOutput, err = s.convertStringToDynamicJSON(resultJobVersionsByNameJSON)
	s.Require().NoError(err)

	executionsList, err = dynamicJSONOutput.Query("$")
	s.Require().NoError(err)
	s.Require().Equal(2, len(executionsList.Array()))

	// Check the first and second job version value there should be 2 versions,
	firstJobVersion, err := dynamicJSONOutput.Query("$[0].Version")
	s.Require().NoError(err)
	secondJobVersion, err := dynamicJSONOutput.Query("$[1].Version")
	s.Require().NoError(err)
	s.Require().True((firstJobVersion.Int() == 1 && secondJobVersion.Int() == 2) ||
		(firstJobVersion.Int() == 2 && secondJobVersion.Int() == 1),
		"Expected job versions to be 1 and 2, but got %d and %d", firstJobVersion.Int(), secondJobVersion.Int())

	// ==========================================================================
	// ==========================================================================
	// Resubmitting the job with no changes, it should not create a new job version
	result, err = s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"/bacalhau_integration_tests/common_assets/job_specs/19-basic-hello-world-v2.yml",
		})
	s.Require().ErrorContains(err, "no changes detected for new job spec. Job Name: 'basic-hello-world'")
	s.Require().ErrorContains(err, "Use the --force flag to override this warning")

	// ==========================================================================
	// ==========================================================================
	// Resubmitting the job with no changes, but with a force flag, it should create a new job version

	result, err = s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"--wait=false",
			"--id-only",
			"--force",
			"/bacalhau_integration_tests/common_assets/job_specs/19-basic-hello-world-v2.yml",
		})
	s.Require().NoError(err, "Error running hello world from job spec")

	jobID, err = utils.ExtractJobIDFromShortOutput(result)
	s.Require().NoErrorf(err, "error extracting Job ID after running it: %q", err)

	completedIn, err = s.waitForJobToComplete(jobName, 30*time.Second)
	s.Require().NoErrorf(err, "Error waiting for job to complete, waited %s: %q", completedIn, err)

	resultDescriptionByName, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobName})
	s.Require().NoError(err, "Error getting job description by name")

	count = strings.Count(resultDescriptionByName, "hello bacalhau world2")
	s.Require().Equal(2, count, "Expected exactly 2 occurrences of 'hello bacalhau world2' "+
		"in job description, but found %d", count)
	s.Require().NotContains(resultDescriptionByName, "hello bacalhau world1", resultDescription)

	resultJobVersionsByNameJSON, err = s.executeCommandInDefaultJumpbox([]string{
		"bacalhau",
		"job",
		"versions",
		jobName,
		"--output=json",
	})
	dynamicJSONOutput, err = s.convertStringToDynamicJSON(resultJobVersionsByNameJSON)
	s.Require().NoError(err)

	executionsList, err = dynamicJSONOutput.Query("$")
	s.Require().NoError(err)
	s.Require().Equal(3, len(executionsList.Array()))

	// Check the first and second job version value there should be 2 versions,
	firstJobVersion, err = dynamicJSONOutput.Query("$[0].Version")
	s.Require().NoError(err)
	secondJobVersion, err = dynamicJSONOutput.Query("$[1].Version")
	s.Require().NoError(err)
	thirdJobVersion, err := dynamicJSONOutput.Query("$[2].Version")
	s.Require().NoError(err)

	// inject the 3 versions in a list and make sure the count of each version is 1
	jobVersions := []int{firstJobVersion.Int(), secondJobVersion.Int(), thirdJobVersion.Int()}
	versionCounts = map[int]int{1: 0, 2: 0, 3: 0}
	for _, version := range jobVersions {
		versionCounts[version]++
	}
	s.Require().Equal(1, versionCounts[1], "Expected exactly 1 execution with version 1")
	s.Require().Equal(1, versionCounts[2], "Expected exactly 1 execution with version 2")
	s.Require().Equal(1, versionCounts[3], "Expected exactly 1 execution with version 3")

	// ==========================================================================
	// ==========================================================================
	// Submit job with dry-run option
	result, err = s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"--dry-run",
			"/bacalhau_integration_tests/common_assets/job_specs/19-basic-hello-world-v1.yml",
		})
	s.Require().NoError(err, "Error running hello world from job spec")
	s.Require().Contains(result, "echo hello bacalhau world1")
	s.Require().Contains(result, "echo hello bacalhau world2")
}

func TestJobsUpdateSuite(t *testing.T) {
	suite.Run(t, NewJobsUpdateTestSuite())
}
