package semantic

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
)

type NetworkingStrategy struct {
	Accept bool
}

var _ bidstrategy.SemanticBidStrategy = (*NetworkingStrategy)(nil)

func NewNetworkingStrategy(accept bool) *NetworkingStrategy {
	return &NetworkingStrategy{accept}
}

// ShouldBid implements BidStrategy
func (s *NetworkingStrategy) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	shouldBid := s.Accept || request.Job.Spec.Network.Disabled()
	return bidstrategy.BidStrategyResponse{
		ShouldBid: shouldBid,
		Reason:    fmt.Sprintf("networking is enabled: %t", s.Accept),
	}, nil
}
