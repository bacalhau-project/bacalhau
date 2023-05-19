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

var waitForError job.CheckStatesFunction = func(js model.JobState) (bool, error) {
	return js.State == model.JobStateError, nil
}

func disabledTestSpec() scenario.Scenario {
	return scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: &devstack.DevStackOptions{
				NumberOfRequesterOnlyNodes: 1,
				NumberOfComputeOnlyNodes:   1,
			},
		},
		Spec:          scenario.WasmHelloWorld.Spec,
		SubmitChecker: scenario.SubmitJobErrorContains("not enough nodes to run job"),
	}
}

func (s *DisabledFeatureTestSuite) TestNothingDisabled() {
	testCase := disabledTestSpec()
	testCase.SubmitChecker = scenario.SubmitJobSuccess()
	testCase.JobCheckers = scenario.WaitUntilSuccessful(1)
	testCase.Spec.Verifier = model.VerifierNoop
	testCase.Spec.Publisher = model.PublisherIpfs
	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledEngine() {
	testCase := disabledTestSpec()
	testCase.Stack.DevStackOptions.DisabledFeatures.Engines = []model.Engine{model.EngineWasm}

	// TODO: This is a hack – because we are doing engine filtering at the node
	// selection rather than node ranking stage (see store.ListForEngine) the
	// StoreNodeDiscoverer returns nothing but the IdentityNodeDiscoverer
	// returns the node we didn't want to see, because it doesn't know any
	// better. Really we either need to push *all* filtering down into the store
	// or allow the store to tell us what nodes it discarded.
	testCase.SubmitChecker = scenario.SubmitJobSuccess()
	testCase.JobCheckers = []job.CheckStatesFunction{waitForError}

	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledStorage() {
	testCase := disabledTestSpec()
	testCase.Stack.DevStackOptions.DisabledFeatures.Storages = []model.StorageSourceType{model.StorageSourceInline}

	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledVerifier() {
	testCase := disabledTestSpec()
	testCase.Spec.Verifier = model.VerifierNoop
	testCase.Stack.DevStackOptions.DisabledFeatures.Verifiers = []model.Verifier{model.VerifierNoop}

	s.RunScenario(testCase)
}

func (s *DisabledFeatureTestSuite) TestDisabledPublisher() {
	testCase := disabledTestSpec()
	testCase.Spec.Publisher = model.PublisherIpfs
	testCase.Stack.DevStackOptions.DisabledFeatures.Publishers = []model.Publisher{model.PublisherIpfs}

	s.RunScenario(testCase)
}
