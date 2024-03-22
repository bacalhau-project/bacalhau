package resource

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type AvailableCapacityStrategyParams struct {
	RunningCapacityTracker  capacity.Tracker
	EnqueuedCapacityTracker capacity.Tracker
}

type AvailableCapacityStrategy struct {
	runningCapacityTracker  capacity.Tracker
	enqueuedCapacityTracker capacity.Tracker
}

func NewAvailableCapacityStrategy(params AvailableCapacityStrategyParams) *AvailableCapacityStrategy {
	s := &AvailableCapacityStrategy{
		runningCapacityTracker:  params.RunningCapacityTracker,
		enqueuedCapacityTracker: params.EnqueuedCapacityTracker,
	}
	return s
}

func (s *AvailableCapacityStrategy) ShouldBidBasedOnUsage(
	ctx context.Context, request bidstrategy.BidStrategyRequest, usage models.Resources) (bidstrategy.BidStrategyResponse, error) {
	runningCapacity := s.runningCapacityTracker.GetAvailableCapacity(ctx)
	enqueuedCapacity := s.enqueuedCapacityTracker.GetAvailableCapacity(ctx)
	totalCapacity := runningCapacity.Add(enqueuedCapacity)
	if usage.LessThanEq(*totalCapacity) {
		return bidstrategy.BidStrategyResponse{
			ShouldBid:  true,
			ShouldWait: false,
			Reason:     "",
		}, nil
	}
	return bidstrategy.BidStrategyResponse{
		ShouldBid:  false,
		ShouldWait: false,
		Reason:     fmt.Sprintf("insuffucuent capacity - requested: %s, available: %s", usage.String(), totalCapacity.String()),
	}, nil
}

// compile-time interface check
var _ bidstrategy.ResourceBidStrategy = (*AvailableCapacityStrategy)(nil)
