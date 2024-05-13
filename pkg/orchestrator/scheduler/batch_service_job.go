package scheduler

import (
	"context"
	"fmt"

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

	// Mark executions that have exceeded their execution timeout as failed
	// Only applicable for batch jobs and not service jobs.
	if !job.IsLongRunning() {
		timeout := job.Task().Timeouts.GetExecutionTimeout()
		expirationTime := b.clock.Now().Add(-timeout)
		var timedOut execSet
		nonTerminalExecs, timedOut = nonTerminalExecs.filterByExecutionTimeout(expirationTime)
		timedOut.markStopped(orchestrator.ExecStoppedByExecutionTimeoutEvent(timeout), plan)
		allFailedExecs = allFailedExecs.union(timedOut)
	}

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
		var placementErr error
		if len(allFailedExecs) > 0 && !b.retryStrategy.ShouldRetry(ctx, orchestrator.RetryRequest{JobID: job.ID}) {
			placementErr = fmt.Errorf("exceeded max retries for job %s", job.ID)
			plan.Event = orchestrator.JobExhaustedRetriesEvent()
		} else {
			_, placementErr = b.createMissingExecs(ctx, remainingExecutionCount, &job, plan)
		}
		if placementErr != nil {
			b.handleFailure(nonTerminalExecs, allFailedExecs, plan, placementErr)
			return b.planner.Process(ctx, plan)
		}
	}

	// stop executions if we over-subscribed and exceeded the desired number of executions
	_, overSubscriptions := execsByApprovalStatus.running.filterByOverSubscriptions(desiredRemainingCount)
	overSubscriptions.markStopped(orchestrator.ExecStoppedByOversubscriptionEvent(), plan)

	// Check the job's state and update it accordingly.
	if desiredRemainingCount <= 0 {
		// If there are no remaining tasks to be done, mark the job as completed.
		plan.MarkJobCompleted()
	}

	plan.MarkJobRunningIfEligible()
	return b.planner.Process(ctx, plan)
}

func (b *BatchServiceJobScheduler) createMissingExecs(
	ctx context.Context, remainingExecutionCount int, job *models.Job, plan *models.Plan) (execSet, error) {
	newExecs := execSet{}
	for i := 0; i < remainingExecutionCount; i++ {
		execution := &models.Execution{
			JobID:        job.ID,
			Job:          job,
			ID:           idgen.ExecutionIDPrefix + uuid.NewString(),
			EvalID:       plan.EvalID,
			Namespace:    job.Namespace,
			ComputeState: models.NewExecutionState(models.ExecutionStateNew),
			DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
		}
		execution.Normalize()
		newExecs[execution.ID] = execution
	}
	if len(newExecs) > 0 {
		err := b.placeExecs(ctx, newExecs, job)
		if err != nil {
			plan.Event = models.EventFromError(orchestrator.EventTopicJobScheduling, err)
			return newExecs, err
		}
	}
	for _, exec := range newExecs {
		plan.AppendExecution(exec)
	}
	return newExecs, nil
}

// placeExecs places the executions
func (b *BatchServiceJobScheduler) placeExecs(ctx context.Context, execs execSet, job *models.Job) error {
	if len(execs) > 0 {
		selectedNodes, err := b.selector.TopMatchingNodes(ctx, job, len(execs))
		if err != nil {
			return err
		}
		i := 0
		for _, exec := range execs {
			exec.NodeID = selectedNodes[i].ID()
			i++
		}
	}
	return nil
}

func (b *BatchServiceJobScheduler) handleFailure(nonTerminalExecs execSet, failed execSet, plan *models.Plan, err error) {
	// TODO(walid): allow scheduling retries in a later time if don't find nodes instead of failing the job
	// mark all non-terminal executions as failed
	nonTerminalExecs.markStopped(plan.Event, plan)

	// mark the job as failed, use the error message passed by the scheduler
	plan.MarkJobFailed(plan.Event)
}

// compile-time assertion that BatchServiceJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &BatchServiceJobScheduler{}
