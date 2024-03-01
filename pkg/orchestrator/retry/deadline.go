package retry

import (
	"context"
	"math"
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

func (s *DeadlineStrategy) ShouldRetry(ctx context.Context, request orchestrator.RetryRequest) (bool, time.Duration) {
	policy := request.Job.ReschedulingPolicy

	// Retry if the job's scheduling deadline is in the future
	timeoutAsDuration := policy.SchedulingTimeout * int64(time.Second)
	deadline := request.Job.CreateTime + timeoutAsDuration
	now := time.Now().UTC().UnixNano()

	secondsUntilDeadline := (deadline - now) / int64(time.Second) // -ve if deadline is past

	if secondsUntilDeadline > 0 {
		// delay = RetryDelay * RetryDelayGrowthFactor ^ failures
		// ...but never more than MaximumRetryDelay
		delay := int64(math.Round(math.Min(
			float64(policy.BaseRetryDelay)*
				math.Pow(policy.RetryDelayGrowthFactor,
					float64(request.PastFailures)),
			float64(policy.MaximumRetryDelay))))

		if secondsUntilDeadline > delay {
			log.Debug().Msgf("Deferring job execution for %d seconds (%d * %f ^ %d max %d) as deadline is %d seconds away",
				delay,
				policy.BaseRetryDelay,
				policy.RetryDelayGrowthFactor,
				request.PastFailures,
				policy.MaximumRetryDelay,
				secondsUntilDeadline)
			return true, time.Duration(delay * int64(time.Second))
		} else {
			log.Debug().Msgf("Aborting job because a delay of %d seconds (%d * %f ^ %d max %d) would exceed the deadline of %d seconds",
				delay,
				policy.BaseRetryDelay,
				policy.RetryDelayGrowthFactor,
				request.PastFailures,
				policy.MaximumRetryDelay,
				secondsUntilDeadline)
			return false, 0
		}
	} else {
		log.Debug().Msgf("Aborting job because deadline is %d seconds ago", -secondsUntilDeadline)
		return false, 0
	}
}

// compile-time interface checks
var _ orchestrator.RetryStrategy = (*DeadlineStrategy)(nil)
