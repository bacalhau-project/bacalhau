package bidstrategy

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/bidstrategy"
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

func (s *MaxCapacityStrategy) ShouldBid(context.Context, bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	return bidstrategy.NewShouldBidResponse(), nil
}

func (s *MaxCapacityStrategy) ShouldBidBasedOnUsage(
	ctx context.Context, request bidstrategy.BidStrategyRequest, usage model.ResourceUsageData) (bidstrategy.BidStrategyResponse, error) {
	// skip bidding if we don't have enough capacity available
	if !usage.LessThanEq(s.maxJobRequirements) {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    "job requirements exceed max allowed per job",
		}, nil
	}

	return bidstrategy.NewShouldBidResponse(), nil
}
