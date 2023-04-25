//go:build integration || !unit

package compute

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inmemory"
)

type AsyncBidSuite struct {
	ComputeSuite

	strategy *bidstrategy.CallbackBidStrategy

	store         store.ExecutionStore
	callbackStore *CallbackStore
}

func TestAsyncBidSuite(t *testing.T) {
	suite.Run(t, new(AsyncBidSuite))
}

func (s *AsyncBidSuite) SetupSuite() {
	s.ComputeSuite.SetupSuite()
	s.strategy = bidstrategy.NewFixedBidStrategy(true, true)
	s.config.BidSemanticStrategy = s.strategy
	s.config.BidResourceStrategy = s.strategy

	s.store = inmemory.NewStore()
	s.callbackStore = &CallbackStore{}
	s.callbackStore.GetExecutionFn = s.store.GetExecution
	s.callbackStore.GetExecutionsFn = s.store.GetExecutions
	s.callbackStore.GetExecutionHistoryFn = s.store.GetExecutionHistory
	s.callbackStore.CreateExecutionFn = s.store.CreateExecution
	s.callbackStore.UpdateExecutionStateFn = s.store.UpdateExecutionState
	s.callbackStore.DeleteExecutionFn = s.store.DeleteExecution
	s.callbackStore.GetExecutionCountFn = s.store.GetExecutionCount
	s.config.ExecutionStore = s.callbackStore
}

func (s *AsyncBidSuite) TestAsyncApproval() {
	s.runAsyncBidTest(true)
}

func (s *AsyncBidSuite) TestAsyncReject() {
	s.runAsyncBidTest(false)
}

func (s *AsyncBidSuite) runAsyncBidTest(shouldBid bool) {
	job := generateJob()

	// override execution store create method so that we may wait for async execution creation after `AskForBid`
	executionCreatedWg := sync.WaitGroup{}
	executionCreatedWg.Add(1)
	s.callbackStore.CreateExecutionFn = func(ctx context.Context, execution store.Execution) error {
		defer executionCreatedWg.Done()
		return s.store.CreateExecution(ctx, execution)
	}
	s.strategy.OnShouldBid = func(ctx context.Context, bsr bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
		return bidstrategy.BidStrategyResponse{ShouldBid: shouldBid, ShouldWait: true}, nil
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expectingResponse := false
	go func() {
		defer wg.Done()
		select {
		case result := <-s.bidChannel:
			s.True(expectingResponse, "received result before it was expected")
			s.Equal(shouldBid, result.Accepted)
			s.Equal(job.Metadata.ID, result.JobID)
		case <-time.After(2 * time.Second):
			s.FailNow("did not receive a bid response")
		}
	}()

	resp, err := s.node.LocalEndpoint.AskForBid(ctx, compute.AskForBidRequest{
		ExecutionMetadata: compute.ExecutionMetadata{
			ExecutionID: "testing-execution-id_" + s.T().Name(),
			JobID:       job.ID(),
		},
		RoutingMetadata: compute.RoutingMetadata{TargetPeerID: s.node.ID, SourcePeerID: s.node.ID},
		Job:             job,
	})
	s.NoError(err)

	executionCreatedWg.Wait()

	execution, err := s.node.ExecutionStore.GetExecution(ctx, resp.ExecutionID)
	s.NoError(err)

	expectingResponse = true
	s.node.Bidder.ReturnBidResult(ctx, execution, &bidstrategy.BidStrategyResponse{
		ShouldBid: shouldBid,
	})

}
