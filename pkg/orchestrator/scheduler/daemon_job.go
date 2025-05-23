package scheduler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

// DaemonJobScheduler is a scheduler for batch jobs that run until completion
type DaemonJobSchedulerParams struct {
	JobStore     jobstore.Store
	Planner      orchestrator.Planner
	NodeSelector orchestrator.NodeSelector
	// RateLimiter controls the rate at which new executions are created
	// If not provided, a NoopRateLimiter is used
	RateLimiter ExecutionRateLimiter
}

type DaemonJobScheduler struct {
	jobStore     jobstore.Store
	planner      orchestrator.Planner
	nodeSelector orchestrator.NodeSelector
	rateLimiter  ExecutionRateLimiter
}

func NewDaemonJobScheduler(params DaemonJobSchedulerParams) *DaemonJobScheduler {
	if params.RateLimiter == nil {
		params.RateLimiter = NewNoopRateLimiter()
	}
	return &DaemonJobScheduler{
		jobStore:     params.JobStore,
		planner:      params.Planner,
		nodeSelector: params.NodeSelector,
		rateLimiter:  params.RateLimiter,
	}
}

func (b *DaemonJobScheduler) Process(ctx context.Context, evaluation *models.Evaluation) (err error) {
	metrics := telemetry.NewMetricRecorder(
		attribute.String(AttrSchedulerType, "daemon"),
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

	allJobVersionsExistingExecs := execSetFromSliceOfValues(allJobExecutions)

	// nonTerminalExecs will include:
	// 1 - executions that are in a terminal state
	nonTerminalExecs := allJobVersionsExistingExecs.filterNonTerminal()

	// early exit if the job is stopped
	if job.IsTerminal() {
		nonTerminalExecs.markCancelled(plan, orchestrator.ExecStoppedByJobStopEvent())
		metrics.AddAttributes(AttrOutcomeKey.String(AttrOutcomeAlreadyTerminal))
		return b.planner.Process(ctx, plan)
	}

	// Retrieve the info for all the nodes that have executions for this job
	nodeInfos, err := existingNodeInfos(ctx, b.nodeSelector, nonTerminalExecs)
	if err != nil {
		return err
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartGetNodes)

	// Mark executions that are running on nodes that are not healthy as failed
	nonTerminalExecs, lost := nonTerminalExecs.groupByNodeHealth(nodeInfos)
	lost.markFailed(plan, orchestrator.ExecStoppedByNodeUnhealthyEvent())
	metrics.CountAndHistogram(ctx, executionsLostTotal, executionsLost, float64(len(lost)))

	// After this call, nonTerminalExecs will now include:
	//	Running execs - Old Job versions (those were left after rate limiting)
	//  Running Execs - New Job version
	//	Pending Execs - New Job version
	// this will enqueue an evaluation and marks executions for cancellation rate limited
	nonTerminalExecs = b.handlePreviousVersionsExecutions(ctx, plan, nonTerminalExecs)

	// Look for new matching nodes and create new executions every time we evaluate the job
	allNewExecsAndOldRunningExecs := allJobVersionsExistingExecs.filterByJobVersion(job.Version).union(nonTerminalExecs)
	_, err = b.createMissingExecs(ctx, metrics, &job, plan, allNewExecsAndOldRunningExecs)
	if err != nil {
		return fmt.Errorf("failed to find/create missing executions: %w", err)
	}

	runningEvent := orchestrator.JobStateUpdateEvent(models.JobStateTypeRunning)
	plan.MarkJobRunningIfEligible(runningEvent)

	err = b.planner.Process(ctx, plan)
	metrics.Latency(ctx, processPartDuration, AttrOperationPartProcessPlan)
	return err
}

func (b *DaemonJobScheduler) createMissingExecs(
	ctx context.Context,
	metrics *telemetry.MetricRecorder,
	job *models.Job,
	plan *models.Plan,
	allJobVersionsExistingExecs execSet,
) (execSet, error) {
	newExecs := execSet{}

	nodes, rejected, err := b.nodeSelector.MatchingNodes(ctx, job)
	if err != nil {
		return newExecs, err
	}
	metrics.Histogram(ctx, nodesMatched, float64(len(nodes)))
	metrics.Histogram(ctx, nodesRejected, float64(len(rejected)))

	// map for existing NodeIDs for faster lookup and filtering of existing executions
	existingNodes := make(map[string]struct{})
	for _, exec := range allJobVersionsExistingExecs {
		existingNodes[exec.NodeID] = struct{}{}
	}

	// Find nodes that need new executions
	var nodesToSchedule []orchestrator.NodeRank
	for _, node := range nodes {
		if _, ok := existingNodes[node.NodeInfo.ID()]; !ok {
			nodesToSchedule = append(nodesToSchedule, node)
		}
	}

	// Apply rate limiting, which will enqueue evaluation.
	execsToCreate := b.rateLimiter.Apply(ctx, plan, len(nodesToSchedule))

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
		newExecs[execution.ID] = execution
	}

	for _, exec := range newExecs {
		plan.AppendExecution(exec, orchestrator.ExecCreatedEvent(exec))
	}
	metrics.CountAndHistogram(ctx, executionsCreatedTotal, executionsCreated, float64(len(newExecs)))
	return newExecs, nil
}

func (b *DaemonJobScheduler) handlePreviousVersionsExecutions(
	ctx context.Context,
	plan *models.Plan,
	nonTerminalExecs execSet,
) execSet {
	// nonTerminalExecs are:
	// 1- Running execs - Old versions
	// 2- Running Execs - New version
	// 3- Pending Execs - Old Versions ==> we should mark them directly as cancelled
	// 4- Pending Execs - New version

	// Previous versions execs
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

// compile-time assertion that DaemonJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &DaemonJobScheduler{}
