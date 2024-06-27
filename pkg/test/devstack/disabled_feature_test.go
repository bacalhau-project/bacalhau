//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
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

func disabledTestSpec(t testing.TB) scenario.Scenario {
	return scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: &devstack.DevStackOptions{
				NumberOfHybridNodes:      1,
				NumberOfComputeOnlyNodes: 1,
			},
		},
		Job: scenario.WasmHelloWorld(t).Job,
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForUnsuccessfulCompletion(),
		},
	}
}

func (s *DisabledFeatureTestSuite) TestNothingDisabled() {
	testCase := disabledTestSpec(s.T())
	testCase.SubmitChecker = scenario.SubmitJobSuccess()
	testCase.JobCheckers = scenario.WaitUntilSuccessful(1)
	testCase.Job.Task().Publisher = publisher_local.NewSpecConfig()
	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledEngine() {
	testCase := disabledTestSpec(s.T())
	testCase.Stack.DevStackOptions.DisabledFeatures.Engines = []string{models.EngineWasm}

	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledStorage() {
	testCase := disabledTestSpec(s.T())
	testCase.Stack.DevStackOptions.DisabledFeatures.Storages = []string{models.StorageSourceInline}

	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledPublisher() {
	testCase := disabledTestSpec(s.T())
	testCase.Job.Task().Publisher = publisher_local.NewSpecConfig()
	testCase.Stack.DevStackOptions.DisabledFeatures.Publishers = []string{models.PublisherLocal}

	s.RunScenario(testCase)
}
