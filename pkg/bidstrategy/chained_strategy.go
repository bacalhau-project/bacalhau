package bidstrategy

import (
	"context"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

type ChainedBidStrategy struct {
	Strategies []BidStrategy
}

func NewChainedBidStrategy(strategies ...BidStrategy) *ChainedBidStrategy {
	return &ChainedBidStrategy{Strategies: strategies}
}

// AddStrategy Add new strategy to the end of the chain
func (c *ChainedBidStrategy) AddStrategy(strategy BidStrategy) {
	c.Strategies = append(c.Strategies, strategy)
}

// ShouldBid Iterate over all strategies, and return shouldBid if no error is thrown
// and none of the strategies return should not bid.
func (c *ChainedBidStrategy) ShouldBid(ctx context.Context, request BidStrategyRequest) (BidStrategyResponse, error) {
	return c.delegate(ctx, func(strategy BidStrategy) (BidStrategyResponse, error) {
		return strategy.ShouldBid(ctx, request)
	})
}

// ShouldBidBasedOnUsage Iterate over all strategies, and return shouldBid if no error is thrown
// and none of the strategies return should not bid.
func (c *ChainedBidStrategy) ShouldBidBasedOnUsage(
	ctx context.Context, request BidStrategyRequest, usage model.ResourceUsageData) (BidStrategyResponse, error) {
	return c.delegate(ctx, func(strategy BidStrategy) (BidStrategyResponse, error) {
		return strategy.ShouldBidBasedOnUsage(ctx, request, usage)
	})
}

func (c *ChainedBidStrategy) delegate(
	ctx context.Context,
	f func(strategy BidStrategy) (BidStrategyResponse, error),
) (BidStrategyResponse, error) {
	for _, strategy := range c.Strategies {
		response, err := f(strategy)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("error asking bidding strategy %s if we should bid",
				reflect.TypeOf(strategy).String())
			return BidStrategyResponse{}, err
		}
		var status string
		if response.ShouldWait {
			status = "should wait"
		} else if !response.ShouldBid {
			status = "should not bid"
		}
		if status != "" {
			log.Ctx(ctx).Debug().Msgf("bidding strategy %s returned %s due to: %s",
				reflect.TypeOf(strategy).String(), status, response.Reason)
			return response, nil
		}
	}

	return NewShouldBidResponse(), nil
}

// Compile-time check to ensure ChainedBidStrategy implements the BidStrategy interface.
var _ BidStrategy = (*ChainedBidStrategy)(nil)
