package test_integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// cSpell:disable
type SSAuthConfigSuite struct {
	BaseDockerComposeTestSuite
}

func NewSSAuthConfigSuite() *SSAuthConfigSuite {
	s := &SSAuthConfigSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *SSAuthConfigSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-and-compute-custom-startup.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/18_sso_auth_enabled_orchestrator.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)

	computeConfigFile := s.commonAssets("nodes_configs/18_sso_auth_enabled_compute_node.yaml")
	computeStartCommand := fmt.Sprintf("bacalhau serve --config=%s", computeConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
		"ComputeStartCommand":      computeStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *SSAuthConfigSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in SSAuthConfigSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *SSAuthConfigSuite) TestAuthIsSkippedForSSOLoginCommand() {
	// We rely on the error to verify we were able to fetch authconfig
	_, err := s.executeCommandInDefaultJumpbox(
		[]string{"bacalhau", "auth", "sso", "login"},
	)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "unable to initiate SSO login flow")
}

func (s *SSAuthConfigSuite) TestAuthIsSkippedForCertainCommands() {
	result, err := s.executeCommandInDefaultJumpbox(
		[]string{"bacalhau", "version"},
	)
	s.Require().NoError(err)
	s.Require().Contains(result, "SERVER", result)

	result, err = s.executeCommandInDefaultJumpbox(
		[]string{"bacalhau", "agent", "version"},
	)
	s.Require().NoError(err)
	s.Require().Contains(result, "BuildDate")
	s.Require().Contains(result, "GitCommit")

	result, err = s.executeCommandInDefaultJumpbox(
		[]string{"bacalhau", "agent", "alive"},
	)
	s.Require().NoError(err)
	s.Require().Contains(result, "Status: OK")
}

func TestSSAuthConfigSuite(t *testing.T) {
	suite.Run(t, NewSSAuthConfigSuite())
}
