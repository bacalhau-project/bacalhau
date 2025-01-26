package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
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

// BatchServiceJobScheduler handles scheduling of batch and service jobs, with support for
// partitioned execution. Each job can specify N parallel executions via its Count field,
// where each execution is assigned a unique partition index from 0 to Count-1.
//
// Partitioning Logic:
// - The job.Count field determines the number of partitions (N)
// - Each execution is assigned a unique PartitionIndex (0 to Count-1)
// - The scheduler ensures exactly one active execution per partition
// - When an execution fails, its partition becomes available for retry
//
// Execution Lifecycle Per Partition:
// 1. Initial scheduling: assign execution to an available node with PartitionIndex
// 2. If execution fails: partition becomes available for retry on another node
// 3. If execution succeeds:
//   - Batch jobs: partition is marked complete, no more executions
//   - Service jobs: partition must maintain one running execution
//
// Example with Count = 3:
// - Creates three partitions (0, 1, 2)
// - Each partition runs independently
// - If partition 1 fails, it can be retried while 0 and 2 continue running
// - Batch job completes when all three partitions have successful executions
// - Service job runs continuously with one execution per partition
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

// Process handles the scheduling logic for partitioned jobs:
// 1. Tracks existing executions per partition
// 2. Handles failed executions and retries while maintaining partition assignment
// 3. Ensures each partition (0 to job.Count-1) has exactly one active execution
// 4. For batch jobs: monitors completion of all partitions
// 5. For service jobs: maintains continuous execution per partition
func (b *BatchServiceJobScheduler) Process(ctx context.Context, evaluation *models.Evaluation) (err error) {
	log.Debug().Msgf("Processing evaluation %s for job %s", evaluation.ID, evaluation.JobID)
	metrics := telemetry.NewMetricRecorder(
		attribute.String(AttrSchedulerType, "batch_service"),
		attribute.String(AttrEvalType, evaluation.Type),
		AttrOutcomeKey.String(AttrOutcomeSuccess), // default outcome is success, unless changed
	)
	defer func() {
		if err != nil {
			metrics.Error(err)
			metrics.AddAttributes(AttrOutcomeKey.String(AttrOutcomeFailure))
		}
		metrics.Count(ctx, processCount)
		metrics.Done(ctx, processDuration)
	}()

	ctx = log.Ctx(ctx).With().Str("JobID", evaluation.JobID).Str("EvalID", evaluation.ID).Logger().WithContext(ctx)

	job, jobExecutions, err := b.loadJobState(ctx, metrics, evaluation)
	if err != nil {
		return err
	}

	// Plan to hold the actions to be taken
	plan := models.NewPlan(evaluation, &job)
	existingExecs := execSetFromSliceOfValues(jobExecutions)
	nonTerminalExecs := existingExecs.filterNonTerminal()

	// early exit if the job is stopped
	if job.IsTerminal() {
		nonTerminalExecs.markCancelled(plan, orchestrator.ExecStoppedByJobStopEvent())
		metrics.AddAttributes(AttrOutcomeKey.String(AttrOutcomeAlreadyTerminal))
		return b.planner.Process(ctx, plan)
	}

	// Retrieve the info for all the nodes that have executions for this job
	nodeInfos, err := existingNodeInfos(ctx, b.selector, nonTerminalExecs)
	if err != nil {
		return err
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartGetNodes)

	// keep track of existing failed executions, and those that will be marked as failed
	allFailedExecs := existingExecs.filterFailed()

	// Mark executions that are running on nodes that are not healthy as failed
	nonTerminalExecs, lost := nonTerminalExecs.groupByNodeHealth(nodeInfos)
	if len(lost) > 0 {
		lost.markFailed(plan, orchestrator.ExecStoppedByNodeUnhealthyEvent())
		metrics.CountAndHistogram(ctx, executionsLostTotal, executionsLost, float64(len(lost)))
		allFailedExecs = allFailedExecs.union(lost)
	}

	nonTerminalExecs, allFailedExecs = b.handleTimeouts(ctx, metrics, plan, nonTerminalExecs, allFailedExecs)

	// nonDiscardedExec is the set of executions that are either active or successfully completed
	nonDiscardedExecs := nonTerminalExecs.union(existingExecs.filterCompleted())
	if job.Type == models.JobTypeService {
		// Service jobs run until the user stops the job, and would be a bug if an execution is marked completed.
		nonDiscardedExecs = nonTerminalExecs
	}

	// Approve/Reject nodes
	b.approveRejectExecs(nonDiscardedExecs, plan)

	// schedule remaining partitions and assign to new executions
	err = b.scheduleRemainingPartitions(ctx, metrics, plan, nonDiscardedExecs, allFailedExecs)
	if err != nil {
		return err
	}

	// if the plan's job state if terminal, stop all active executions
	if plan.IsJobFailed() {
		nonTerminalExecs.markCancelled(plan, orchestrator.ExecStoppedDueToJobFailureEvent())
	} else {
		if b.isJobComplete(&job, existingExecs) {
			// If there are no remaining partitions to be done, mark the job as completed.
			plan.MarkJobCompleted(orchestrator.JobStateUpdateEvent(models.JobStateTypeCompleted))
		}
		plan.MarkJobRunningIfEligible(orchestrator.JobStateUpdateEvent(models.JobStateTypeRunning))
	}

	err = b.planner.Process(ctx, plan)
	metrics.Latency(ctx, processPartDuration, AttrOperationPartProcessPlan)
	return err
}

