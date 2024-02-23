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
	policy := request.Job.ReschedulingPolicy

	// Retry if the job's scheduling deadline is in the future
	timeoutAsDuration := policy.SchedulingTimeout * int64(time.Second)
	deadline := request.Job.CreateTime + timeoutAsDuration
	now := time.Now().UTC().UnixNano()

	if deadline > now {
		log.Debug().Msgf("Retrying job because %d second deadline %d is in the future (from %d)", policy.SchedulingTimeout, deadline, now)
	} else {
		log.Debug().Msgf("Aborting job because %d second deadline %d is in the future (from %d)", policy.SchedulingTimeout, deadline, now)
	}
	return deadline > now
}

// compile-time interface checks
var _ orchestrator.RetryStrategy = (*DeadlineStrategy)(nil)
