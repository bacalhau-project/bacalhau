package resource

import (
	"context"
	"fmt"

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
	if usage.LessThanEq(s.maxJobRequirements) {
		return bidstrategy.BidStrategyResponse{
			ShouldBid:  true,
			ShouldWait: false,
			Reason:     "",
		}, nil
	}
	return bidstrategy.BidStrategyResponse{
		ShouldBid:  false,
		ShouldWait: false,
		Reason:     fmt.Sprintf("insufficient resources - requested: %s, available: %s", usage.String(), s.maxJobRequirements.String()),
	}, nil
}

// compile-time interface check
var _ bidstrategy.ResourceBidStrategy = (*MaxCapacityStrategy)(nil)
