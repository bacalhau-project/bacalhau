package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

type BatchServiceJobSchedulerParams struct {
	JobStore      jobstore.Store
	Planner       orchestrator.Planner
	NodeSelector  orchestrator.NodeSelector
	RetryStrategy orchestrator.RetryStrategy
	QueueBackoff  time.Duration
	// Clock is the clock used for time-based operations.
	// If not provided, the system clock is used.
	Clock clock.Clock
}

// BatchServiceJobScheduler is a scheduler for:
// - batch jobs that run until completion on N number of nodes
// - service jobs than run until stopped on N number of nodes
type BatchServiceJobScheduler struct {
	jobStore      jobstore.Store
	planner       orchestrator.Planner
	selector      orchestrator.NodeSelector
	retryStrategy orchestrator.RetryStrategy
	queueBackoff  time.Duration
	clock         clock.Clock
}

func NewBatchServiceJobScheduler(params BatchServiceJobSchedulerParams) *BatchServiceJobScheduler {
	if params.Clock == nil {
		params.Clock = clock.New()
	}
	return &BatchServiceJobScheduler{
		jobStore:      params.JobStore,
		planner:       params.Planner,
		selector:      params.NodeSelector,
		retryStrategy: params.RetryStrategy,
		queueBackoff:  params.QueueBackoff,
		clock:         params.Clock,
	}
}

func (b *BatchServiceJobScheduler) Process(ctx context.Context, evaluation *models.Evaluation) error {
	ctx = log.Ctx(ctx).With().Str("JobID", evaluation.JobID).Str("EvalID", evaluation.ID).Logger().WithContext(ctx)

	job, err := b.jobStore.GetJob(ctx, evaluation.JobID)
	if err != nil {
		return fmt.Errorf("failed to retrieve job %s: %w", evaluation.JobID, err)
	}
	// Retrieve the job state
	jobExecutions, err := b.jobStore.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID: evaluation.JobID,
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve job state for job %s when evaluating %s: %w",
			evaluation.JobID, evaluation, err)
	}

	// Plan to hold the actions to be taken
	plan := models.NewPlan(evaluation, &job)

	existingExecs := execSetFromSliceOfValues(jobExecutions)
	nonTerminalExecs := existingExecs.filterNonTerminal()

	// early exit if the job is stopped
	if job.IsTerminal() {
		nonTerminalExecs.markStopped(orchestrator.ExecStoppedByJobStopEvent(), plan)
		return b.planner.Process(ctx, plan)
	}

	// Retrieve the info for all the nodes that have executions for this job
	nodeInfos, err := existingNodeInfos(ctx, b.selector, nonTerminalExecs)
	if err != nil {
		return err
	}

	// keep track or existing failed executions, and those that will be marked as failed
	allFailedExecs := existingExecs.filterFailed()

	// Mark executions that are running on nodes that are not healthy as failed
	nonTerminalExecs, lost := nonTerminalExecs.filterByNodeHealth(nodeInfos)
	lost.markStopped(orchestrator.ExecStoppedByNodeUnhealthyEvent(), plan)
	allFailedExecs = allFailedExecs.union(lost)

	nonTerminalExecs, allFailedExecs = b.handleTimeouts(plan, nonTerminalExecs, allFailedExecs)

	// Calculate remaining job count
	// Service jobs run until the user stops the job, and would be a bug if an execution is marked completed. So the desired
	// remaining count equals the count specified in the job spec.
	// Batch jobs on the other hand run until completion and the desired remaining count excludes the completed executions
	desiredRemainingCount := job.Count
	if job.Type == models.JobTypeBatch {
		desiredRemainingCount = math.Max(0, job.Count-existingExecs.countCompleted())
	}

	// Approve/Reject nodes
	execsByApprovalStatus := nonTerminalExecs.filterByApprovalStatus(desiredRemainingCount)
	execsByApprovalStatus.toApprove.markApproved(plan)
	execsByApprovalStatus.toReject.markStopped(orchestrator.ExecStoppedByNodeRejectedEvent(), plan)

	// create new executions if needed
	remainingExecutionCount := desiredRemainingCount - execsByApprovalStatus.activeCount()
	if remainingExecutionCount > 0 {
		// first, check if there were any failed executions and if we should retry
		if len(allFailedExecs) > 0 && !b.retryStrategy.ShouldRetry(ctx, orchestrator.RetryRequest{JobID: job.ID}) {
			plan.MarkJobFailed(orchestrator.JobExhaustedRetriesEvent())
		}

		if !plan.IsJobFailed() {
			// find matching nodes for the remaining executions
			err = b.createMissingExecs(ctx, remainingExecutionCount, &job, plan)
			if err != nil {
				return err // internal error
			}
		}
	}

	// if the plan's job state if terminal, stop all active executions
	if plan.IsJobFailed() {
		nonTerminalExecs.markStopped(plan.Event, plan)
	} else {
		// stop executions if we over-subscribed and exceeded the desired number of executions
		_, overSubscriptions := execsByApprovalStatus.running.filterByOverSubscriptions(desiredRemainingCount)
		overSubscriptions.markStopped(orchestrator.ExecStoppedByOversubscriptionEvent(), plan)

		// Check the job's state and update it accordingly.
		if desiredRemainingCount <= 0 {
			// If there are no remaining tasks to be done, mark the job as completed.
			plan.MarkJobCompleted()
		}
		plan.MarkJobRunningIfEligible()
	}
	return b.planner.Process(ctx, plan)
}

