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
		defer func() { _ = resp.Body.Close() }()
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

func (s *NetworkingSuite) TestUndefinedNetworking() {
	testCase := s.networkScenario(&models.Task{
		Name: s.T().Name(),
		Engine: dockmodels.NewDockerEngineBuilder("curlimages/curl:8.1.0").
			WithEntrypoint("curl", "-s", "-f", fmt.Sprintf("http://host.docker.internal:%d", s.port)).
			MustBuild(),
		Publisher: publisher_local.NewSpecConfig(),
		Network: &models.NetworkConfig{
			Type: models.NetworkDefault,
		},
	})
	testCase.ResultsChecker = scenario.FileEquals("stdout", "OK")
	testCase.JobCheckers = []scenario.StateChecks{scenario.WaitForSuccessfulCompletion()}

	s.RunScenario(testCase)
}

func (s *NetworkingSuite) TestUndefinedNetworkingRejected() {
	testCase := s.networkScenario(&models.Task{
		Name: s.T().Name(),
		Engine: dockmodels.NewDockerEngineBuilder("curlimages/curl:8.1.0").
			WithEntrypoint("curl", "-s", "-f", fmt.Sprintf("http://localhost:%d", s.port)).
			MustBuild(),
		Publisher: publisher_local.NewSpecConfig(),
		Network: &models.NetworkConfig{
			Type: models.NetworkDefault,
		},
	})
	testCase.ResultsChecker = scenario.ManyChecks(
		scenario.FileEquals("stdout", ""),
		scenario.FileEquals("exitCode", "7"),
	)
	testCase.JobCheckers = []scenario.StateChecks{scenario.WaitForSuccessfulCompletion()}

	testCase.Stack.DevStackOptions = append(testCase.Stack.DevStackOptions, devstack.WithBacalhauConfigOverride(types.Bacalhau{
		JobAdmissionControl: types.JobAdmissionControl{
			RejectNetworkedJobs: true,
		},
	}))

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
			Ports: models.PortMap{
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
		defer func() { _ = resp.Body.Close() }()
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
			Ports: models.PortMap{
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
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond)
}

func (s *NetworkingSuite) TestInvalidPortMapping() {
	testCases := []struct {
		name    string
		network *models.NetworkConfig
	}{
		{
			name: "privileged port",
			network: &models.NetworkConfig{
				Type: models.NetworkBridge,
				Ports: models.PortMap{
					{
						Name:   "http",
						Static: 999, // Privileged port, should fail
						Target: 80,
					},
				},
			},
		},
		{
			name: "port out of range",
			network: &models.NetworkConfig{
				Type: models.NetworkBridge,
				Ports: models.PortMap{
					{
						Name:   "invalid",
						Static: 65536, // Port > 65535, should fail
						Target: 8080,
					},
				},
			},
		},
		{
			name: "target port without bridge mode",
			network: &models.NetworkConfig{
				Type: models.NetworkHost,
				Ports: models.PortMap{
					{
						Name:   "http",
						Static: 8080,
						Target: 80, // Target port not valid in host mode
					},
				},
			},
		},
		{
			name: "duplicate port names",
			network: &models.NetworkConfig{
				Type: models.NetworkBridge,
				Ports: models.PortMap{
					{
						Name:   "http",
						Static: 8080,
						Target: 80,
					},
					{
						Name:   "http", // Same name, should fail
						Static: 8081,
						Target: 81,
					},
				},
			},
		},
		{
			name: "duplicate static ports",
			network: &models.NetworkConfig{
				Type: models.NetworkBridge,
				Ports: models.PortMap{
					{
						Name:   "http1",
						Static: 8080,
						Target: 80,
					},
					{
						Name:   "http2",
						Static: 8080, // Same static port, should fail
						Target: 81,
					},
				},
			},
		},
		{
			name: "duplicate target ports",
			network: &models.NetworkConfig{
				Type: models.NetworkBridge,
				Ports: models.PortMap{
					{
						Name:   "http1",
						Static: 8080,
						Target: 80,
					},
					{
						Name:   "http2",
						Static: 8081,
						Target: 80, // Same target port, should fail
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			testCase := s.networkScenario(&models.Task{
				Name: s.T().Name(),
				Engine: dockmodels.NewDockerEngineBuilder("busybox:1.37.0").
					WithEntrypoint("sh", "-c", "nc -l -p 80").
					MustBuild(),
				Publisher: publisher_local.NewSpecConfig(),
				Network:   tc.network,
			})
			testCase.SubmitChecker = scenario.SubmitJobFail()
			s.RunScenario(testCase)
		})
	}
}
