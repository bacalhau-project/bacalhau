package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

// BatchServiceJobScheduler is a scheduler for:
// - batch jobs that run until completion on N number of nodes
// - service jobs than run until stopped on N number of nodes
type BatchServiceJobScheduler struct {
	jobStore      jobstore.Store
	planner       orchestrator.Planner
	nodeSelector  orchestrator.NodeSelector
	retryStrategy orchestrator.RetryStrategy
}

type BatchServiceJobSchedulerParams struct {
	JobStore      jobstore.Store
	Planner       orchestrator.Planner
	NodeSelector  orchestrator.NodeSelector
	RetryStrategy orchestrator.RetryStrategy
}

func NewBatchServiceJobScheduler(params BatchServiceJobSchedulerParams) *BatchServiceJobScheduler {
	return &BatchServiceJobScheduler{
		jobStore:      params.JobStore,
		planner:       params.Planner,
		nodeSelector:  params.NodeSelector,
		retryStrategy: params.RetryStrategy,
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
		nonTerminalExecs.markStopped(execNotNeeded, plan)
		return b.planner.Process(ctx, plan)
	}

	// Retrieve the info for all the nodes that have executions for this job
	nodeInfos, err := existingNodeInfos(ctx, b.nodeSelector, nonTerminalExecs)
	if err != nil {
		return err
	}

	// Mark executions that are running on nodes that are not healthy as failed
	nonTerminalExecs, lost := nonTerminalExecs.filterByNodeHealth(nodeInfos)
	lost.markStopped(execLost, plan)

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
	execsByApprovalStatus.toReject.markStopped(execRejected, plan)

	// How many executions failed due to compute nodes rejecting bids?
	rejected := existingExecs.filterRejected()

	// All failed/lost/rejected executions
	allFailed := existingExecs.filterFailed().union(lost).union(rejected)

	shouldRetry, retryDelay := b.retryStrategy.ShouldRetry(ctx, orchestrator.RetryRequest{Job: &job})

	// If we have failed executions, we need to retry in case the nodes accept
	// them in future (eg, due to transient capacity issues). b.handleRetry()
	// will do this for us, but it results in this same scheduler code being
	// re-executed after the delay. Therefore, we must check
	// evaluation.TriggeredBy to see if we are being triggered by a deferred
	// retry; and if so, we do NOT just defer a retry into the future, as
	// otherwise we'd loop forever.
	if (len(allFailed) > 0) && evaluation.TriggeredBy != models.EvalTriggerDefer {
		if shouldRetry {
			// Only delay if we have rejected executions, otherwise fall through to retry immediately
			if len(rejected) > 0 {
				b.handleRetry(ctx, plan, &job, retryDelay)
				return b.planner.Process(ctx, plan)
			}
		} else {
			b.handleFailure(nonTerminalExecs, allFailed, plan, fmt.Errorf("exceeded max retries for job %s", job.ID))
			return b.planner.Process(ctx, plan)
		}
	}

	// create new executions if needed
	remainingExecutionCount := desiredRemainingCount - execsByApprovalStatus.activeCount()
	if remainingExecutionCount > 0 {
		newExecs, placementErr := b.createMissingExecs(ctx, remainingExecutionCount, &job, retryDelay, plan)
		if placementErr != nil {
			b.handleFailure(nonTerminalExecs, allFailed, plan, placementErr)
			return b.planner.Process(ctx, plan)
		}
		if newExecs.countPlaced() < remainingExecutionCount {
			// Not enough nodes were available for the number of executions we
			// want.  Nodes may become available later, either directly or through
			// having had executions currently within the retryDelay interval that
			// time out of it. So as well as creating these new executions, let's
			// also schedule a new evaluation in retryDelay seconds.
			if shouldRetry {
				b.handleRetry(ctx, plan, &job, retryDelay)
			} else {
				// Remove any new executions we managed from the plan, no sense creating them now
				plan.CancelNewExecutions()
				b.handleFailure(nonTerminalExecs, allFailed, plan, fmt.Errorf("exceeded max retries for job %s", job.ID))
				return b.planner.Process(ctx, plan)
			}
		}
	}

	// stop executions if we over-subscribed and exceeded the desired number of executions
	_, overSubscriptions := execsByApprovalStatus.running.filterByOverSubscriptions(desiredRemainingCount)
	overSubscriptions.markStopped(execNotNeeded, plan)

	// Check the job's state and update it accordingly.
	if desiredRemainingCount <= 0 {
		// If there are no remaining tasks to be done, mark the job as completed.
		plan.MarkJobCompleted()
	}

	plan.MarkJobRunningIfEligible()
	return b.planner.Process(ctx, plan)
}

func (b *BatchServiceJobScheduler) createMissingExecs(
	ctx context.Context, remainingExecutionCount int, job *models.Job, retryDelay time.Duration, plan *models.Plan) (execSet, error) {
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
		err := b.placeExecs(ctx, newExecs, job, retryDelay)
		if err != nil {
			return newExecs, err
		}
	}
	for _, exec := range newExecs {
		// Not all execs might get Node IDs from placeExecs, if there weren't
		// enough nodes to place them all at once
		if exec.NodeID != "" {
			plan.AppendExecution(exec)
		}
	}
	return newExecs, nil
}

// placeExecs places the executions
func (b *BatchServiceJobScheduler) placeExecs(ctx context.Context, execs execSet, job *models.Job, retryDelay time.Duration) error {
	if len(execs) > 0 {
		selectedNodes, err := b.nodeSelector.TopMatchingNodes(ctx, job, retryDelay, len(execs))
		if err != nil {
			return err
		}
		i := 0
		for _, exec := range execs {
			// We may get less nodes back than we requested, if there aren't enough available yet
			if i < len(selectedNodes) {
				exec.NodeID = selectedNodes[i].ID()
			} else {
				exec.NodeID = ""
			}
			i++
		}
	}
	return nil
}

func (b *BatchServiceJobScheduler) handleRetry(ctx context.Context, plan *models.Plan, job *models.Job, delay time.Duration) {
	// Compute time to try again
	waitUntil := time.Now().Add(delay)
	// Schedule a new evaluation
	plan.DeferEvaluation(waitUntil)
	// Record deferral
	b.jobStore.RecordJobDeferral(ctx, job.ID, waitUntil, fmt.Sprintf("Deferring rescheduling for %s", delay))
}

func (b *BatchServiceJobScheduler) handleFailure(nonTerminalExecs execSet, failed execSet, plan *models.Plan, err error) {
	// mark all non-terminal executions as failed
	nonTerminalExecs.markStopped(jobFailed, plan)

	// mark the job as failed, using the error message of the latest failed execution, if any, or use
	// the error message passed by the scheduler
	latestErr := err.Error()
	if len(failed) > 0 {
		latestErr = failed.latest().ComputeState.Message
	}
	plan.MarkJobFailed(latestErr)
}

// compile-time assertion that BatchServiceJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &BatchServiceJobScheduler{}
