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
)

type AsyncBidSuite struct {
	ComputeSuite

	strategy *bidstrategy.CallbackBidStrategy
}

func TestAsyncBidSuite(t *testing.T) {
	suite.Run(t, new(AsyncBidSuite))
}

func (s *AsyncBidSuite) SetupSuite() {
	s.ComputeSuite.SetupSuite()
	s.strategy = bidstrategy.NewFixedBidStrategy(true, true)
	s.config.BidSemanticStrategy = s.strategy
	s.config.BidResourceStrategy = s.strategy
}

func (s *AsyncBidSuite) TestAsyncApproval() {
	s.runAsyncBidTest(true)
}

func (s *AsyncBidSuite) TestAsyncReject() {
	s.runAsyncBidTest(false)
}

func (s *AsyncBidSuite) runAsyncBidTest(response bool) {
	job := generateJob()

	s.strategy.OnShouldBid = func(ctx context.Context, bsr bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
		return bidstrategy.BidStrategyResponse{ShouldBid: response, ShouldWait: true}, nil
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
			s.Equal(response, result.Accepted)
			s.Equal(job.Metadata.ID, result.JobID)
		case <-time.After(2 * time.Second):
			s.FailNow("did not receive a bid response")
		}
	}()

	resp, err := s.node.LocalEndpoint.AskForBid(ctx, compute.AskForBidRequest{
		RoutingMetadata: compute.RoutingMetadata{TargetPeerID: s.node.ID, SourcePeerID: s.node.ID},
		Job:             job,
	})
	s.NoError(err)

	execution, err := s.node.ExecutionStore.GetExecution(ctx, resp.ExecutionID)
	s.NoError(err)

	expectingResponse = true
	s.node.Bidder.ReturnBidResult(ctx, execution, &bidstrategy.BidStrategyResponse{
		ShouldBid: response,
	})
}
