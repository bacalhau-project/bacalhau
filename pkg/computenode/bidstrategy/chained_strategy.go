package bidstrategy

import (
	"context"
	"errors"
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
	if c.Strategies == nil {
		return BidStrategyResponse{}, errors.New("no strategies registered")
	}

	// All strategies are called, unless one of them returns an error, or a strategy returns should not bid.
	for _, strategy := range c.Strategies {
		response, err := strategy.ShouldBid(ctx, request)
		if err != nil || !response.ShouldBid {
			return response, err
		}
	}

	return newShouldBidResponse(), nil
}

// Compile-time check to ensure ChainedBidStrategy implements the BidStrategy interface.
var _ BidStrategy = (*ChainedBidStrategy)(nil)
