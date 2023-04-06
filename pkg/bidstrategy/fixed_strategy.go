package bidstrategy

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// FixedBidStrategy is a bid strategy that always returns the same response, which is useful for testing
type FixedBidStrategy struct {
	Response bool
	Wait     bool
}

// NewFixedBidStrategy creates a new FixedBidStrategy
func NewFixedBidStrategy(response, wait bool) *FixedBidStrategy {
	return &FixedBidStrategy{
		Response: response,
		Wait:     wait,
	}
}

func (s *FixedBidStrategy) ShouldBid(_ context.Context, _ BidStrategyRequest) (BidStrategyResponse, error) {
	return BidStrategyResponse{ShouldBid: s.Response, ShouldWait: s.Wait}, nil
}

func (s *FixedBidStrategy) ShouldBidBasedOnUsage(
	context.Context, BidStrategyRequest, model.ResourceUsageData) (BidStrategyResponse, error) {
	return BidStrategyResponse{ShouldBid: s.Response, ShouldWait: s.Wait}, nil
}
