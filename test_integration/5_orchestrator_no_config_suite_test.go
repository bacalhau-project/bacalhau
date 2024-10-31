package test_integration

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type OrchestratorNoConfigSuite struct {
	BaseDockerComposeTestSuite
}

func NewOrchestratorNoConfigSuite() *OrchestratorNoConfigSuite {
	s := &OrchestratorNoConfigSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *OrchestratorNoConfigSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-node-with-custom-start-command.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorStartCommand := "bacalhau serve"
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *OrchestratorNoConfigSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in OrchestratorNoConfigSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *OrchestratorNoConfigSuite) TestNodesCanBeListed() {
	nodeListOutput, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "node", "list", "--output=json"})
	s.Require().NoErrorf(err, "Error listing nodes: %q", err)

	unmarshalledOutput, err := s.unmarshalJSONString(nodeListOutput, JSONArray)
	s.Require().NoErrorf(err, "Error unmarshalling response: %q", err)

	unmarshalledOutputArray := unmarshalledOutput.([]interface{})
	s.Require().Equalf(1, len(unmarshalledOutputArray), "There should be only one node, but actual count is: %d", len(unmarshalledOutputArray))

	nodeType := unmarshalledOutputArray[0].(map[string]interface{})["Info"].(map[string]interface{})["NodeType"].(string)
	s.Require().Equalf("Requester", nodeType, "Expected node to be an orchestrator, but actual is: %s", nodeType)
}

func (s *OrchestratorNoConfigSuite) TestStartingOrchestratorNodeWithConfigFile() {
	agentConfigOutput, err := s.executeCommandInDefaultJumpbox([]string{"curl", "http://bacalhau-orchestrator-node:1234/api/v1/agent/config"})
	s.Require().NoErrorf(err, "Error getting orchestrator agent config: %q", err)

	unmarshalledOutput, err := s.unmarshalJSONString(agentConfigOutput, JSONObject)
	s.Require().NoErrorf(err, "Error unmarshalling response: %q", err)

	unmarshalledOutputMap := unmarshalledOutput.(map[string]interface{})["config"].(map[string]interface{})

	orchestratorEnabled := unmarshalledOutputMap["Orchestrator"].(map[string]interface{})["Enabled"].(bool)
	s.Require().Truef(orchestratorEnabled, "Expected orchestrator to be enabled, got: %t", orchestratorEnabled)

	computeEnabled := unmarshalledOutputMap["Compute"].(map[string]interface{})["Enabled"]
	s.Require().Nilf(computeEnabled, "Expected Compute to be disabled, got: %q", computeEnabled)

	computeLabels := unmarshalledOutputMap["Compute"].(map[string]interface{})["Labels"]
	s.Require().Nilf(computeLabels, "Expected Compute labels to be null, got: %q", computeLabels)

	nameProvider := unmarshalledOutputMap["NameProvider"].(string)
	s.Require().Equalf("puuid", nameProvider, "Incorrect NameProvider, got: %q", nameProvider)

	apiHost := unmarshalledOutputMap["API"].(map[string]interface{})["Host"].(string)
	s.Require().Equalf("0.0.0.0", apiHost, "Incorrect API.Host, got: %q", apiHost)

	apiPort := int(unmarshalledOutputMap["API"].(map[string]interface{})["Port"].(float64))
	s.Require().Equalf(1234, apiPort, "Incorrect API.Port, got: %d", apiPort)
}

func TestOrchestratorNoConfigSuite(t *testing.T) {
	suite.Run(t, NewOrchestratorNoConfigSuite())
}
