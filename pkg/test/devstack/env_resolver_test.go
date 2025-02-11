//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	dockmodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type EnvResolverSuite struct {
	scenario.ScenarioRunner
}

func TestDevstackEnvResolverSuite(t *testing.T) {
	suite.Run(t, new(EnvResolverSuite))
}

func (s *EnvResolverSuite) SetupSuite() {
	docker.MustHaveDocker(s.T())
}

// TestEnvVarResolution verifies that environment variables are properly resolved and passed to jobs
func (s *EnvResolverSuite) TestEnvVarResolution() {
	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithNumberOfHybridNodes(1),
			},
		},
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: s.T().Name(),
					Engine: dockmodels.NewDockerEngineBuilder("busybox:1.37.0").
						WithEntrypoint("sh", "-c", "echo -n Literal=$LITERAL_VAR Host=$HOST_VAR").
						MustBuild(),
					Publisher: publisher_local.NewSpecConfig(),
					Env: map[string]models.EnvVarValue{
						"LITERAL_VAR": "literal-value",
						"HOST_VAR":    "env:TEST_HOST_VAR",
					},
				},
			},
		},
		ResultsChecker: scenario.FileContains(
			"stdout",
			[]string{
				"Literal=literal-value Host=host-value",
			},
			1, // Expect exactly one line
		),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
		},
	}

	// Set host environment variable before test
	s.T().Setenv("TEST_HOST_VAR", "host-value")
	s.RunScenario(testCase)
}

// TestEnvVarResolverDenied verifies that non-allowlisted environment variables are blocked
func (s *EnvResolverSuite) TestEnvVarResolverDenied() {
	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithNumberOfHybridNodes(1),
			},
		},
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: s.T().Name(),
					Engine: dockmodels.NewDockerEngineBuilder("busybox:1.37.0").
						WithEntrypoint("sh", "-c", "echo $DENIED_VAR").
						MustBuild(),
					Publisher: publisher_local.NewSpecConfig(),
					Env: map[string]models.EnvVarValue{
						"DENIED_VAR": "env:DENIED_HOST_VAR", // This should be denied
					},
				},
			},
		},
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForUnsuccessfulCompletion(),
			scenario.WaitForExecutionStates(map[models.ExecutionStateType]int{
				models.ExecutionStateAskForBidRejected: 1,
			}),
		},
	}

	s.T().Setenv("DENIED_HOST_VAR", "should-not-pass")
	s.RunScenario(testCase)
}
