//go:build integration || !unit

package devstack

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	dockmodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type NetworkingSuite struct {
	scenario.ScenarioRunner
	server *http.Server
	port   int
}

func TestNetworkingSuite(t *testing.T) {
	suite.Run(t, new(NetworkingSuite))
}

func (s *NetworkingSuite) SetupSuite() {
	docker.MustHaveDocker(s.T())

	// Create a simple HTTP server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Get a free port
	port, err := network.GetFreePort()
	s.Require().NoError(err)
	s.port = port

	// Start server
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}
	listener, err := net.Listen("tcp", s.server.Addr)
	s.Require().NoError(err)

	go s.server.Serve(listener)

	// Wait for server to be ready
	s.Require().Eventually(func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond)
}

func (s *NetworkingSuite) TearDownSuite() {
	if s.server != nil {
		s.server.Close()
	}
}

// Helper to create a standard test scenario with networking enabled
func (s *NetworkingSuite) networkScenario(task *models.Task) scenario.Scenario {
	return scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithNumberOfHybridNodes(1),
				devstack.WithBacalhauConfigOverride(types.Bacalhau{
					JobAdmissionControl: types.JobAdmissionControl{
						AcceptNetworkedJobs: true,
					},
				}),
			},
		},
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{task},
		},
	}
}

func (s *NetworkingSuite) TestHostNetworking() {
	testutils.SkipIfNotLinux(s.T(), "docker host mode is not supported on non-linux platforms")

	testCase := s.networkScenario(&models.Task{
		Name: s.T().Name(),
		Engine: dockmodels.NewDockerEngineBuilder("curlimages/curl:8.1.0").
			WithEntrypoint("curl", "-s", "-f", fmt.Sprintf("http://localhost:%d", s.port)).
			MustBuild(),
		Publisher: publisher_local.NewSpecConfig(),
		Network: &models.NetworkConfig{
			Type: models.NetworkHost,
		},
	})
	testCase.ResultsChecker = scenario.FileEquals("stdout", "OK")
	testCase.JobCheckers = []scenario.StateChecks{scenario.WaitForSuccessfulCompletion()}

	s.RunScenario(testCase)
}

func (s *NetworkingSuite) TestBridgeNetworking() {
	testCase := s.networkScenario(&models.Task{
		Name: s.T().Name(),
		Engine: dockmodels.NewDockerEngineBuilder("curlimages/curl:8.1.0").
			WithEntrypoint("curl", "-s", "-f", fmt.Sprintf("http://host.docker.internal:%d", s.port)).
			MustBuild(),
		Publisher: publisher_local.NewSpecConfig(),
		Network: &models.NetworkConfig{
			Type: models.NetworkBridge,
		},
	})
	testCase.ResultsChecker = scenario.FileEquals("stdout", "OK")
	testCase.JobCheckers = []scenario.StateChecks{scenario.WaitForSuccessfulCompletion()}

	s.RunScenario(testCase)
}

func (s *NetworkingSuite) TestNetworkingNone() {
	testCase := s.networkScenario(&models.Task{
		Name: s.T().Name(),
		Engine: dockmodels.NewDockerEngineBuilder("curlimages/curl:8.1.0").
			WithEntrypoint("curl", "-s", "-f", fmt.Sprintf("http://localhost:%d", s.port)).
			MustBuild(),
		Publisher: publisher_local.NewSpecConfig(),
		Network: &models.NetworkConfig{
			Type: models.NetworkNone,
		},
	})
	testCase.ResultsChecker = scenario.ManyChecks(
		scenario.FileEquals("stdout", ""),
		scenario.FileEquals("exitCode", "7"),
	)
	testCase.JobCheckers = []scenario.StateChecks{scenario.WaitForSuccessfulCompletion()}

	s.RunScenario(testCase)
}

func (s *NetworkingSuite) TestPortMappingInBridgeMode() {
	containerPort := 8080
	hostPort, err := network.GetFreePort()
	s.Require().NoError(err)

	testCase := s.networkScenario(&models.Task{
		Name: s.T().Name(),
		Engine: dockmodels.NewDockerEngineBuilder("busybox:1.37.0").
			WithEntrypoint("sh", "-c", fmt.Sprintf(
				`while true; do echo -e "HTTP/1.1 200 OK\n\nOK" | nc -l -p %d; done`,
				containerPort)).
			MustBuild(),
		Publisher: publisher_local.NewSpecConfig(),
		Network: &models.NetworkConfig{
			Type: models.NetworkBridge,
			Ports: []*models.PortMapping{
				{
					Name:   "http",
					Static: hostPort,
					Target: containerPort,
				},
			},
		},
	})
	testCase.JobCheckers = []scenario.StateChecks{scenario.WaitForRunningState()}

	s.RunScenario(testCase)

	s.Require().Eventually(func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d", hostPort))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond)
}

func (s *NetworkingSuite) TestPortMappingInHostMode() {
	testutils.SkipIfNotLinux(s.T(), "docker host mode is not supported on non-linux platforms")

	hostPort, err := network.GetFreePort()
	s.Require().NoError(err)

	testCase := s.networkScenario(&models.Task{
		Name: s.T().Name(),
		Engine: dockmodels.NewDockerEngineBuilder("busybox:1.37.0").
			WithEntrypoint("sh", "-c", fmt.Sprintf(
				`while true; do echo -e "HTTP/1.1 200 OK\n\nOK" | nc -l -p %d; done`,
				hostPort)).
			MustBuild(),
		Publisher: publisher_local.NewSpecConfig(),
		Network: &models.NetworkConfig{
			Type: models.NetworkHost,
			Ports: []*models.PortMapping{
				{
					Name:   "http",
					Static: hostPort,
				},
			},
		},
	})
	testCase.JobCheckers = []scenario.StateChecks{scenario.WaitForRunningState()}

	s.RunScenario(testCase)

	// In host mode, the container port should be directly accessible
	s.Require().Eventually(func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d", hostPort))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond)
}

func (s *NetworkingSuite) TestInvalidPortMapping() {
	// Try to map to a privileged port
	testCase := s.networkScenario(&models.Task{
		Name: s.T().Name(),
		Engine: dockmodels.NewDockerEngineBuilder("busybox:1.37.0").
			WithEntrypoint("sh", "-c", "nc -l -p 80").
			MustBuild(),
		Publisher: publisher_local.NewSpecConfig(),
		Network: &models.NetworkConfig{
			Type: models.NetworkBridge,
			Ports: []*models.PortMapping{
				{
					Name:   "http",
					Static: 999, // Privileged port, should fail
					Target: 80,
				},
			},
		},
	})
	testCase.SubmitChecker = scenario.SubmitJobFail()
	s.RunScenario(testCase)
}
