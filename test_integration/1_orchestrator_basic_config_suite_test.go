package test_integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/exec"
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
}

func (s *OrchestratorBasicConfigSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in OrchestratorBasicConfigSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *OrchestratorBasicConfigSuite) TestNodesCanBeListed() {
	nodeListOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau", "node", "list", "--output=json",
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
	s.Require().Equal("label2Value", label2Value.String())
}

func (s *OrchestratorBasicConfigSuite) TestOrchestratorNodeUpAndEnabled() {
	agentConfigOutput, err := s.executeCommandInDefaultJumpbox(
		[]string{"bacalhau", "agent", "config", "--output=json"},
	)
	s.Require().NoErrorf(err, "Error getting orchestrator agent config: %q", err)

	// TODO: Unfortunately the "bacalhau agent config" command is
	// TODO: unpredictable and returns wierd first characters.
	// TODO: For now, Trim the first character if it is not "{"
	cleanedJsonString := strings.TrimLeftFunc(agentConfigOutput, func(r rune) bool {
		return r != '{'
	})

	output, err := s.convertStringToDynamicJSON(cleanedJsonString)
	s.Require().NoError(err)

	orchestratorEnabled, err := output.Query("$.Orchestrator.Enabled")
	s.Require().NoError(err)
	s.Require().True(orchestratorEnabled.Bool())
}

func (s *OrchestratorBasicConfigSuite) TestLocalConfigBasicPersistence() {
	configFilePath := "/app/test-persistent.yaml"

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
		},
		exec.WithEnv([]string{
			"BACALHAU_UPDATECONFIG_INTERVAL=",
		}),
	)
	s.Require().NoErrorf(err, "Error listing config: %q", err)

	unmarshalledOutput, err := s.unmarshalJSONString(commandOutput, JSONArray)
	s.Require().NoErrorf(err, "Error unmarshalling response: %q", err)

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