func (b *BatchServiceJobScheduler) loadJobState(ctx context.Context, metrics *telemetry.MetricRecorder,
	evaluation *models.Evaluation) (models.Job, []models.Execution, error) {
	job, err := b.jobStore.GetJob(ctx, evaluation.JobID)
	if err != nil {
		return models.Job{}, nil, fmt.Errorf("failed to retrieve job %s: %w", evaluation.JobID, err)
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartGetJob)
	metrics.AddAttributes(attribute.String(AttrJobType, job.Type))

	// Retrieve the job executions
	jobExecutions, err := b.jobStore.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID: evaluation.JobID,
	})
	if err != nil {
		return models.Job{}, nil, fmt.Errorf("failed to retrieve executions for job %s when evaluating %s: %w",
			evaluation.JobID, evaluation, err)
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartGetExecs)
	metrics.Histogram(ctx, executionsExisting, float64(len(jobExecutions)))

	// loop and log execution id, desired state and compute state
	for _, exec := range jobExecutions {
		log.Ctx(ctx).Debug().Msgf("Found Execution %s: DesiredState=%s, ComputeState=%s", exec.ID, exec.DesiredState,
			exec.ComputeState)
	}
	return job, jobExecutions, nil
}

func (b *BatchServiceJobScheduler) handleTimeouts(ctx context.Context, metrics *telemetry.MetricRecorder,
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
			metrics.AddAttributes(AttrOutcomeKey.String(AttrOutcomeTimeout))
		}

		// check if the executions have exceeded their execution timeout
		if !plan.IsJobFailed() && job.Task().Timeouts.GetExecutionTimeout() > 0 {
			timeout := job.Task().Timeouts.GetExecutionTimeout()
			expirationTime := b.clock.Now().Add(-timeout)
			var timedOut execSet
			nonTerminalExecs, timedOut = nonTerminalExecs.groupByExecutionTimeout(expirationTime)
			timedOut.markFailed(plan, orchestrator.ExecStoppedByExecutionTimeoutEvent(timeout))
			metrics.CountN(ctx, executionsTimedOut, int64(len(timedOut)))
			allFailedExecs = allFailedExecs.union(timedOut)
		}
	}
	return nonTerminalExecs, allFailedExecs
}

