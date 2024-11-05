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

type NatsTLSCustomCASuite struct {
	BaseDockerComposeTestSuite
}

func NewNatsTLSCustomCATestSuite() *NatsTLSCustomCASuite {
	s := &NatsTLSCustomCASuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *NatsTLSCustomCASuite) SetupSuite() {
	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-and-compute-custom-startup.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	// In the configuration we do not specify the CA cert of NATS since it was already embed in the
	// container trust store for both compute and orchestrator, thus testing default trust stores
	orchestratorConfigFile := s.commonAssets("nodes_configs/8_tls_enabled_orchestrator_with_custom_ca.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)

	computeConfigFile := s.commonAssets("nodes_configs/8_tls_enabled_compute_with_custom_ca.yaml")
	computeStartCommand := fmt.Sprintf("bacalhau serve --config=%s", computeConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
		"ComputeStartCommand":      computeStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *NatsTLSCustomCASuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in NatsTLSCustomCASuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *NatsTLSCustomCASuite) TestRunHelloWorldJobWithCustomCAEnabledNats() {
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

func (s *NatsTLSCustomCASuite) TestNatsServerDoesNotAcceptNonTLSConnections() {
	_, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"nats",
			"--server=nats://i_am_very_secret_token@bacalhau-orchestrator-node:4222",
			"pub",
			"node.info",
			"helloworld",
		})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "tls: failed to verify certificate: x509: certificate signed by unknown authority")
}

func (s *NatsTLSCustomCASuite) TestNatsServerAcceptCustomCAConnections() {
	result, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"nats",
			"--server=nats://i_am_very_secret_token@bacalhau-orchestrator-node:4222",
			"--tlsca=/bacalhau_integration_tests/common_assets/certificates/nats_custom/nats_root_ca.crt",
			"pub",
			"node.info",
			"helloworld",
		})
	s.Require().NoError(err)
	s.Require().Contains(result, `Published 10 bytes to "node.info"`)
}

func TestNatsTLSCustomCASuite(t *testing.T) {
	suite.Run(t, NewNatsTLSCustomCATestSuite())
}
