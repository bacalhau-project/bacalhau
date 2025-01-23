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
type OpsJobScheduler struct {
	jobStore jobstore.Store
	planner  orchestrator.Planner
	selector orchestrator.NodeSelector
	clock    clock.Clock
}

type OpsJobSchedulerParams struct {
	JobStore     jobstore.Store
	Planner      orchestrator.Planner
	NodeSelector orchestrator.NodeSelector
	// Clock is the clock used for time-based operations.
	// If not provided, the system clock is used.
	Clock clock.Clock
}

func NewOpsJobScheduler(params OpsJobSchedulerParams) *OpsJobScheduler {
	if params.Clock == nil {
		params.Clock = clock.New()
	}
	return &OpsJobScheduler{
		jobStore: params.JobStore,
		planner:  params.Planner,
		selector: params.NodeSelector,
		clock:    params.Clock,
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
	jobExecutions, err := b.jobStore.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID: evaluation.JobID,
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve job state for job %s when evaluating %s: %w",
			evaluation.JobID, evaluation, err)
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartGetExecs)
	metrics.Histogram(ctx, executionsExisting, float64(len(jobExecutions)))

	// Plan to hold the actions to be taken
	plan := models.NewPlan(evaluation, &job)

	existingExecs := execSetFromSliceOfValues(jobExecutions)
	nonTerminalExecs := existingExecs.filterNonTerminal()

	// early exit if the job is stopped
	if job.IsTerminal() {
		nonTerminalExecs.markStopped(plan, orchestrator.ExecStoppedByJobStopEvent())
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
	lost.markStopped(plan, orchestrator.ExecStoppedByNodeUnhealthyEvent())
	metrics.CountAndHistogram(ctx, executionsLostTotal, executionsLost, float64(len(lost)))
	allFailedExecs = allFailedExecs.union(lost)

	nonTerminalExecs, allFailedExecs = b.handleTimeouts(ctx, metrics, plan, nonTerminalExecs, allFailedExecs)

	// Look for matching nodes and create new executions if this is
	// the first time we are evaluating the job
	var newExecs execSet
	if len(existingExecs) == 0 {
		if newExecs, err = b.createMissingExecs(ctx, metrics, &job, plan); err != nil {
			return err // internal error
		}
	}

	// if the plan's job state is failed, stop all active executions
	if plan.IsJobFailed() {
		nonTerminalExecs.markStopped(plan, orchestrator.ExecStoppedDueToJobFailureEvent())
	} else {
		// mark job as completed if there are no more active or new executions
		if len(nonTerminalExecs) == 0 && len(newExecs) == 0 {
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
		nonTerminalExecs, timedOut = nonTerminalExecs.filterByExecutionTimeout(expirationTime)
		timedOut.markStopped(plan, orchestrator.ExecStoppedByExecutionTimeoutEvent(timeout))
		metrics.CountN(ctx, executionsTimedOut, int64(len(timedOut)))
		allFailedExecs = allFailedExecs.union(timedOut)
	}
	return nonTerminalExecs, allFailedExecs
}

func (b *OpsJobScheduler) createMissingExecs(
	ctx context.Context, metrics *telemetry.MetricRecorder,
	job *models.Job, plan *models.Plan) (execSet, error) {
	matching, rejected, err := b.selector.MatchingNodes(ctx, job)
	if err != nil {
		return nil, err
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartMatchNodes)
	metrics.Histogram(ctx, nodesMatched, float64(len(matching)))
	metrics.Histogram(ctx, nodesRejected, float64(len(rejected)))

	if len(matching) == 0 {
		plan.MarkJobFailed(
			*models.EventFromError(orchestrator.EventTopicJobScheduling, orchestrator.NewErrNoMatchingNodes(rejected)))
		return nil, nil
	}
	newExecs := execSet{}
	for _, node := range matching {
		execution := &models.Execution{
			JobID:        job.ID,
			Job:          job,
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
	metrics.CountAndHistogram(ctx, executionsCreatedTotal, executionsCreated, float64(len(matching)))
	return newExecs, nil
}

// compile-time assertion that OpsJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &OpsJobScheduler{}