// approveRejectExecs processes executions partition by partition (0 to job.Count-1).
// For each partition:
// - If partition has a completed execution: reject all other executions
// - If partition has a running execution: keep oldest, reject others
// - If partition has no running execution: approve oldest bid, reject others
// This ensures exactly one active execution per partition at any time.
func (b *BatchServiceJobScheduler) approveRejectExecs(nonDiscardedExecs execSet, plan *models.Plan) {
	// Process each partition independently, ensuring only one execution
	// can be active per partition at any time
	for partitionIndex, partitionExecs := range nonDiscardedExecs.groupByPartition() {
		execsByApprovalStatus := partitionExecs.getApprovalStatuses()

		// loop and log the partition index, and the number of executions in each state
		log.Debug().Msgf("Partition %d: %d to approve, %d to reject, %d to cancel", partitionIndex,
			len(execsByApprovalStatus.toApprove), len(execsByApprovalStatus.toReject), len(execsByApprovalStatus.toCancel))

		for _, exec := range partitionExecs {
			log.Debug().Msgf("approveRejectExecs Execution %s: DesiredState=%s, ComputeState=%s", exec.ID, exec.DesiredState,
				exec.ComputeState)
		}

		// loop and log the execution id, desired state and compute state for each toApprove, toReject and toCancel
		for _, exec := range execsByApprovalStatus.toApprove {
			log.Debug().Msgf("ToApprove Execution %s: DesiredState=%s, ComputeState=%s", exec.ID, exec.DesiredState,
				exec.ComputeState)
		}
		for _, exec := range execsByApprovalStatus.toReject {
			log.Debug().Msgf("ToReject Execution %s: DesiredState=%s, ComputeState=%s", exec.ID, exec.DesiredState,
				exec.ComputeState)
		}
		for _, exec := range execsByApprovalStatus.toCancel {
			log.Debug().Msgf("ToCancel Execution %s: DesiredState=%s, ComputeState=%s", exec.ID, exec.DesiredState,
				exec.ComputeState)
		}

		execsByApprovalStatus.toApprove.markApproved(plan, orchestrator.ExecRunningEvent())
		execsByApprovalStatus.toReject.markRejected(plan, orchestrator.ExecStoppedByNodeRejectedEvent())
		execsByApprovalStatus.toCancel.markCancelled(plan, orchestrator.ExecStoppedByOversubscriptionEvent())
	}
}

func (b *BatchServiceJobScheduler) scheduleRemainingPartitions(ctx context.Context, metrics *telemetry.MetricRecorder, plan *models.Plan,
	nonDiscardedExecs execSet, allFailedExecs execSet) error {
	remainingPartitions := nonDiscardedExecs.remainingPartitions(plan.Job.Count)
	if len(remainingPartitions) == 0 {
		return nil
	}

	// first, check if there were any failed executions and if we should retry
	if len(allFailedExecs) > 0 {
		if !b.retryStrategy.ShouldRetry(ctx, orchestrator.RetryRequest{JobID: plan.Job.ID}) {
			plan.MarkJobFailed(orchestrator.JobExhaustedRetriesEvent())
			metrics.Count(ctx, retriesExhausted)
			metrics.AddAttributes(AttrOutcomeKey.String(AttrOutcomeExhaustedRetries))
			return nil
		} else {
			metrics.Count(ctx, retriesAttempted)
		}
	}

	// find matching nodes for the remaining executions
	return b.createMissingExecs(ctx, metrics, plan, remainingPartitions)
}

