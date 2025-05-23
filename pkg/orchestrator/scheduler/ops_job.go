package scheduler

import (
	"context"
	"fmt"

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

// OpsJobScheduler is a scheduler for batch jobs that run until completion
type OpsJobSchedulerParams struct {
	JobStore     jobstore.Store
	Planner      orchestrator.Planner
	NodeSelector orchestrator.NodeSelector
	// RateLimiter controls the rate at which new executions are created
	// If not provided, a NoopRateLimiter is used
	RateLimiter ExecutionRateLimiter
	// Clock is the clock used for time-based operations.
	// If not provided, the system clock is used.
	Clock clock.Clock
}

type OpsJobScheduler struct {
	jobStore    jobstore.Store
	planner     orchestrator.Planner
	selector    orchestrator.NodeSelector
	rateLimiter ExecutionRateLimiter
	clock       clock.Clock
}

func NewOpsJobScheduler(params OpsJobSchedulerParams) *OpsJobScheduler {
	if params.Clock == nil {
		params.Clock = clock.New()
	}
	if params.RateLimiter == nil {
		params.RateLimiter = NewNoopRateLimiter()
	}
	return &OpsJobScheduler{
		jobStore:    params.JobStore,
		planner:     params.Planner,
		selector:    params.NodeSelector,
		rateLimiter: params.RateLimiter,
		clock:       params.Clock,
	}
}

func (b *OpsJobScheduler) Process(ctx context.Context, evaluation *models.Evaluation) (err error) {
	metrics := telemetry.NewMetricRecorder(
		attribute.String(AttrSchedulerType, "ops"),
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

	job, err := b.jobStore.GetJob(ctx, evaluation.JobID)
	if err != nil {
		return fmt.Errorf("failed to retrieve job %s: %w", evaluation.JobID, err)
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartGetJob)
	metrics.AddAttributes(attribute.String(AttrJobType, job.Type))

	// Retrieve the job state
	allJobExecutions, err := b.jobStore.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID:          evaluation.JobID,
		AllJobVersions: true,
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve job state for job %s when evaluating %s: %w",
			evaluation.JobID, evaluation, err)
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartGetExecs)
	metrics.Histogram(ctx, executionsExisting, float64(len(allJobExecutions)))

	// Plan to hold the actions to be taken
	plan := models.NewPlan(evaluation, &job)

	existingExecs := execSetFromSliceOfValues(allJobExecutions)
	nonTerminalExecs := existingExecs.filterNonTerminal()

	// early exit if the job is stopped
	if job.IsTerminal() {
		nonTerminalExecs.markCancelled(plan, orchestrator.ExecStoppedByJobStopEvent())
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
	nonTerminalExecs, lost := nonTerminalExecs.groupByNodeHealth(nodeInfos)
	lost.markFailed(plan, orchestrator.ExecStoppedByNodeUnhealthyEvent())
	metrics.CountAndHistogram(ctx, executionsLostTotal, executionsLost, float64(len(lost)))
	allFailedExecs = allFailedExecs.union(lost)

	nonTerminalExecs, allFailedExecs = b.handleTimeouts(ctx, metrics, plan, nonTerminalExecs, allFailedExecs)

	// this will enqueue an evaluation and marks executions for cancellation rate limited
	nonTerminalExecs = b.handlePreviousVersionsExecutions(ctx, plan, nonTerminalExecs)

	// Look for matching nodes and create new executions if this is
	// the first time we are evaluating the job
	// For ops job, we need to pass all the execs with new job version no matter
	//  their state, and also pass running execs with older job versions
	allNewExecsAndOldRunningExecs := existingExecs.filterByJobVersion(job.Version).union(nonTerminalExecs)
	if _, err = b.createMissingExecs(ctx, metrics, &job, plan, allNewExecsAndOldRunningExecs); err != nil {
		return err // internal error
	}

	// if the plan's job state is failed, stop all active executions
	if plan.IsJobFailed() {
		nonTerminalExecs.markCancelled(plan, orchestrator.ExecStoppedDueToJobFailureEvent())
	} else {
		// mark job as completed if there are no more active or new executions
		if len(nonTerminalExecs) == 0 && !plan.HasPendingWork() {
			if len(allFailedExecs) > 0 {
				plan.MarkJobFailed(orchestrator.JobExecutionsFailedEvent())
			} else {
				plan.MarkJobCompleted(orchestrator.JobStateUpdateEvent(models.JobStateTypeCompleted))
			}
		}
	}
	plan.MarkJobRunningIfEligible(orchestrator.JobStateUpdateEvent(models.JobStateTypeRunning))
	err = b.planner.Process(ctx, plan)
	metrics.Latency(ctx, processPartDuration, AttrOperationPartProcessPlan)
	return err
}

func (b *OpsJobScheduler) handleTimeouts(
	ctx context.Context, metrics *telemetry.MetricRecorder,
	plan *models.Plan, nonTerminalExecs execSet, allFailedExecs execSet) (execSet, execSet) {
	// Mark job/executions that have exceeded their total/execution timeout as failed
	// check if job has exceeded total timeout
	job := plan.Job
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
	return nonTerminalExecs, allFailedExecs
}

func (b *OpsJobScheduler) createMissingExecs(
	ctx context.Context, metrics *telemetry.MetricRecorder,
	job *models.Job, plan *models.Plan, existingExecs execSet) (execSet, error) {
	matching, rejected, err := b.selector.MatchingNodes(ctx, job)
	if err != nil {
		return nil, err
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartMatchNodes)
	metrics.Histogram(ctx, nodesMatched, float64(len(matching)))
	metrics.Histogram(ctx, nodesRejected, float64(len(rejected)))

	// Only fail if we have no matching nodes AND no existing executions
	if len(matching) == 0 && len(existingExecs) == 0 {
		plan.MarkJobFailed(
			*models.EventFromError(orchestrator.EventTopicJobScheduling, orchestrator.NewErrNoMatchingNodes(rejected)))
		return nil, nil
	}

	// If no matching nodes but we have existing executions, just return empty set
	if len(matching) == 0 {
		return execSet{}, nil
	}

	// Filter out nodes that we've already tried to schedule on
	existingNodes := make(map[string]struct{})
	for _, exec := range existingExecs {
		existingNodes[exec.NodeID] = struct{}{}
	}

	var nodesToSchedule []orchestrator.NodeRank
	for _, node := range matching {
		if _, exists := existingNodes[node.NodeInfo.ID()]; !exists {
			nodesToSchedule = append(nodesToSchedule, node)
		}
	}

	// If no new nodes to schedule on, return early
	if len(nodesToSchedule) == 0 {
		return execSet{}, nil
	}

	// Apply rate limiting to new nodes
	execsToCreate := b.rateLimiter.Apply(ctx, plan, len(nodesToSchedule))
	newExecs := execSet{}

	// Create executions up to the limit
	for i := 0; i < execsToCreate; i++ {
		node := nodesToSchedule[i]
		execution := &models.Execution{
			JobID:        job.ID,
			Job:          job,
			JobVersion:   job.Version,
			EvalID:       plan.EvalID,
			ID:           idgen.ExecutionIDPrefix + uuid.NewString(),
			Namespace:    job.Namespace,
			ComputeState: models.NewExecutionState(models.ExecutionStateNew),
			DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning),
			NodeID:       node.NodeInfo.ID(),
		}
		execution.Normalize()
		plan.AppendExecution(execution, orchestrator.ExecCreatedEvent(execution))
		newExecs[execution.ID] = execution
	}
	metrics.CountAndHistogram(ctx, executionsCreatedTotal, executionsCreated, float64(len(newExecs)))
	return newExecs, nil
}

