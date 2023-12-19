package retry

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog/log"
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
	log.Debug().Msgf("ABS DEBUG: Got into FixedStrategy somehow?!")
	return s.shouldRetry
}

// compile-time interface checks
var _ orchestrator.RetryStrategy = (*FixedStrategy)(nil)
