package moderation

import (
	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
)

func asBidStrategyResponse(resp *types.Moderation) *bidstrategy.BidStrategyResponse {
	return &bidstrategy.BidStrategyResponse{
		ShouldBid:  resp.Status,
		ShouldWait: false,
		Reason:     resp.Notes,
	}
}