func (b *OpsJobScheduler) handlePreviousVersionsExecutions(
	ctx context.Context,
	plan *models.Plan,
	nonTerminalExecs execSet,
) execSet {
	// nonTerminalExecs are:
	// 1- Running execs - Old versions
	// 2- Running Execs - New version
	// 3- Pending Execs - Old Versions ==> we should mark them directly as cancelled
	// 4- Pending Execs - New version

	previousJobVersionsExecs := nonTerminalExecs.filterByOutdatedJobVersion(plan.Job.Version)
	pendingExecsWithOldJobVersions := previousJobVersionsExecs.filterByDesiredState(models.ExecutionDesiredStatePending)
	pendingExecsWithOldJobVersions.markCancelled(plan, orchestrator.ExecStoppedForJobUpdateEvent())

	runningExecsWithOldJobVersions := previousJobVersionsExecs.filterByDesiredState(models.ExecutionDesiredStateRunning)

	// The rate limiter will enqueue a delayed evaluation
	countOfRunningExecsToCancel := b.rateLimiter.Apply(ctx, plan, len(runningExecsWithOldJobVersions))
	runningExecsToCancel, remainingRunningExecutions := runningExecsWithOldJobVersions.splitByCount(uint(countOfRunningExecsToCancel))
	runningExecsToCancel.markCancelled(plan, orchestrator.ExecStoppedForJobUpdateEvent())
	remainingExecs := nonTerminalExecs.difference(previousJobVersionsExecs).union(remainingRunningExecutions)

	return remainingExecs
}

// compile-time assertion that OpsJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &OpsJobScheduler{}
