package test_integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type AgentLicenseInspectSuite struct {
	BaseDockerComposeTestSuite
}

func NewAgentLicenseInspectSuite() *AgentLicenseInspectSuite {
	s := &AgentLicenseInspectSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *AgentLicenseInspectSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-node-with-custom-start-command.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/13_orchestrator_config_with_license.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *AgentLicenseInspectSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in AgentLicenseInspectSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *AgentLicenseInspectSuite) TestValidateRemoteLicense() {
	agentLicenseInspectionOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau", "agent", "license", "inspect",
		},
	)
	s.Require().NoErrorf(err, "Error inspecting license: %q", err)

	s.Require().Contains(agentLicenseInspectionOutput, "Product      = Bacalhau")
	s.Require().Contains(agentLicenseInspectionOutput, "License ID   = e66d1f3a-a8d8-4d57-8f14-00722844afe2")
	s.Require().Contains(agentLicenseInspectionOutput, "Customer ID  = test-customer-id-123")
	s.Require().Contains(agentLicenseInspectionOutput, "Valid Until  = 2045-07-28")
	s.Require().Contains(agentLicenseInspectionOutput, "Version      = v1")
	s.Require().Contains(agentLicenseInspectionOutput, "Expired      = false")
	s.Require().Contains(agentLicenseInspectionOutput, "Capabilities = max_nodes=1")
	s.Require().Contains(agentLicenseInspectionOutput, "Metadata     = {}")
}

func (s *AgentLicenseInspectSuite) TestValidateAgentLicenseJSONOutput() {
	agentLicenseInspectionOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"agent",
			"license",
			"inspect",
			"--output=json",
		},
	)
	s.Require().NoErrorf(err, "Error inspecting license: %q", err)

	output, err := s.convertStringToDynamicJSON(agentLicenseInspectionOutput)
	s.Require().NoError(err)

	productName, err := output.Query("$.product")
	s.Require().NoError(err)
	s.Require().Equal("Bacalhau", productName.String())

	licenseID, err := output.Query("$.license_id")
	s.Require().NoError(err)
	s.Require().Equal("e66d1f3a-a8d8-4d57-8f14-00722844afe2", licenseID.String())

	customerID, err := output.Query("$.customer_id")
	s.Require().NoError(err)
	s.Require().Equal("test-customer-id-123", customerID.String())

	licenseVersion, err := output.Query("$.license_version")
	s.Require().NoError(err)
	s.Require().Equal("v1", licenseVersion.String())

	capabilitiesMaxNodes, err := output.Query("$.capabilities.max_nodes")
	s.Require().NoError(err)
	s.Require().Equal("1", capabilitiesMaxNodes.String())
}

func TestAgentLicenseInspectSuite(t *testing.T) {
	suite.Run(t, NewAgentLicenseInspectSuite())
}
