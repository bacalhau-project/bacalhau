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
	"github.com/testcontainers/testcontainers-go/exec"
)

// cSpell:disable
type BasicAuthConfigSuite struct {
	BaseDockerComposeTestSuite
}

func NewBasicAuthConfigSuite() *BasicAuthConfigSuite {
	s := &BasicAuthConfigSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *BasicAuthConfigSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-and-compute-custom-startup.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/17_basic_auth_enabled_orchestrator.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)

	computeConfigFile := s.commonAssets("nodes_configs/17_basic_auth_enabled_compute_node.yaml")
	computeStartCommand := fmt.Sprintf("bacalhau serve --config=%s", computeConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
		"ComputeStartCommand":      computeStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *BasicAuthConfigSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in BasicAuthConfigSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *BasicAuthConfigSuite) TestRunHelloWorldJobWithDifferentAuth() {
	result, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"--wait=false",
			"--id-only",
			"/bacalhau_integration_tests/common_assets/job_specs/hello_world.yml",
		},
		exec.WithEnv([]string{
			"BACALHAU_API_USERNAME=snoopyusername",
			"BACALHAU_API_PASSWORD=snoopypassword",
		}),
	)
	s.Require().NoError(err)

	jobID, err := utils.ExtractJobIDFromShortOutput(result)
	s.Require().NoError(err)

	_, err = s.waitForJobToComplete(
		jobID,
		30*time.Second,
		exec.WithEnv([]string{
			"BACALHAU_API_USERNAME=snoopyusername",
			"BACALHAU_API_PASSWORD=snoopypassword",
		}),
	)
	s.Require().NoError(err)

	// Retrieve Jobs using readonly username/password
	resultDescription, err := s.executeCommandInDefaultJumpbox(
		[]string{"bacalhau", "job", "describe", jobID},
		exec.WithEnv([]string{
			"BACALHAU_API_USERNAME=readonlyusername",
			"BACALHAU_API_PASSWORD=readonlyuserpassword",
		}),
	)
	s.Require().NoError(err)
	s.Require().Contains(resultDescription, "hello bacalhau world", resultDescription)

	// Retrieve Jobs using API Key that has that capability
	resultDescription, err = s.executeCommandInDefaultJumpbox(
		[]string{"bacalhau", "job", "describe", jobID},
		exec.WithEnv([]string{
			"BACALHAU_API_KEY=P7D4CBB284634DD081FAC33868436ECCL",
		}),
	)
	s.Require().NoError(err)
	s.Require().Contains(resultDescription, "hello bacalhau world", resultDescription)

	// Retrieve Jobs using username/password
	resultDescription, err = s.executeCommandInDefaultJumpbox(
		[]string{"bacalhau", "job", "describe", jobID},
		exec.WithEnv([]string{
			"BACALHAU_API_USERNAME=snoopyusername",
			"BACALHAU_API_PASSWORD=snoopypassword",
		}),
	)
	s.Require().NoError(err)
	s.Require().Contains(resultDescription, "hello bacalhau world", resultDescription)

	// Submit a job without job write capability with username/password
	result, err = s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"/bacalhau_integration_tests/common_assets/job_specs/hello_world.yml",
		},
		exec.WithEnv([]string{
			"BACALHAU_API_USERNAME=readonlyusername",
			"BACALHAU_API_PASSWORD=readonlyuserpassword",
		}),
	)
	s.Require().ErrorContains(err, "user 'readonlyuser' does not have the required capability 'write:job'")

	// Submit a job without job write capability with APIKey
	result, err = s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"/bacalhau_integration_tests/common_assets/job_specs/hello_world.yml",
		},
		exec.WithEnv([]string{
			"BACALHAU_API_KEY=QWERTYHFGCBNSKFIREHFURHUFE7KEEFBN",
		}),
	)
	s.Require().ErrorContains(err, "user 'API key ending in ...EEFBN' does not have the required capability 'write:job'")

}

func (s *BasicAuthConfigSuite) TestCommandsBypassingAuthentication() {
	result, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "agent", "alive"})
	s.Require().NoError(err)
	s.Require().Contains(result, "Status: OK")

	result, err = s.executeCommandInDefaultJumpbox([]string{"bacalhau", "version"})
	s.Require().NoError(err)
	s.Require().Contains(result, "SERVER")
}

func (s *BasicAuthConfigSuite) TestAuthInfoCommand() {
	// Auth Info
	result, err := s.executeCommandInDefaultJumpbox(
		[]string{"bacalhau", "auth", "info"},
		exec.WithEnv([]string{
			"BACALHAU_API_USERNAME=snoopyusername",
			"BACALHAU_API_PASSWORD=snoopypassword",
		}),
	)
	s.Require().NoError(err)
	s.Require().Contains(result, "Environment Variables:", result)
	s.Require().Contains(result, "API Key: Not Set", result)
	s.Require().Contains(result, "Username: Set", result)
	s.Require().Contains(result, "Password: Set", result)
	s.Require().Contains(result, "Node SSO Authentication:", result)
	s.Require().Contains(result, "Server does not support SSO login", result)
	s.Require().Contains(result, "Note: Environment variables take precedence over other authentication mechanisms including SSO.", result)
	s.Require().Contains(result, "To use SSO login, please unset Auth related environment variables first.", result)
}

func TestBasicAuthConfigSuite(t *testing.T) {
	suite.Run(t, NewBasicAuthConfigSuite())
}
