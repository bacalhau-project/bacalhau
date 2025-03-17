package test_integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type OrchestratorConfigOverrideAndFlagAndConfigFlagSuite struct {
	BaseDockerComposeTestSuite
}

func NewOrchestratorConfigOverrideAndFlagAndConfigFlagSuite() *OrchestratorConfigOverrideAndFlagAndConfigFlagSuite {
	s := &OrchestratorConfigOverrideAndFlagAndConfigFlagSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *OrchestratorConfigOverrideAndFlagAndConfigFlagSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-node-with-custom-start-command.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile1 := s.commonAssets("nodes_configs/2_config.yaml")
	orchestratorConfigFile2 := s.commonAssets("nodes_configs/2_config_override.yaml")
	orchestratorStartCommand := fmt.Sprintf(
		"bacalhau serve --config=%s --config=%s --config WebUI.Enabled=false --config webui.enabled=true",
		orchestratorConfigFile1,
		orchestratorConfigFile2,
	)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *OrchestratorConfigOverrideAndFlagAndConfigFlagSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in OrchestratorConfigOverrideAndFlagAndConfigFlagSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *OrchestratorConfigOverrideAndFlagAndConfigFlagSuite) TestConfigOverrideFileAndFlagAndConfigFlag() {
	nodeListOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"node",
			"list",
			"--output=json",
		},
	)
	s.Require().NoErrorf(err, "Error listing nodes: %q", err)

	output, err := s.convertStringToDynamicJSON(nodeListOutput)
	s.Require().NoError(err)

	nodeList, err := output.Query("$")
	s.Require().NoError(err)
	s.Require().Equal(1, len(nodeList.Array()))

	nodeInfo, err := output.Query("$[0].Info.NodeType")
	s.Require().NoError(err)
	s.Require().Equal("Requester", nodeInfo.String())

	label1Value, err := output.Query("$[0].Info.Labels.label1")
	s.Require().NoError(err)
	s.Require().Equal("label1Value", label1Value.String())
	label2Value, err := output.Query("$[0].Info.Labels.label2")
	s.Require().NoError(err)
	s.Require().Equal("Override2Value", label2Value.String())

	// Query WebUI Status
	agentConfigOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"agent",
			"config",
			"--output=json",
		},
	)
	s.Require().NoError(err)

	agentConfigJSON, err := s.convertStringToDynamicJSON(agentConfigOutput)
	s.Require().NoError(err)

	webUIEnabled, err := agentConfigJSON.Query("$.WebUI.Enabled")
	s.Require().NoError(err)
	s.Require().True(webUIEnabled.Bool())
}

func TestOrchestratorConfigOverrideAndFlagAndConfigFlagSuite(t *testing.T) {
	suite.Run(t, NewOrchestratorConfigOverrideAndFlagAndConfigFlagSuite())
}
