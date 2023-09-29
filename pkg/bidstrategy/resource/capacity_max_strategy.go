package resource

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type MaxCapacityStrategyParams struct {
	MaxJobRequirements models.Resources
}

type MaxCapacityStrategy struct {
	maxJobRequirements models.Resources
}

func NewMaxCapacityStrategy(params MaxCapacityStrategyParams) *MaxCapacityStrategy {
	return &MaxCapacityStrategy{
		maxJobRequirements: params.MaxJobRequirements,
	}
}

func (s *MaxCapacityStrategy) ShouldBidBasedOnUsage(
	ctx context.Context, request bidstrategy.BidStrategyRequest, usage models.Resources) (bidstrategy.BidStrategyResponse, error) {
	// skip bidding if we don't have enough capacity available
	if !usage.LessThanEq(s.maxJobRequirements) {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    "job requirements exceed max allowed per job",
		}, nil
	}

	return bidstrategy.NewShouldBidResponse(), nil
}

// compile-time interface check
var _ bidstrategy.ResourceBidStrategy = (*MaxCapacityStrategy)(nil)
