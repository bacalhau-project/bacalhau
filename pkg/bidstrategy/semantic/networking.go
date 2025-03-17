package semantic

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
)

type NetworkingStrategy struct {
	Reject bool
}

var _ bidstrategy.SemanticBidStrategy = (*NetworkingStrategy)(nil)

func NewNetworkingStrategy(reject bool) *NetworkingStrategy {
	return &NetworkingStrategy{reject}
}

const (
	accessReason    = "run jobs that require network access"
	localOnlyReason = "run jobs that do not require network access"
)

// ShouldBid implements BidStrategy
func (s *NetworkingStrategy) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	if request.Job.Task().Network.Disabled() {
		return bidstrategy.NewBidResponse(true, localOnlyReason), nil
	}

	return bidstrategy.NewBidResponse(!s.Reject, accessReason), nil
}