func (b *BatchServiceJobScheduler) handleTimeouts(
	plan *models.Plan, nonTerminalExecs, allFailedExecs execSet) (execSet, execSet) {
	// Mark job/executions that have exceeded their total/execution timeout as failed
	// Only applicable for batch jobs and not service jobs.
	job := plan.Job
	if !job.IsLongRunning() {
		// check if job has exceeded total timeout
		jobTimeout := job.Task().Timeouts.GetTotalTimeout()
		jobExpirationTime := b.clock.Now().Add(-jobTimeout)
		if job.IsExpired(jobExpirationTime) {
			plan.MarkJobFailed(orchestrator.JobTimeoutEvent(jobTimeout))
		}

		// check if the executions have exceeded their execution timeout
		if !plan.IsJobFailed() && job.Task().Timeouts.GetExecutionTimeout() > 0 {
			timeout := job.Task().Timeouts.GetExecutionTimeout()
			expirationTime := b.clock.Now().Add(-timeout)
			var timedOut execSet
			nonTerminalExecs, timedOut = nonTerminalExecs.filterByExecutionTimeout(expirationTime)
			timedOut.markStopped(orchestrator.ExecStoppedByExecutionTimeoutEvent(timeout), plan)
			allFailedExecs = allFailedExecs.union(timedOut)
		}
	}
	return nonTerminalExecs, allFailedExecs
}

func (b *BatchServiceJobScheduler) createMissingExecs(
	ctx context.Context, remainingExecutionCount int, job *models.Job, plan *models.Plan) error {
	// find matching nodes for the job
	matching, rejected, err := b.selector.MatchingNodes(ctx, job)
	if err != nil {
		return err
	}

	// fail fast if there are not enough nodes, and we've passed job queue timeout
	if len(matching) < remainingExecutionCount {
		// check if we can retry scheduling the job at a later time if we don't have enough nodes
		timeout := job.Task().Timeouts.GetQueueTimeout()
		expirationTime := b.clock.Now().Add(-timeout)

		// TODO: we are calculating queue timeout based on job creation time, but we should probably
		//  calculate it based on the time the job stayed in the queue so that rescheduling the job
		//  would reset the queue timeout.
		if job.GetCreateTime().Before(expirationTime) {
			plan.MarkJobFailed(models.EventFromError(
				orchestrator.EventTopicJobScheduling,
				orchestrator.NewErrNotEnoughNodes(remainingExecutionCount, append(matching, rejected...))))
			return nil
		}
	}

	// create new executions
	for i := 0; i < min(len(matching), remainingExecutionCount); i++ {
		execution := &models.Execution{
			NodeID:       matching[i].NodeInfo.ID(),
			JobID:        job.ID,
			Job:          job,
			ID:           idgen.ExecutionIDPrefix + uuid.NewString(),
			EvalID:       plan.EvalID,
			Namespace:    job.Namespace,
			ComputeState: models.NewExecutionState(models.ExecutionStateNew),
			DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
		}
		execution.Normalize()
		plan.AppendExecution(execution)
	}

	if len(matching) < remainingExecutionCount {
		// create a delayed evaluation to retry scheduling the job if we don't have enough nodes,
		// and we haven't passed the queue timeout
		waitUntil := b.clock.Now().Add(b.queueBackoff)
		comment := orchestrator.NewErrNotEnoughNodes(remainingExecutionCount, append(matching, rejected...)).Error()
		delayedEvaluation := plan.Eval.NewDelayedEvaluation(waitUntil).
			WithTriggeredBy(models.EvalTriggerJobQueue).
			WithComment(comment)
		plan.AppendEvaluation(delayedEvaluation)
		log.Ctx(ctx).Debug().Msgf("Creating delayed evaluation %s to retry scheduling job %s in %s due to: %s",
			delayedEvaluation.ID, job.ID, waitUntil.Sub(b.clock.Now()), comment)
	}
	return nil
}

// compile-time assertion that BatchServiceJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &BatchServiceJobScheduler{}
