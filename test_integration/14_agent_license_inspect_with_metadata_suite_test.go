package test_integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AgentLicenseInspectWithMetadataSuite struct {
	BaseDockerComposeTestSuite
}

func NewAgentLicenseInspectWithMetadataSuite() *AgentLicenseInspectWithMetadataSuite {
	s := &AgentLicenseInspectWithMetadataSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *AgentLicenseInspectWithMetadataSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-node-with-custom-start-command.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/14_orchestrator_config_with_license_with_metadata.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *AgentLicenseInspectWithMetadataSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in AgentLicenseInspectWithMetadataSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *AgentLicenseInspectWithMetadataSuite) TestValidateRemoteLicenseWithMetadata() {
	agentLicenseInspectionOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau", "agent", "license", "inspect",
		},
	)
	s.Require().NoErrorf(err, "Error inspecting license: %q", err)

	expectedOutput := `Product      = Bacalhau
License ID   = 2d58c7c9-ec29-45a5-a5cd-cb8f7fee6678
Customer ID  = test-customer-id-123
Valid Until  = 2045-07-28
Version      = v1
Capabilities = max_nodes=1
Metadata     = someMetadata=valueOfSomeMetadata`

	s.Require().Contains(agentLicenseInspectionOutput, expectedOutput)
}

func (s *AgentLicenseInspectWithMetadataSuite) TestValidateAgentLicenseYAMLOutput() {
	agentLicenseInspectionOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"agent",
			"license",
			"inspect",
			"--output=yaml",
		},
	)
	s.Require().NoErrorf(err, "Error inspecting license: %q", err)

	s.Require().Contains(agentLicenseInspectionOutput, "someMetadata: valueOfSomeMetadata")
	s.Require().Contains(agentLicenseInspectionOutput, "jti: 2d58c7c9-ec29-45a5-a5cd-cb8f7fee6678")
	s.Require().Contains(agentLicenseInspectionOutput, "iss: https://expanso.io/")
	s.Require().Contains(agentLicenseInspectionOutput, "iat: 1736889682")
	s.Require().Contains(agentLicenseInspectionOutput, "exp: 2384889682")
	s.Require().Contains(agentLicenseInspectionOutput, "customer_id: test-customer-id-123")
	s.Require().Contains(agentLicenseInspectionOutput, "license_id: 2d58c7c9-ec29-45a5-a5cd-cb8f7fee6678")
	s.Require().Contains(agentLicenseInspectionOutput, "license_type: standard")
	s.Require().Contains(agentLicenseInspectionOutput, "sub: test-customer-id-123")
	s.Require().Contains(agentLicenseInspectionOutput, "product: Bacalhau")
	s.Require().Contains(agentLicenseInspectionOutput, "license_version: v1")
	s.Require().Contains(agentLicenseInspectionOutput, "max_nodes: \"1\"")
}

func TestAgentLicenseInspectWithMetadataSuite(t *testing.T) {
	suite.Run(t, NewAgentLicenseInspectWithMetadataSuite())
}
