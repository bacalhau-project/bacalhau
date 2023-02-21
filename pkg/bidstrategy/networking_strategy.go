package bidstrategy

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type NetworkingStrategy struct {
	Accept bool
}

func NewNetworkingStrategy(accept bool) *NetworkingStrategy {
	return &NetworkingStrategy{accept}
}

// ShouldBid implements BidStrategy
func (s *NetworkingStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	shouldBid := s.Accept || request.Job.Spec.Network.Disabled()
	return BidStrategyResponse{
		ShouldBid: shouldBid,
		Reason:    fmt.Sprintf("networking is enabled: %t", s.Accept),
	}, nil
}

// ShouldBidBasedOnUsage implements BidStrategy
func (s *NetworkingStrategy) ShouldBidBasedOnUsage(
	ctx context.Context,
	request BidStrategyRequest,
	resourceUsage model.ResourceUsageData,
) (BidStrategyResponse, error) {
	return s.ShouldBid(ctx, request)
}

var _ BidStrategy = (*NetworkingStrategy)(nil)
