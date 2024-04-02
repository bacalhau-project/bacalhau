package retry

import (
	"context"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog/log"
)

type Chain struct {
	strategies []orchestrator.RetryStrategy
}

func NewChain() *Chain {
	return &Chain{}
}

func (c *Chain) Add(strategies ...orchestrator.RetryStrategy) {
	c.strategies = append(c.strategies, strategies...)
}

func (c *Chain) ShouldRetry(ctx context.Context, request orchestrator.RetryRequest) bool {
	doRetry := false
	for _, strategy := range c.strategies {
		shouldRetry := strategy.ShouldRetry(ctx, request)
		if !shouldRetry {
			log.Ctx(ctx).Debug().Msgf("retry strategy %s decided not to retry", reflect.TypeOf(strategy).String())
			return false
		}
		if shouldRetry {
			log.Ctx(ctx).Debug().Msgf("retry strategy %s decided okay to retry", reflect.TypeOf(strategy).String())
			doRetry = true
		}
	}
	return doRetry
}
