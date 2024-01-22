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

const resourceReason = "run jobs that require this many resources (%s requested but only %s is allowed)"

func (s *MaxCapacityStrategy) ShouldBidBasedOnUsage(
	ctx context.Context, request bidstrategy.BidStrategyRequest, usage models.Resources) (bidstrategy.BidStrategyResponse, error) {
	// skip bidding if we don't have enough capacity available
	return bidstrategy.NewBidResponse(usage.LessThanEq(s.maxJobRequirements), resourceReason, usage, s.maxJobRequirements), nil
}

// compile-time interface check
var _ bidstrategy.ResourceBidStrategy = (*MaxCapacityStrategy)(nil)
