package bidstrategy

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type MaxCapacityStrategyParams struct {
	MaxJobRequirements model.ResourceUsageData
}

type MaxCapacityStrategy struct {
	maxJobRequirements model.ResourceUsageData
}

func NewMaxCapacityStrategy(params MaxCapacityStrategyParams) *MaxCapacityStrategy {
	return &MaxCapacityStrategy{
		maxJobRequirements: params.MaxJobRequirements,
	}
}

func (s *MaxCapacityStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	return newShouldBidResponse(), nil
}

func (s *MaxCapacityStrategy) ShouldBidBasedOnUsage(
	ctx context.Context, request BidStrategyRequest, usage model.ResourceUsageData) (BidStrategyResponse, error) {
	// skip bidding if we don't have enough capacity available
	if !usage.LessThanEq(s.maxJobRequirements) {
		return BidStrategyResponse{
			ShouldBid: false,
			Reason:    "job requirements exceed max allowed per job",
		}, nil
	}

	return newShouldBidResponse(), nil
}
