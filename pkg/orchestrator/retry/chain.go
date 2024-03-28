package retry

import (
	"context"
	"reflect"
	"time"

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

func (c *Chain) ShouldRetry(ctx context.Context, request orchestrator.RetryRequest) (bool, time.Duration) {
	doRetry := false
	finalDelay := time.Duration(0)
	for _, strategy := range c.strategies {
		shouldRetry, delay := strategy.ShouldRetry(ctx, request)
		if !shouldRetry {
			log.Ctx(ctx).Debug().Msgf("retry strategy %s decided not to retry", reflect.TypeOf(strategy).String())
			return false, 0
		}
		if shouldRetry {
			log.Ctx(ctx).Debug().Msgf("retry strategy %s decided okay to retry in %s", reflect.TypeOf(strategy).String(), delay.String())
			doRetry = true
			finalDelay = delay
		}
	}
	return doRetry, finalDelay
}