// createMissingExecs creates new executions for partitions that need them.
// The remainingPartitions parameter contains available partition indices (0 to job.Count-1)
// that have no active or completed executions. This happens in two cases:
// 1. Initial scheduling: all partitions need first execution
// 2. Retry after failure: failed partitions need new execution
//
// For example, with job.Count = 3:
// - Initial: remainingPartitions = [0,1,2]
// - If partition 1 fails: remainingPartitions = [1]
// - If all complete (batch): remainingPartitions = []
func (b *BatchServiceJobScheduler) createMissingExecs(
	ctx context.Context, metrics *telemetry.MetricRecorder, plan *models.Plan, remainingPartitions []int) error {
	// find matching nodes for the job
	matching, rejected, err := b.selector.MatchingNodes(ctx, plan.Job)
	if err != nil {
		return err
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartMatchNodes)
	metrics.Histogram(ctx, nodesMatched, float64(len(matching)))
	metrics.Histogram(ctx, nodesRejected, float64(len(rejected)))

	// fail fast if there are not enough nodes, and we've passed job queue timeout
	if len(matching) < len(remainingPartitions) {
		// check if we can retry scheduling the job at a later time if we don't have enough nodes
		timeout := plan.Job.Task().Timeouts.GetQueueTimeout()
		expirationTime := b.clock.Now().Add(-timeout)

		// TODO: we are calculating queue timeout based on job creation time, but we should probably
		//  calculate it based on the time the job stayed in the queue so that rescheduling the job
		//  would reset the queue timeout.
		if plan.Job.GetCreateTime().Before(expirationTime) {
			plan.MarkJobFailed(*models.EventFromError(
				orchestrator.EventTopicJobScheduling,
				orchestrator.NewErrNotEnoughNodes(len(remainingPartitions), append(matching, rejected...))))
			metrics.AddAttributes(AttrOutcomeKey.String(AttrOutcomeQueueTimeout))
			return nil
		}

		// create a delayed evaluation to retry scheduling the job if we don't have enough nodes,
		// and we haven't passed the queue timeout
		waitUntil := b.clock.Now().Add(b.queueBackoff)
		comment := orchestrator.NewErrNotEnoughNodes(len(remainingPartitions), append(matching, rejected...)).Error()
		delayedEvaluation := plan.Eval.NewDelayedEvaluation(waitUntil).
			WithTriggeredBy(models.EvalTriggerJobQueue).
			WithComment(comment)
		plan.AppendEvaluation(delayedEvaluation)
		metrics.AddAttributes(AttrOutcomeKey.String(AttrOutcomeQueueing))
		log.Ctx(ctx).Debug().Msgf("Creating delayed evaluation %s to retry scheduling job %s in %s due to: %s",
			delayedEvaluation.ID, plan.Job.ID, waitUntil.Sub(b.clock.Now()), comment)

		// if not a single node was matched, then the job if fully queued and we should reflect that
		// in the job state and events
		if len(matching) == 0 {
			// only update the state if the is running, or pending and triggered by job registration
			if plan.Job.State.StateType == models.JobStateTypeRunning ||
				plan.Eval.TriggeredBy == models.EvalTriggerJobRegister {
				plan.MarkJobQueued(orchestrator.JobQueueingEvent(comment))
			}
		}
	}

	// create new executions
	var count float64
	for i := 0; i < min(len(matching), len(remainingPartitions)); i++ {
		execution := &models.Execution{
			NodeID:         matching[i].NodeInfo.ID(),
			JobID:          plan.Job.ID,
			Job:            plan.Job,
			ID:             idgen.ExecutionIDPrefix + uuid.NewString(),
			EvalID:         plan.EvalID,
			Namespace:      plan.Job.Namespace,
			ComputeState:   models.NewExecutionState(models.ExecutionStateNew),
			DesiredState:   models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
			PartitionIndex: remainingPartitions[i],
		}
		execution.Normalize()
		plan.AppendExecution(execution, orchestrator.ExecCreatedEvent(execution))
		count++
	}
	metrics.CountAndHistogram(ctx, executionsCreatedTotal, executionsCreated, count)

	return nil
}

// isJobComplete determines if all partitions have completed successfully.
// For a job with Count = N, this means:
// - We have exactly N completed partitions (one per index 0 to N-1)
// - Each partition has exactly one successful execution
// Only applicable to batch jobs as service jobs run continuously.
func (b *BatchServiceJobScheduler) isJobComplete(job *models.Job, existingExecs execSet) bool {
	if job.Type != models.JobTypeBatch {
		return false
	}
	if len(existingExecs.completedPartitions()) < job.Count {
		return false
	}
	return true
}

// compile-time assertion that BatchServiceJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &BatchServiceJobScheduler{}
