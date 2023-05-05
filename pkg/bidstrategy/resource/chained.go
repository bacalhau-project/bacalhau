package resource

import (
	"context"
	"reflect"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type ChainedResourceBidStrategy struct {
	Strategies []bidstrategy.ResourceBidStrategy
}

func NewChainedResourceBidStrategy(strategies ...bidstrategy.ResourceBidStrategy) *ChainedResourceBidStrategy {
	return &ChainedResourceBidStrategy{Strategies: strategies}
}

// AddStrategy Add new strategy to the end of the chain
func (c *ChainedResourceBidStrategy) AddStrategy(strategy bidstrategy.ResourceBidStrategy) {
	c.Strategies = append(c.Strategies, strategy)
}

// ShouldBidBasedOnUsage Iterate over all strategies, and return shouldBid if no error is thrown
// and none of the strategies return should not bid.
func (c *ChainedResourceBidStrategy) ShouldBidBasedOnUsage(
	ctx context.Context, request bidstrategy.BidStrategyRequest, usage model.ResourceUsageData) (bidstrategy.BidStrategyResponse, error) {
	return c.delegate(ctx, func(strategy bidstrategy.ResourceBidStrategy) (bidstrategy.BidStrategyResponse, error) {
		return strategy.ShouldBidBasedOnUsage(ctx, request, usage)
	})
}

func (c *ChainedResourceBidStrategy) delegate(
	ctx context.Context,
	f func(strategy bidstrategy.ResourceBidStrategy) (bidstrategy.BidStrategyResponse, error),
) (bidstrategy.BidStrategyResponse, error) {
	for _, strategy := range c.Strategies {
		response, err := f(strategy)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("error asking bidding strategy %s if we should bid",
				reflect.TypeOf(strategy).String())
			return bidstrategy.BidStrategyResponse{}, err
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

	return bidstrategy.BidStrategyResponse{ShouldBid: true}, nil
}
