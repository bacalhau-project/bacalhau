package test_integration

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
)

type OrchestratorConfigOverrideSuite struct {
	BaseDockerComposeTestSuite
}

func NewOrchestratorConfigOverrideSuite() *OrchestratorConfigOverrideSuite {
	s := &OrchestratorConfigOverrideSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *OrchestratorConfigOverrideSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-node-with-custom-start-command.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile1 := s.commonAssets("nodes_configs/2_config.yaml")
	orchestratorConfigFile2 := s.commonAssets("nodes_configs/2_config_override.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s --config=%s", orchestratorConfigFile1, orchestratorConfigFile2)
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

func (s *OrchestratorConfigOverrideSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in OrchestratorConfigOverrideSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *OrchestratorConfigOverrideSuite) TestConfigOverrideFile() {
	nodeListOutput, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "node", "list", "--output=json"})
	s.Require().NoErrorf(err, "Error listing nodes: %q", err)

	marshalledOutput, jsonType, err := s.unmarshalJSONString(nodeListOutput)
	s.Require().NoErrorf(err, "Error unmarshalling json string: %q. JSON output received: %q", err, nodeListOutput)
	s.Require().Equalf("array", jsonType, "incorrect json output type")

	marshalledOutputArray := marshalledOutput.([]interface{})
	s.Require().Equalf(1, len(marshalledOutputArray), "There should be only one node, but actual count is: %d", len(marshalledOutputArray))

	nodeMembership := marshalledOutputArray[0].(map[string]interface{})["Membership"]
	s.Require().Equalf("APPROVED", nodeMembership.(string), "Expected node to be approved, but actual is: %s", nodeMembership)

	nodeConnection := marshalledOutputArray[0].(map[string]interface{})["Connection"]
	s.Require().Equalf("CONNECTED", nodeConnection.(string), "Expected node to be connectted, but actual is: %s", nodeConnection)

	nodeInfo := marshalledOutputArray[0].(map[string]interface{})["Info"].(map[string]interface{})
	nodeType := nodeInfo["NodeType"].(string)
	s.Require().Equalf("Requester", nodeType, "Expected node to be an orchestrator, but actual is: %s", nodeType)

	nodeLabels := nodeInfo["Labels"].(map[string]interface{})
	s.Require().Equalf("label1Value", nodeLabels["label1"].(string), "Expected label1 to be label1Value, but actual is: %s", nodeLabels["label1"].(string))
	s.Require().Equalf("Override2Value", nodeLabels["label2"].(string), "Expected label2 to be Override2Value, but actual is: %s", nodeLabels["label2"].(string))
}

func TestOrchestratorConfigOverrideSuite(t *testing.T) {
	suite.Run(t, NewOrchestratorConfigOverrideSuite())
}
