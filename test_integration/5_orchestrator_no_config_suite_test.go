package test_integration

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
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

	_, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"mkdir",
			"-p",
			fmt.Sprintf("/app/%s", s.SuiteRunIdentifier),
		})
	s.Require().NoErrorf(err, "Error creating a suite tmp folder in the jumpbox node: %q", err)
}

func (s *OrchestratorNoConfigSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in OrchestratorNoConfigSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *OrchestratorNoConfigSuite) TestNodesCanBeListed() {
	nodeListOutput, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "node", "list", "--output=json"})
	s.Require().NoErrorf(err, "Error listing nodes: %q", err)

	marshalledOutput, jsonType, err := s.unmarshalJSONString(nodeListOutput)
	s.Require().NoErrorf(err, "Error unmarshalling json string: %q. JSON output received: %q", err, nodeListOutput)
	s.Require().Equalf("array", jsonType, "incorrect json output type")

	marshalledOutputArray := marshalledOutput.([]interface{})
	s.Require().Equalf(1, len(marshalledOutputArray), "There should be only one node, but actual count is: %d", len(marshalledOutputArray))

	nodeType := marshalledOutputArray[0].(map[string]interface{})["Info"].(map[string]interface{})["NodeType"].(string)
	s.Require().Equalf("Requester", nodeType, "Expected node to be an orchestrator, but actual is: %s", nodeType)
}

func (s *OrchestratorNoConfigSuite) TestStartingOrchestratorNodeWithConfigFile() {
	agentConfigOutput, err := s.executeCommandInDefaultJumpbox([]string{"curl", "http://bacalhau-orchestrator-node:1234/api/v1/agent/config"})
	s.Require().NoErrorf(err, "Error getting orchestrator agent config: %q", err)

	unmarshalledOutput, jsonType, err := s.unmarshalJSONString(agentConfigOutput)
	s.Require().NoErrorf(err, "Error unmarshalling json string: %q", err)
	s.Require().Equalf("object", jsonType, "incorrect json output type")

	unmarshalledOutputMap := unmarshalledOutput.(map[string]interface{})

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
