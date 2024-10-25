package test_integration

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
)

type OrchestratorConfigOverrideAndFlagSuite struct {
	BaseDockerComposeTestSuite
}

func NewOrchestratorConfigOverrideAndFlagSuite() *OrchestratorConfigOverrideAndFlagSuite {
	s := &OrchestratorConfigOverrideAndFlagSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *OrchestratorConfigOverrideAndFlagSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-node-with-custom-start-command.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile1 := s.commonAssets("nodes_configs/2_config.yaml")
	orchestratorConfigFile2 := s.commonAssets("nodes_configs/2_config_override.yaml")
	orchestratorStartCommand := fmt.Sprintf(
		"bacalhau serve --config=%s --config=%s --labels=extralabel=extravalue",
		orchestratorConfigFile1,
		orchestratorConfigFile2,
	)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *OrchestratorConfigOverrideAndFlagSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in OrchestratorConfigOverrideAndFlagSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *OrchestratorConfigOverrideAndFlagSuite) TestConfigOverrideFileAndFlag() {
	nodeListOutput, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "node", "list", "--output=json"})
	s.Require().NoErrorf(err, "Error listing nodes: %q", err)

	unmarshalledOutput, err := s.unmarshalJSONString(nodeListOutput, JSONArray)
	s.Require().NoErrorf(err, "Error unmarshalling response: %q", err)

	unmarshalledOutputArray := unmarshalledOutput.([]interface{})
	s.Require().Equalf(1, len(unmarshalledOutputArray), "There should be only one node, but actual count is: %d", len(unmarshalledOutputArray))

	nodeMembership := unmarshalledOutputArray[0].(map[string]interface{})["Membership"]
	s.Require().Equalf("APPROVED", nodeMembership.(string), "Expected node to be approved, but actual is: %s", nodeMembership)

	nodeConnection := unmarshalledOutputArray[0].(map[string]interface{})["Connection"]
	s.Require().Equalf("CONNECTED", nodeConnection.(string), "Expected node to be connectted, but actual is: %s", nodeConnection)

	nodeInfo := unmarshalledOutputArray[0].(map[string]interface{})["Info"].(map[string]interface{})
	nodeType := nodeInfo["NodeType"].(string)
	s.Require().Equalf("Requester", nodeType, "Expected node to be an orchestrator, but actual is: %s", nodeType)

	nodeLabels := nodeInfo["Labels"].(map[string]interface{})

	_, label1Exists := nodeLabels["label1"]
	_, label2Exists := nodeLabels["label2"]
	s.Require().False(label1Exists, "Expected label1 to not exist")
	s.Require().False(label2Exists, "Expected label2 to not exist")
	s.Require().Equalf("extravalue", nodeLabels["extralabel"].(string), "Expected extralabel to be extravalue, but actual is: %s", nodeLabels["extralabel"].(string))
}

func TestOrchestratorConfigOverrideAndFlagSuite(t *testing.T) {
	suite.Run(t, NewOrchestratorConfigOverrideAndFlagSuite())
}
