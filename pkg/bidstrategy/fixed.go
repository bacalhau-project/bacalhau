package bidstrategy

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// FixedBidStrategy is a bid strategy that always returns the same response, which is useful for testing
func NewFixedBidStrategy(response, wait bool) *CallbackBidStrategy {
	return &CallbackBidStrategy{
		OnShouldBid: func(_ context.Context, _ BidStrategyRequest) (BidStrategyResponse, error) {
			return BidStrategyResponse{ShouldBid: response, ShouldWait: wait}, nil
		},
		OnShouldBidBasedOnUsage: func(
			context.Context, BidStrategyRequest, models.Resources) (BidStrategyResponse, error) {
			return BidStrategyResponse{ShouldBid: response, ShouldWait: wait}, nil
		},
	}
}
