//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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
		Spec: scenario.WasmHelloWorld(t).Spec,
		JobCheckers: []legacy_job.CheckStatesFunction{
			legacy_job.WaitForUnsuccessfulCompletion(),
		},
	}
}

func (s *DisabledFeatureTestSuite) TestNothingDisabled() {
	testCase := disabledTestSpec(s.T())
	testCase.SubmitChecker = scenario.SubmitJobSuccess()
	testCase.JobCheckers = scenario.WaitUntilSuccessful(1)
	testCase.Spec.PublisherSpec = model.PublisherSpec{
		Type: model.PublisherLocal,
	}
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
	testCase.Spec.PublisherSpec = model.PublisherSpec{
		Type: model.PublisherLocal,
	}
	testCase.Stack.DevStackOptions.DisabledFeatures.Publishers = []string{models.PublisherLocal}

	s.RunScenario(testCase)
}
