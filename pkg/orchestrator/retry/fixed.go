package retry

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

type FixedStrategyParams struct {
	ShouldRetry bool
	Delay       time.Duration
}
type FixedStrategy struct {
	shouldRetry bool
	delay       time.Duration
}

func NewFixedStrategy(params FixedStrategyParams) *FixedStrategy {
	return &FixedStrategy{
		shouldRetry: params.ShouldRetry,
		delay:       params.Delay,
	}
}

func (s *FixedStrategy) ShouldRetry(ctx context.Context, request orchestrator.RetryRequest) (bool, time.Duration) {
	return s.shouldRetry, s.delay
}

// compile-time interface checks
var _ orchestrator.RetryStrategy = (*FixedStrategy)(nil)
