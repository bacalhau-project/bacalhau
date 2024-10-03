//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type DisabledFeatureTestSuite struct {
	scenario.ScenarioRunner
}

func TestDisabledFeatureSuite(t *testing.T) {
	suite.Run(t, new(DisabledFeatureTestSuite))
}

func disabledTestSpec(t testing.TB, disabled node.FeatureConfig) scenario.Scenario {
	return scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithNumberOfHybridNodes(1),
				devstack.WithNumberOfComputeOnlyNodes(1),
				devstack.WithDisabledFeatures(disabled),
			},
		},
		Job: scenario.WasmHelloWorld(t).Job,
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForUnsuccessfulCompletion(),
		},
	}
}

func (s *DisabledFeatureTestSuite) TestNothingDisabled() {
	testCase := disabledTestSpec(s.T(), node.FeatureConfig{})
	testCase.SubmitChecker = scenario.SubmitJobSuccess()
	testCase.JobCheckers = scenario.WaitUntilSuccessful(1)
	testCase.Job.Task().Publisher = publisher_local.NewSpecConfig()
	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledEngine() {
	testCase := disabledTestSpec(s.T(), node.FeatureConfig{
		Engines: []string{models.EngineWasm},
	})
	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledStorage() {
	testCase := disabledTestSpec(s.T(), node.FeatureConfig{
		Storages: []string{models.StorageSourceInline},
	})
	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledPublisher() {
	testCase := disabledTestSpec(s.T(), node.FeatureConfig{
		Publishers: []string{models.PublisherLocal},
	})
	testCase.Job.Task().Publisher = publisher_local.NewSpecConfig()
	s.RunScenario(testCase)
}
