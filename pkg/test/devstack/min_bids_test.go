//go:build integration || !unit

package devstack

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/requester/retry"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
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
	concurrency    int
	minBids        int
	errorNodes     uint32
	expectedResult map[model.ExecutionStateType]int
	submitChecker  scenario.CheckSubmitResponse
	errorStates    []model.ExecutionStateType
}

func (s *MinBidsSuite) testMinBids(testCase minBidsTestCase) {
	responses := atomic.Uint32{}
	computeConfig := node.DefaultComputeConfig
	globalBidStrat := &bidstrategy.CallbackBidStrategy{
		OnShouldBid: func(ctx context.Context, bsr bidstrategy.BidStrategyRequest) (r bidstrategy.BidStrategyResponse, err error) {
			r = bidstrategy.NewShouldBidResponse()
			if num := responses.Add(1); num <= testCase.errorNodes {
				s.T().Logf("Node ID %s will return an error response", bsr.NodeID)
				err = fmt.Errorf("bad response")
			}
			return
		},
		OnShouldBidBasedOnUsage: func(ctx context.Context, bsr bidstrategy.BidStrategyRequest, rud model.ResourceUsageData) (bidstrategy.BidStrategyResponse, error) {
			return bidstrategy.NewShouldBidResponse(), nil
		},
	}
	computeConfig.BidResourceStrategy = globalBidStrat
	computeConfig.BidSemanticStrategy = globalBidStrat

	// We have to turn off retries for this test so that we can check what
	// happens when not enough bids are received If retries are switched on, the
	// requester will just try again and receive an adequate number of bids
	requesterConfig := node.DefaultRequesterConfig
	requesterConfig.RetryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: false})

	testScenario := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: &devstack.DevStackOptions{NumberOfHybridNodes: testCase.nodes},
			ComputeConfig:   node.NewComputeConfigWith(computeConfig),
			RequesterConfig: node.NewRequesterConfigWith(requesterConfig),
		},
		Spec: scenario.WasmHelloWorld.Spec,
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

func (s *MinBidsSuite) Test0and1Node() {
	// sanity test that with min bids at zero and 1 node we get the job through
	s.testMinBids(minBidsTestCase{
		nodes:       1,
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

func (s *MinBidsSuite) Test1and3Node() {
	// sanity test that with min bids at number of nodes, all nodes receive a bid response
	s.testMinBids(minBidsTestCase{
		nodes:       3,
		concurrency: 1,
		minBids:     3,
		expectedResult: map[model.ExecutionStateType]int{
			model.ExecutionStateCompleted:   1,
			model.ExecutionStateBidRejected: 2,
		},
		errorStates: []model.ExecutionStateType{
			model.ExecutionStateFailed,
		},
	})
}

func (s *MinBidsSuite) TestCancelsJobIfNotEnoughBids() {
	// test that when we have min bids and enough nodes fail to respond, we don't run
	s.testMinBids(minBidsTestCase{
		nodes:       3,
		errorNodes:  2,
		concurrency: 1,
		minBids:     2,
		expectedResult: map[model.ExecutionStateType]int{
			model.ExecutionStateCanceled: 1,
			model.ExecutionStateFailed:   2,
		},
		errorStates: []model.ExecutionStateType{
			model.ExecutionStateCompleted,
		},
	})
}

func (s *MinBidsSuite) TestAtConcurrency() {
	// test that when min bids is concurrency we get the job through
	s.testMinBids(minBidsTestCase{
		nodes:       3,
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

func (s *MinBidsSuite) TestNoBidsWhenNetworkTooSmall() {
	// test that no bids are made because there are not enough nodes on the network
	// to satisfy the min bids
	s.testMinBids(minBidsTestCase{
		nodes:         3,
		concurrency:   3,
		minBids:       5,
		submitChecker: scenario.SubmitJobErrorContains("not enough"),
	})

}
