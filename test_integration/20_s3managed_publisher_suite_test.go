package test_integration

import (
	"bacalhau/integration_tests/utils"
	"context"
	"fmt"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type S3ManagedPublisherTestSuite struct {
	BaseDockerComposeTestSuite
}

func NewS3ManagedPublisherSuite() *S3ManagedPublisherTestSuite {
	s := &S3ManagedPublisherTestSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *S3ManagedPublisherTestSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/deployment-with-minio-and-registry.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/20_orchestrator_with_minio_managed_publisher.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)

	mcAlias := "bacalhau-minio"

	// Create MinIO alias using the environment variables
	// We use sh -c to enable shell expansion of environment variables
	_, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"sh", "-c",
			fmt.Sprintf("mc alias set %s http://$BACALHAU_MINIO_NODE_HOST:9000 $MINIO_ROOT_USER $MINIO_ROOT_PASSWORD", mcAlias),
		})
	s.Require().NoErrorf(err, "Error creating minio alias: %q", err)

	// Create the bucket
	_, err = s.executeCommandInDefaultJumpbox(
		[]string{
			"mc",
			"mb",
			"--ignore-existing",
			mcAlias + "/bacalhau-managed-publisher",
		})
	s.Require().NoErrorf(err, "Error creating bucket for managed publisher: %q", err)
}

func (s *S3ManagedPublisherTestSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in S3ManagedPublisherTestSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *S3ManagedPublisherTestSuite) TestS3ManagedPublisherJobCompletes() {
	result, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"--wait=false",
			"--id-only",
			"/bacalhau_integration_tests/common_assets/job_specs/20-s3managed-publisher.yml",
		})
	s.Require().NoError(err, "Error running job with S3 managed publisher")

	jobID, err := utils.ExtractJobIDFromShortOutput(result)
	s.Require().NoErrorf(err, "error extracting Job ID after running it: %q", err)

	completedIn, err := s.waitForJobToComplete(jobID, 30*time.Second)
	s.Require().NoErrorf(err, "Error waiting for job to complete, waited %s: %q", completedIn, err)

	// Prepare a directory to store the job results on jumpbox
	outputDir := fmt.Sprintf("/tmp/bacalhau-job-results-%s", jobID)
	result, err = s.executeCommandInDefaultJumpbox(
		[]string{
			"mkdir",
			"-p",
			outputDir,
		})
	s.Require().NoErrorf(err, "Error creating a directory to store job results: %q", err)

	jobGetResult, err := s.fetchJobResults(jobID, outputDir)
	s.Require().NoErrorf(err, "Error getting job results: %q", err)
	s.Require().Contains(jobGetResult, fmt.Sprintf("Results for job '%s' have been written", jobID))

	result, err = s.executeCommandInDefaultJumpbox(
		[]string{
			"cat",
			path.Join(outputDir, "stdout"),
		})
	s.Require().NoErrorf(err, "Error reading job results: %q", err)
	s.Require().Contains(result, "expected execution stdout")
}

func TestS3ManagedPublisherTestSuite(t *testing.T) {
	suite.Run(t, NewS3ManagedPublisherSuite())
}
