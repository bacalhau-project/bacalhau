package test_integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type CLIAcceptUrlInHostParamSuite struct {
	BaseDockerComposeTestSuite
}

func NewCLIAcceptUrlInHostParamSuite() *CLIAcceptUrlInHostParamSuite {
	s := &CLIAcceptUrlInHostParamSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *CLIAcceptUrlInHostParamSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-node-with-url-as-host.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	orchestratorConfigFile := s.commonAssets("nodes_configs/11_http_url_host_config.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *CLIAcceptUrlInHostParamSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in CLIAcceptUrlInHostParamSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *CLIAcceptUrlInHostParamSuite) TestNodesCanBeListedWithURLHost() {
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

func TestCLIAcceptUrlInHostParamSuite(t *testing.T) {
	suite.Run(t, NewCLIAcceptUrlInHostParamSuite())
}
