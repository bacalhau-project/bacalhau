package bidstrategy

import (
	"context"
	"errors"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/model"
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
	ctx context.Context, f func(strategy BidStrategy) (BidStrategyResponse, error)) (BidStrategyResponse, error) {
	if c.Strategies == nil {
		return BidStrategyResponse{}, errors.New("no strategies registered")
	}
	for _, strategy := range c.Strategies {
		response, err := f(strategy)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("error asking bidding strategy %s if we should bid",
				reflect.TypeOf(strategy).String())
			return BidStrategyResponse{}, err
		}
		if !response.ShouldBid {
			log.Ctx(ctx).Debug().Msgf("bidding strategy %s returned should not bid due to: %s",
				reflect.TypeOf(strategy).String(), response.Reason)
			return response, nil
		}
	}

	return newShouldBidResponse(), nil
}

// Compile-time check to ensure ChainedBidStrategy implements the BidStrategy interface.
var _ BidStrategy = (*ChainedBidStrategy)(nil)
