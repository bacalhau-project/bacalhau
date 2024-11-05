package test_integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"bacalhau/integration_tests/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type NatsTLSDefaultTruststoreSuite struct {
	BaseDockerComposeTestSuite
}

func NewNatsTLSDefaultTruststoreTestSuite() *NatsTLSDefaultTruststoreSuite {
	s := &NatsTLSDefaultTruststoreSuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *NatsTLSDefaultTruststoreSuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-and-compute-custom-startup.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	// In the configuration we do not specify the CA cert of NATS since it was already embed in the
	// container trust store for both compute and orchestrator, thus testing default trust stores
	orchestratorConfigFile := s.commonAssets("nodes_configs/7_tls_enabled_orchestrator.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)

	computeConfigFile := s.commonAssets("nodes_configs/7_tls_enabled_compute.yaml")
	computeStartCommand := fmt.Sprintf("bacalhau serve --config=%s", computeConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
		"ComputeStartCommand":      computeStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *NatsTLSDefaultTruststoreSuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in NatsTLSDefaultTruststoreSuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *NatsTLSDefaultTruststoreSuite) TestRunHelloWorldJobWithTLSEnabledNats() {
	result, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"bacalhau",
			"job",
			"run",
			"--wait=false",
			"--id-only",
			"/bacalhau_integration_tests/common_assets/job_specs/hello_world.yml",
		})
	s.Require().NoError(err)

	jobID, err := utils.ExtractJobIDFromShortOutput(result)
	s.Require().NoError(err)

	_, err = s.waitForJobToComplete(jobID, 30*time.Second)
	s.Require().NoError(err)

	resultDescription, err := s.executeCommandInDefaultJumpbox([]string{"bacalhau", "job", "describe", jobID})
	s.Require().NoError(err)
	s.Require().Contains(resultDescription, "hello bacalhau world", resultDescription)
}

func TestNatsTLSDefaultTruststoreSuite(t *testing.T) {
	suite.Run(t, NewNatsTLSDefaultTruststoreTestSuite())
}
