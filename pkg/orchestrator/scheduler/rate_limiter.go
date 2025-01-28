package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// ExecutionRateLimiter provides rate limiting functionality for job executions
type ExecutionRateLimiter interface {
	// Apply checks if the number of executions exceeds the limit and optionally creates
	// a delayed evaluation for remaining executions. Returns the number of executions
	// that should be created in this evaluation.
	Apply(ctx context.Context, plan *models.Plan, totalNeeded int) int
}

// NoopRateLimiter is a rate limiter that does not impose any limits
type NoopRateLimiter struct{}

func NewNoopRateLimiter() *NoopRateLimiter {
	return &NoopRateLimiter{}
}

func (n *NoopRateLimiter) Apply(ctx context.Context, plan *models.Plan, totalNeeded int) int {
	return totalNeeded
}

// BatchRateLimiter limits the number of executions that can be created in a single
// evaluation and creates delayed evaluations for remaining executions
type BatchRateLimiter struct {
	maxExecutionsPerEval  int
	executionLimitBackoff time.Duration
	clock                 clock.Clock
}

type BatchRateLimiterParams struct {
	// MaxExecutionsPerEval limits the number of new executions that can be created in a single evaluation
	MaxExecutionsPerEval int
	// ExecutionLimitBackoff is the duration to wait before creating a new evaluation when hitting execution limits
	ExecutionLimitBackoff time.Duration
	// Clock is used for time-based operations. If not provided, system clock is used.
	Clock clock.Clock
}

func NewBatchRateLimiter(params BatchRateLimiterParams) *BatchRateLimiter {
	if params.Clock == nil {
		params.Clock = clock.New()
	}
	return &BatchRateLimiter{
		maxExecutionsPerEval:  params.MaxExecutionsPerEval,
		executionLimitBackoff: params.ExecutionLimitBackoff,
		clock:                 params.Clock,
	}
}

func (b *BatchRateLimiter) Apply(ctx context.Context, plan *models.Plan, totalNeeded int) int {
	if totalNeeded <= b.maxExecutionsPerEval {
		return totalNeeded
	}

	if b.maxExecutionsPerEval <= 0 {
		return totalNeeded
	}

	// Create delayed evaluation for remaining executions
	comment := fmt.Sprintf("Creating delayed evaluation to schedule remaining %d executions after reaching limit of %d",
		totalNeeded-b.maxExecutionsPerEval, b.maxExecutionsPerEval)

	waitUntil := b.clock.Now().Add(b.executionLimitBackoff)
	delayedEvaluation := plan.Eval.NewDelayedEvaluation(waitUntil).
		WithTriggeredBy(models.EvalTriggerExecutionLimit).
		WithComment(comment)
	plan.AppendEvaluation(delayedEvaluation)

	log.Ctx(ctx).Debug().Msg(comment)

	return b.maxExecutionsPerEval
}
