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

type OrchestratorBehindReverseProxySuite struct {
	BaseDockerComposeTestSuite
}

func NewOrchestratorBehindReverseProxySuite() *OrchestratorBehindReverseProxySuite {
	s := &OrchestratorBehindReverseProxySuite{}
	s.GlobalRunIdentifier = globalTestExecutionId
	s.SuiteRunIdentifier = strings.ToLower(strings.Split(uuid.New().String(), "-")[0])
	return s
}

func (s *OrchestratorBehindReverseProxySuite) SetupSuite() {
	// In this test suite, the orchestrator is running behind a reverse proxy, and all
	// the NATS traffic between orchestrator and compute Node go through a real reverse proxy (Traefik)

	rawDockerComposeFilePath := "./common_assets/docker_compose_files/orchestrator-compute-traefik-custom-startup.yml"
	s.Context, s.Cancel = context.WithCancel(context.Background())

	traefikConfigFile := s.commonAssets("nodes_configs/9_traefik_static_config.yaml")
	traefikStartCommand := fmt.Sprintf("--configFile=%s", traefikConfigFile)

	orchestratorConfigFile := s.commonAssets("nodes_configs/9_orchestrator_node_behind_reverse_proxy.yaml")
	orchestratorStartCommand := fmt.Sprintf("bacalhau serve --config=%s", orchestratorConfigFile)

	computeConfigFile := s.commonAssets("nodes_configs/9_compute_node_with_enforced_tls_nats.yaml")
	computeStartCommand := fmt.Sprintf("bacalhau serve --config=%s", computeConfigFile)
	extraRenderingData := map[string]interface{}{
		"OrchestratorStartCommand": orchestratorStartCommand,
		"ComputeStartCommand":      computeStartCommand,
		"TraefikStartCommand":      traefikStartCommand,
	}
	s.BaseDockerComposeTestSuite.SetupSuite(rawDockerComposeFilePath, extraRenderingData)
}

func (s *OrchestratorBehindReverseProxySuite) TearDownSuite() {
	s.T().Log("Tearing down [Test Suite] in OrchestratorBehindReverseProxySuite...")
	s.BaseDockerComposeTestSuite.TearDownSuite()
}

func (s *OrchestratorBehindReverseProxySuite) TestRunHelloWorldJobWithOrchestratorBehindReverseProxy() {
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

func (s *OrchestratorBehindReverseProxySuite) TestNatsConnectionWillFailWithoutRequireTLS() {
	_, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"nats",
			"--server=nats://i_am_very_secret_token@bacalhau-traefik-node:4222",
			"--no-tlsfirst",
			"pub",
			"node.info",
			"helloworld",
		})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "error: read tcp")
	s.Require().ErrorContains(err, "timeout")
}

func (s *OrchestratorBehindReverseProxySuite) TestNatsTLSConnectionWillFailWithoutGoingThroughReverseProxy() {
	_, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"nats",
			"--server=nats://i_am_very_secret_token@bacalhau-orchestrator-node:4222",
			"--tlsca=/bacalhau_integration_tests/common_assets/certificates/nats_custom/nats_root_ca.crt",
			"--tlsfirst",
			"pub",
			"node.info",
			"helloworld",
		})
	s.Require().Error(err)
	s.Require().ErrorContains(err, "error: tls: first record does not look like a TLS handshake")
}

func (s *OrchestratorBehindReverseProxySuite) TestNatsConnectionWillSucceedWithRequireTLS() {
	result, err := s.executeCommandInDefaultJumpbox(
		[]string{
			"nats",
			"--server=nats://i_am_very_secret_token@bacalhau-traefik-node:4222",
			"--tlsfirst",
			"pub",
			"node.info",
			"helloworld",
		})
	s.Require().NoError(err)
	s.Require().Contains(result, `Published 10 bytes to "node.info"`)
}

func TestOrchestratorBehindReverseProxySuite(t *testing.T) {
	suite.Run(t, NewOrchestratorBehindReverseProxySuite())
}
