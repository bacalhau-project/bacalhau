package bidstrategy

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type AvailableCapacityStrategyParams struct {
	RunningCapacityTracker  capacity.Tracker
	EnqueuedCapacityTracker capacity.Tracker
}

type AvailableCapacityStrategy struct {
	runningCapacityTracker  capacity.Tracker
	enqueuedCapacityTracker capacity.Tracker
}

func NewAvailableCapacityStrategy(ctx context.Context, params AvailableCapacityStrategyParams) *AvailableCapacityStrategy {
	s := &AvailableCapacityStrategy{
		runningCapacityTracker:  params.RunningCapacityTracker,
		enqueuedCapacityTracker: params.EnqueuedCapacityTracker,
	}
	return s
}

func (s *AvailableCapacityStrategy) ShouldBid(
	ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	return newShouldBidResponse(), nil
}

func (s *AvailableCapacityStrategy) ShouldBidBasedOnUsage(
	ctx context.Context, request BidStrategyRequest, usage model.ResourceUsageData) (BidStrategyResponse, error) {
	// skip bidding if we don't have enough capacity available
	availableCapacity := s.runningCapacityTracker.GetAvailableCapacity(ctx).Add(s.enqueuedCapacityTracker.GetAvailableCapacity(ctx))
	if !usage.LessThanEq(availableCapacity) {
		return BidStrategyResponse{
			ShouldBid: false,
			Reason:    "not enough capacity available",
		}, nil
	}

	return newShouldBidResponse(), nil
}

// compile-time interface check
var _ BidStrategy = (*AvailableCapacityStrategy)(nil)
