package retry

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog/log"
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
	timeoutAsDuration := request.Job.SchedulingTimeout*int64(time.Second)
	deadline := request.Job.CreateTime + timeoutAsDuration
	now := time.Now().UTC().UnixNano()
	log.Debug().Msgf("ABS DEBUG: DeadlineStrategy %#v %#v %#v %#v %#v %#v",
		request.Job.SchedulingTimeout, request.Job.CreateTime,
		timeoutAsDuration, deadline,
		now, deadline > now)
	return deadline > now
}

// compile-time interface checks
var _ orchestrator.RetryStrategy = (*DeadlineStrategy)(nil)
