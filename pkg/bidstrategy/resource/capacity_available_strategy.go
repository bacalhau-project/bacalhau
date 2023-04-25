package resource

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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

func (s *AvailableCapacityStrategy) ShouldBidBasedOnUsage(
	ctx context.Context, request bidstrategy.BidStrategyRequest, usage model.ResourceUsageData) (bidstrategy.BidStrategyResponse, error) {
	// skip bidding if we don't have enough capacity available
	availableCapacity := s.runningCapacityTracker.GetAvailableCapacity(ctx).Add(s.enqueuedCapacityTracker.GetAvailableCapacity(ctx))
	if !usage.LessThanEq(availableCapacity) {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    "not enough capacity available",
		}, nil
	}

	return bidstrategy.NewShouldBidResponse(), nil
}

// compile-time interface check
var _ bidstrategy.ResourceBidStrategy = (*AvailableCapacityStrategy)(nil)
