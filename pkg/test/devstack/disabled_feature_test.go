//go:build integration || !unit

package devstack

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/suite"
)

type DisabledFeatureTestSuite struct {
	scenario.ScenarioRunner
}

func TestDisabledFeatureSuite(t *testing.T) {
	suite.Run(t, new(DisabledFeatureTestSuite))
}

func disabledTestSpec() scenario.Scenario {
	return scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: &devstack.DevStackOptions{
				NumberOfRequesterOnlyNodes: 1,
				NumberOfComputeOnlyNodes:   1,
			},
		},
		Spec: scenario.WasmHelloWorld.Spec,
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForUnsuccessfulCompletion(),
		},
	}
}

func (s *DisabledFeatureTestSuite) TestNothingDisabled() {
	testCase := disabledTestSpec()
	testCase.SubmitChecker = scenario.SubmitJobSuccess()
	testCase.JobCheckers = scenario.WaitUntilSuccessful(1)
	testCase.Spec.Publisher = model.PublisherIpfs
	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledEngine() {
	testCase := disabledTestSpec()
	testCase.Stack.DevStackOptions.DisabledFeatures.Engines = []model.Engine{model.EngineWasm}

	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledStorage() {
	testCase := disabledTestSpec()
	testCase.Stack.DevStackOptions.DisabledFeatures.Storages = []model.StorageSourceType{model.StorageSourceInline}

	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledPublisher() {
	testCase := disabledTestSpec()
	testCase.Spec.Publisher = model.PublisherIpfs
	testCase.Stack.DevStackOptions.DisabledFeatures.Publishers = []model.Publisher{model.PublisherIpfs}

	s.RunScenario(testCase)
}
