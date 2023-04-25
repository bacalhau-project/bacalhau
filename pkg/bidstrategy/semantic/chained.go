package semantic

import (
	"context"
	"reflect"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
)

type ChainedBidStrategy struct {
	Strategies []bidstrategy.SemanticBidStrategy
}

func NewChainedSemanticBidStrategy(strategies ...bidstrategy.SemanticBidStrategy) *ChainedBidStrategy {
	return &ChainedBidStrategy{Strategies: strategies}
}

// AddStrategy Add new strategy to the end of the chain
func (c *ChainedBidStrategy) AddStrategy(strategy bidstrategy.SemanticBidStrategy) {
	c.Strategies = append(c.Strategies, strategy)
}

// ShouldBid Iterate over all strategies, and return shouldBid if no error is thrown
// and none of the strategies return should not bid.
func (c *ChainedBidStrategy) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	return c.delegate(ctx, func(strategy bidstrategy.SemanticBidStrategy) (bidstrategy.BidStrategyResponse, error) {
		return strategy.ShouldBid(ctx, request)
	})
}

func (c *ChainedBidStrategy) delegate(
	ctx context.Context,
	f func(strategy bidstrategy.SemanticBidStrategy) (bidstrategy.BidStrategyResponse, error),
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
