package test_integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AgentLicenseInspectExpiredSuite struct {
	BaseDockerComposeTestSuite
}

func NewAgentLicenseInspectExpiredSuite() *AgentLicenseInspectExpiredSuite {
	s := &AgentLicenseInspectExpiredSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *AgentLicenseInspectExpiredSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-node-with-custom-start-command.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/16_orchestrator_config_with_expired_license.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *AgentLicenseInspectExpiredSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in AgentLicenseInspectExpiredSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *AgentLicenseInspectExpiredSuite) TestValidateRemoteExpiredLicense() {
	agentLicenseInspectionOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau", "agent", "license", "inspect",
		},
	)
	s.Require().NoErrorf(err, "Error inspecting license: %q", err)

	expectedOutput := `Product      = Bacalhau
License ID   = 2fa89cd0-2ce1-4963-bde9-77ad182713bf
Customer ID  = test-customer-id-123
Valid Until  = 2025-01-07
Version      = v1
Expired      = true
Capabilities = max_nodes=1
Metadata     = someMetadata=valueOfSomeMetadata`

	s.Require().Contains(agentLicenseInspectionOutput, expectedOutput)
}

func TestAgentLicenseInspectExpiredSuite(t *testing.T) {
	suite.Run(t, NewAgentLicenseInspectExpiredSuite())
}
