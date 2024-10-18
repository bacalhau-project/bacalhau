package test_integration

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
)

type OrchestratorBasicConfigSuite struct {
	BaseDockerComposeTestSuite
}

func NewOrchestratorBasicConfigSuite() *OrchestratorBasicConfigSuite {
	s := &OrchestratorBasicConfigSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *OrchestratorBasicConfigSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-node-with-custom-start-command.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/1_basic_orchestrator_config.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)
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

func (s *OrchestratorBasicConfigSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in OrchestratorBasicConfigSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *OrchestratorBasicConfigSuite) TestNodesCanBeListed() {
	nodeListOutput, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "node", "list", "--output=json"})
	s.Require().NoErrorf(err, "Error listing nodes: %q", err)

	marshalledOutput, jsonType, err := s.unmarshalJSONString(nodeListOutput)
	s.Require().NoErrorf(err, "Error unmarshalling json string: %q. JSON output received: %q", err, nodeListOutput)
	s.Require().Equalf("array", jsonType, "incorrect json output type")

	marshalledOutputArray := marshalledOutput.([]interface{})
	s.Require().Equalf(1, len(marshalledOutputArray), "Node count should be 1")

	nodeInfo := marshalledOutputArray[0].(map[string]interface{})["Info"].(map[string]interface{})

	nodeType := nodeInfo["NodeType"].(string)
	s.Require().Equal("Requester", nodeType)

	nodeLabels := nodeInfo["Labels"].(map[string]interface{})
	s.Require().Equal("label1Value", nodeLabels["label1"].(string))
	s.Require().Equal("label2Value", nodeLabels["label2"].(string))
}

func (s *OrchestratorBasicConfigSuite) TestStartingOrchestratorNodeWithConfigFile() {
	agentConfigOutput, err := s.executeCommandInDefaultJumpbox([]string{"curl", "http://bacalhau-orchestrator-node:1234/api/v1/agent/config"})
	s.Require().NoErrorf(err, "Error getting orchestrator agent config: %q", err)

	marshalledOutput, jsonType, err := s.unmarshalJSONString(agentConfigOutput)
	s.Require().NoErrorf(err, "Error unmarshalling json string: %q", err)
	s.Require().Equalf("object", jsonType, "incorrect json output type")

	orchestratorEnabled := marshalledOutput.(map[string]interface{})["Orchestrator"].(map[string]interface{})["Enabled"].(bool)
	s.Require().Truef(orchestratorEnabled, "Expected orchestrator to be enabled, got: %t", orchestratorEnabled)
}

func (s *OrchestratorBasicConfigSuite) TestLocalConfigBasicPersistence() {
	configFilePath := fmt.Sprintf("/app/%s/test-persistent.yaml", s.SuiteRunIdentifier)

	_, err := s.executeCommandInDefaultJumpbox([]string{"touch", configFilePath})
	s.Require().NoErrorf(err, "Error creating config file: %q", err)

	_, err = s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"config",
			"set",
			fmt.Sprintf("--config=%s", configFilePath),
			"NameProvider=Some_Random_VALUE",
		})
	s.Require().NoErrorf(err, "Error setting config value to config file: %q", err)

	commandOutput, err := s.executeCommandInDefaultJumpbox([]string{"cat", configFilePath})
	s.Require().NoErrorf(err, "Error catting config file: %q", err)
	s.Require().Contains(commandOutput, "nameprovider: Some_Random_VALUE")

	_, err = s.executeCommandInDefaultJumpbox([]string{"rm", configFilePath})
	s.Require().NoErrorf(err, "Error removing test config file: %q", err)
}

func (s *OrchestratorBasicConfigSuite) TestDefaultUpdateCheckInterval() {
	commandOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"config",
			"list",
			"--output=json",
		})
	s.Require().NoErrorf(err, "Error listing config: %q", err)

	unmarshalledOutput, jsonType, err := s.unmarshalJSONString(commandOutput)
	s.Require().NoErrorf(err, "Error unmarshalling json string: %q. JSON output received: %q", err, unmarshalledOutput)
	s.Require().Equalf("array", jsonType, "incorrect json output type")

	configListArray := unmarshalledOutput.([]interface{})

	checkForUpdatesInterval := ""
	for _, configItemRaw := range configListArray {
		configItem := configItemRaw.(map[string]interface{})
		if configItemName, exists := configItem["Key"]; exists && configItemName == "UpdateConfig.Interval" {
			if configItemValue, exists := configItem["Value"]; exists {
				checkForUpdatesInterval = configItemValue.(string)
				break
			}
		}
	}

	s.Require().Equalf("24h0m0s", checkForUpdatesInterval, "Default interval to check for updates should be 24h0m0s")
}

func TestOrchestratorBasicConfigSuite(t *testing.T) {
	suite.Run(t, NewOrchestratorBasicConfigSuite())
}
