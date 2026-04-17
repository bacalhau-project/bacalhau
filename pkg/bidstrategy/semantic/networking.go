package semantic

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/models"
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
	undefinedReason = "run jobs with undefined network access"
)

// ShouldBid implements BidStrategy
func (s *NetworkingStrategy) ShouldBid(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	if request.Job.Task().Network.Disabled() {
		return bidstrategy.NewBidResponse(true, localOnlyReason), nil
	}
	if request.Job.Task().Network.Type == models.NetworkDefault {
		return bidstrategy.NewBidResponse(true, undefinedReason), nil
	}

	return bidstrategy.NewBidResponse(!s.Reject, accessReason), nil
}
