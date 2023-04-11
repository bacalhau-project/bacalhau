package bidstrategy

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type waitingStrategy struct {
	underlying             BidStrategy
	waitOnBid, waitOnNoBid bool
}

func NewWaitingStrategy(underlying BidStrategy, waitOnBid, waitOnNoBid bool) BidStrategy {
	return &waitingStrategy{
		underlying:  underlying,
		waitOnBid:   waitOnBid,
		waitOnNoBid: waitOnNoBid,
	}
}

// ShouldBid implements BidStrategy
func (s *waitingStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	resp, err := s.underlying.ShouldBid(ctx, request)
	if (resp.ShouldBid && s.waitOnBid) || (!resp.ShouldBid && s.waitOnNoBid) {
		resp.ShouldWait = true
	}
	return resp, err
}

// ShouldBidBasedOnUsage implements BidStrategy
func (s *waitingStrategy) ShouldBidBasedOnUsage(
	ctx context.Context,
	request BidStrategyRequest,
	resourceUsage model.ResourceUsageData,
) (BidStrategyResponse, error) {
	resp, err := s.underlying.ShouldBidBasedOnUsage(ctx, request, resourceUsage)
	if (resp.ShouldBid && s.waitOnBid) || (!resp.ShouldBid && s.waitOnNoBid) {
		resp.ShouldWait = true
	}
	return resp, err
}

var _ BidStrategy = (*waitingStrategy)(nil)
