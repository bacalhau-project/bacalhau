//go:build integration

package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"

	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/suite"
)

type MinBidsSuite struct {
	scenario.ScenarioRunner
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestMinBidsSuite(t *testing.T) {
	suite.Run(t, new(MinBidsSuite))
}

type minBidsTestCase struct {
	nodes          int
	shards         int
	concurrency    int
	minBids        int
	expectedResult map[model.ExecutionStateType]int
	submitChecker  scenario.CheckSubmitResponse
	errorStates    []model.ExecutionStateType
}

func (s *MinBidsSuite) testMinBids(testCase minBidsTestCase) {
	spec := scenario.WasmHelloWorld.Spec
	spec.Sharding = model.JobShardingConfig{
		GlobPattern: "/input/*",
		BatchSize:   1,
	}

	testScenario := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: &devstack.DevStackOptions{NumberOfHybridNodes: testCase.nodes},
		},
		Inputs: scenario.StoredFile(
			prepareFolderWithFiles(s.T(), testCase.shards),
			"/input",
		),
		Spec: spec,
		Deal: model.Deal{
			Concurrency: testCase.concurrency,
			MinBids:     testCase.minBids,
		},
		JobCheckers: []job.CheckStatesFunction{
			job.WaitExecutionsThrowErrors(testCase.errorStates),
			job.WaitForExecutionStates(testCase.expectedResult),
		},
		SubmitChecker: testCase.submitChecker,
	}

	s.RunScenario(testScenario)
}

func (s *MinBidsSuite) TestMinBids_0and1Node() {
	// sanity test that with min bids at zero and 1 node we get the job through
	s.testMinBids(minBidsTestCase{
		nodes:       1,
		shards:      1,
		concurrency: 1,
		minBids:     0,
		expectedResult: map[model.ExecutionStateType]int{
			model.ExecutionStateCompleted: 1,
		},
		errorStates: []model.ExecutionStateType{
			model.ExecutionStateFailed,
		},
	})
}

func (s *MinBidsSuite) TestMinBids_isConcurrency() {
	// test that when min bids is concurrency we get the job through
	s.testMinBids(minBidsTestCase{
		nodes:       3,
		shards:      1,
		concurrency: 3,
		minBids:     3,
		expectedResult: map[model.ExecutionStateType]int{
			model.ExecutionStateCompleted: 3,
		},
		errorStates: []model.ExecutionStateType{
			model.ExecutionStateFailed,
		},
	})

}

func (s *MinBidsSuite) TestMinBids_noBids() {
	// test that no bids are made because there are not enough nodes on the network
	// to satisfy the min bids
	s.testMinBids(minBidsTestCase{
		nodes:         3,
		shards:        1,
		concurrency:   3,
		minBids:       5,
		submitChecker: scenario.SubmitJobErrorContains("not enough"),
	})

}
