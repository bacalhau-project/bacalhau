//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/job"
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
				NumberOfRequesterOnlyNodes: 1,
				NumberOfComputeOnlyNodes:   1,
			},
		},
		Spec: scenario.WasmHelloWorld(t).Spec,
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForUnsuccessfulCompletion(),
		},
	}
}

func (s *DisabledFeatureTestSuite) TestNothingDisabled() {
	testCase := disabledTestSpec(s.T())
	testCase.SubmitChecker = scenario.SubmitJobSuccess()
	testCase.JobCheckers = scenario.WaitUntilSuccessful(1)
	testCase.Spec.Publisher = model.PublisherIpfs
	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledEngine() {
	testCase := disabledTestSpec(s.T())
	testCase.Stack.DevStackOptions.DisabledFeatures.Engines = []model.Engine{model.EngineWasm}

	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledStorage() {
	testCase := disabledTestSpec(s.T())
	testCase.Stack.DevStackOptions.DisabledFeatures.Storages = []model.StorageSourceType{model.StorageSourceInline}

	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledPublisher() {
	testCase := disabledTestSpec(s.T())
	testCase.Spec.Publisher = model.PublisherIpfs
	testCase.Stack.DevStackOptions.DisabledFeatures.Publishers = []model.Publisher{model.PublisherIpfs}

	s.RunScenario(testCase)
}
