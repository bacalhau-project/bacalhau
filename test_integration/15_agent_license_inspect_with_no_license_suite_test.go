package test_integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AgentLicenseInspectWithNoLicenseSuite struct {
	BaseDockerComposeTestSuite
}

func NewAgentLicenseInspectWithNoLicenseSuite() *AgentLicenseInspectWithNoLicenseSuite {
	s := &AgentLicenseInspectWithNoLicenseSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *AgentLicenseInspectWithNoLicenseSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-node-with-custom-start-command.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/15_orchestrator_config_with_no_license.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *AgentLicenseInspectWithNoLicenseSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in AgentLicenseInspectWithNoLicenseSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *AgentLicenseInspectWithNoLicenseSuite) TestValidateRemoteLicenseWithNoLicense() {
	_, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau", "agent", "license", "inspect",
		},
	)

	s.Require().ErrorContains(err, "Error inspecting orchestrator license: No license configured for orchestrator.")
}

func TestAgentLicenseInspectWithNoLicenseSuite(t *testing.T) {
	suite.Run(t, NewAgentLicenseInspectWithNoLicenseSuite())
}
