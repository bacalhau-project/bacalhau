package test_integration

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
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
		"bacalhau serve --config=%s --config=%s --web-ui=false --config webui.enabled=true",
		orchestratorConfigFile1,
		orchestratorConfigFile2,
	)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)

	_, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"mkdir",
			"-p",
			fmt.Sprintf("/app/%s", s.SuiteRunIdentifier),
		})
	s.Require().NoErrorf(err, "Error creating a suite tmp folder in the jumpbox node: %q", err)
}

func (s *OrchestratorConfigOverrideAndFlagAndConfigFlagSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in OrchestratorConfigOverrideAndFlagAndConfigFlagSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *OrchestratorConfigOverrideAndFlagAndConfigFlagSuite) TestConfigOverrideFileAndFlagAndConfigFlag() {
	nodeListOutput, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "node", "list", "--output=json"})
	s.Require().NoErrorf(err, "Error listing nodes: %q", err)

	marshalledOutput, jsonType, err := s.unmarshalJSONString(nodeListOutput)
	s.Require().NoErrorf(err, "Error unmarshalling json string: %q. JSON output received: %q", err, nodeListOutput)
	s.Require().Equalf("array", jsonType, "incorrect json output type")

	marshalledOutputArray := marshalledOutput.([]interface{})
	s.Require().Equalf(1, len(marshalledOutputArray), "There should be only one node, but actual count is: %d", len(marshalledOutputArray))

	nodeInfo := marshalledOutputArray[0].(map[string]interface{})["Info"].(map[string]interface{})
	nodeType := nodeInfo["NodeType"].(string)
	s.Require().Equalf("Requester", nodeType, "Expected node to be an orchestrator, but actual is: %s", nodeType)

	nodeLabels := nodeInfo["Labels"].(map[string]interface{})
	s.Require().Equalf("label1Value", nodeLabels["label1"].(string), "Expected label1 to be label1Value, but actual is: %s", nodeLabels["label1"].(string))
	s.Require().Equalf("Override2Value", nodeLabels["label2"].(string), "Expected label2 to be Override2Value, but actual is: %s", nodeLabels["label2"].(string))

	// Query WebUI Status
	agentConfigOutput, err := s.executeCommandInDefaultJumpbox([]string{"curl", "http://bacalhau-orchestrator-node:1234/api/v1/agent/config"})
	s.Require().NoErrorf(err, "Error getting orchestrator agent config: %q", err)

	marshalledAgentOutput, jsonType, err := s.unmarshalJSONString(agentConfigOutput)
	s.Require().NoErrorf(err, "Error unmarshalling json string: %q", err)
	s.Require().Equalf("object", jsonType, "incorrect json output type")

	webuiEnabled := marshalledAgentOutput.(map[string]interface{})["WebUI"].(map[string]interface{})["Enabled"].(bool)
	s.Require().Truef(webuiEnabled, "Expected orchestrator to be enabled, got: %t", webuiEnabled)
}

func TestOrchestratorConfigOverrideAndFlagAndConfigFlagSuite(t *testing.T) {
	suite.Run(t, NewOrchestratorConfigOverrideAndFlagAndConfigFlagSuite())
}
