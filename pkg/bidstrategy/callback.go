package bidstrategy

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type CallbackBidStrategy struct {
	OnShouldBid             func(context.Context, BidStrategyRequest) (BidStrategyResponse, error)
	OnShouldBidBasedOnUsage func(context.Context, BidStrategyRequest, model.ResourceUsageData) (BidStrategyResponse, error)
}

// ShouldBid implements BidStrategy
func (s *CallbackBidStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	return s.OnShouldBid(ctx, request)
}

// ShouldBidBasedOnUsage implements BidStrategy
func (s *CallbackBidStrategy) ShouldBidBasedOnUsage(
	ctx context.Context,
	request BidStrategyRequest,
	resourceUsage model.ResourceUsageData,
) (BidStrategyResponse, error) {
	return s.OnShouldBidBasedOnUsage(ctx, request, resourceUsage)
}

var _ BidStrategy = (*CallbackBidStrategy)(nil)
