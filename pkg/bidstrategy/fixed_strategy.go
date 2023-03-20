package bidstrategy

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// FixedBidStrategy is a bid strategy that always returns the same response, which is useful for testing
type FixedBidStrategy struct {
	response bool
}

// NewFixedBidStrategy creates a new FixedBidStrategy
func NewFixedBidStrategy(response bool) *FixedBidStrategy {
	return &FixedBidStrategy{
		response: response,
	}
}

func (s *FixedBidStrategy) ShouldBid(_ context.Context, _ BidStrategyRequest) (BidStrategyResponse, error) {
	return BidStrategyResponse{ShouldBid: s.response}, nil
}

func (s *FixedBidStrategy) ShouldBidBasedOnUsage(
	context.Context, BidStrategyRequest, model.ResourceUsageData) (BidStrategyResponse, error) {
	return BidStrategyResponse{ShouldBid: s.response}, nil
}
