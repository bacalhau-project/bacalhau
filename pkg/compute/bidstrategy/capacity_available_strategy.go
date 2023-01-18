package bidstrategy

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

type AvailableCapacityStrategyParams struct {
	CapacityTracker capacity.Tracker
	CommitFactor    float64
}

type AvailableCapacityStrategy struct {
	capacityTracker capacity.Tracker
	commitFactor    float64
}

func NewAvailableCapacityStrategy(ctx context.Context, params AvailableCapacityStrategyParams) *AvailableCapacityStrategy {
	s := &AvailableCapacityStrategy{
		capacityTracker: params.CapacityTracker,
		commitFactor:    params.CommitFactor,
	}
	log.Info().Msgf("Compute node configured with total capacity of %s, and over commit factor of %f",
		s.capacityTracker.GetMaxCapacity(ctx), s.commitFactor)
	return s
}

func (s *AvailableCapacityStrategy) ShouldBid(
	ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	return newShouldBidResponse(), nil
}

func (s *AvailableCapacityStrategy) ShouldBidBasedOnUsage(
	ctx context.Context, request BidStrategyRequest, usage model.ResourceUsageData) (BidStrategyResponse, error) {
	// skip bidding if we don't have enough capacity available
	availableCapacity := s.capacityTracker.GetAvailableCapacity(ctx).Multi(s.commitFactor)
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
