package bidstrategy

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
)

type AvailableCapacityStrategyParams struct {
	CapacityManager capacitymanager.CapacityManager
}

type AvailableCapacityStrategy struct {
	capacityManager capacitymanager.CapacityManager
}

func NewAvailableCapacityStrategy(params AvailableCapacityStrategyParams) *AvailableCapacityStrategy {
	return &AvailableCapacityStrategy{
		capacityManager: params.CapacityManager,
	}
}

func (s *AvailableCapacityStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	// skip bidding if we don't have enough capacity available
	withinCapacityLimits := s.capacityManager.FilterRequirements(request.ResourceUsageRequirements)
	if !withinCapacityLimits {
		return BidStrategyResponse{
			ShouldBid: false,
			Reason:    "not enough capacity available",
		}, nil
	}

	return newShouldBidResponse(), nil
}
