package retry

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

type DeadlineStrategyParams struct {
}
type DeadlineStrategy struct {
}

func NewDeadlineStrategy(params DeadlineStrategyParams) *DeadlineStrategy {
	return &DeadlineStrategy{}
}

func (s *DeadlineStrategy) ShouldRetry(ctx context.Context, request orchestrator.RetryRequest) bool {
	// Retry if the job's scheduling deadline is in the future
	timeoutAsDuration := request.Job.SchedulingTimeout * int64(time.Second)
	deadline := request.Job.CreateTime + timeoutAsDuration
	now := time.Now().UTC().UnixNano()
	return deadline > now
}

// compile-time interface checks
var _ orchestrator.RetryStrategy = (*DeadlineStrategy)(nil)
