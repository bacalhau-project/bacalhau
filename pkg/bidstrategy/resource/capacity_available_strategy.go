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

func NewAvailableCapacityStrategy(ctx context.Context, params AvailableCapacityStrategyParams) *AvailableCapacityStrategy {
	s := &AvailableCapacityStrategy{
		runningCapacityTracker:  params.RunningCapacityTracker,
		enqueuedCapacityTracker: params.EnqueuedCapacityTracker,
	}
	return s
}

func (s *AvailableCapacityStrategy) ShouldBidBasedOnUsage(
	ctx context.Context, request bidstrategy.BidStrategyRequest, usage models.Resources) (bidstrategy.BidStrategyResponse, error) {
	// skip bidding if we don't have enough capacity available
	runningCapacity := s.runningCapacityTracker.GetAvailableCapacity(ctx)
	enqueuedCapacity := s.enqueuedCapacityTracker.GetAvailableCapacity(ctx)
	totalCapacity := runningCapacity.Add(enqueuedCapacity)
	if !usage.LessThanEq(*totalCapacity) {
		return bidstrategy.BidStrategyResponse{
			ShouldBid: false,
			Reason:    fmt.Sprintf("not enough capacity available. Total: %s Requested: %s", totalCapacity, &usage),
		}, nil
	}

	return bidstrategy.NewShouldBidResponse(), nil
}

// compile-time interface check
var _ bidstrategy.ResourceBidStrategy = (*AvailableCapacityStrategy)(nil)
