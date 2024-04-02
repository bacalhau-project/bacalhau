package retry

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

type FixedStrategyParams struct {
	ShouldRetry bool
}
type FixedStrategy struct {
	shouldRetry bool
}

func NewFixedStrategy(params FixedStrategyParams) *FixedStrategy {
	return &FixedStrategy{shouldRetry: params.ShouldRetry}
}

func (s *FixedStrategy) ShouldRetry(ctx context.Context, request orchestrator.RetryRequest) bool {
	return s.shouldRetry
}

// compile-time interface checks
var _ orchestrator.RetryStrategy = (*FixedStrategy)(nil)
