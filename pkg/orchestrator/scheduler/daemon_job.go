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
	previouslyCreatedExecutionsList, err := b.jobStore.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID:          evaluation.JobID,
		AllJobVersions: true,
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve job state for job %s when evaluating %s: %w",
			evaluation.JobID, evaluation, err)
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartGetExecs)
	metrics.Histogram(ctx, executionsExisting, float64(len(previouslyCreatedExecutionsList)))

	// Plan to hold the actions to be taken
	plan := models.NewPlan(evaluation, &job)

	allPreviouslyCreatedExecutionsSetRaw := execSetFromSliceOfValues(previouslyCreatedExecutionsList)

	currentRuntimeIDExecutionsSet := execSet{}
	otherRuntimeIDExecutionsSet := execSet{} // These executions will be eventually marked for cancellation
	if evaluation.RuntimeID != "" {
		// A runtime ID was passed in the evaluation, make
		currentRuntimeIDExecutionsSet = allPreviouslyCreatedExecutionsSetRaw.filterByRuntimeID(evaluation.RuntimeID)
		otherRuntimeIDExecutionsSet = allPreviouslyCreatedExecutionsSetRaw.excludeThisRuntimeID(evaluation.RuntimeID)
	} else {
		// In case no runtime ID was passed in the evaluation
		currentRuntimeIDExecutionsSet = allPreviouslyCreatedExecutionsSetRaw
	}

	nonTerminalCurrentRuntimeIDExecutionsSet := currentRuntimeIDExecutionsSet.filterNonTerminal()

	// early exit if the job is stopped
	if job.IsTerminal() {
		nonTerminalCurrentRuntimeIDExecutionsSet.markCancelled(plan, orchestrator.ExecStoppedByJobStopEvent())
		metrics.AddAttributes(AttrOutcomeKey.String(AttrOutcomeAlreadyTerminal))
		return b.planner.Process(ctx, plan)
	}

	// If the evaluation was triggered by a Job Rerun/Update, mark otherRuntimeIDExecutionsSet to cancelled
	if evaluation.TriggeredBy == models.EvalTriggerJobRerun {
		log.Debug().Msgf("Processing job rerun for job %s, canceling any current active executions", evaluation.JobID)
		nonTerminalOtherRuntimeIDsExecutionsSet := otherRuntimeIDExecutionsSet.filterNonTerminal()
		nonTerminalOtherRuntimeIDsExecutionsSet.markCancelled(plan, orchestrator.ExecStoppedForJobRerunEvent())
		metrics.CountAndHistogram(ctx, executionsLostTotal, executionsLost, float64(len(nonTerminalOtherRuntimeIDsExecutionsSet)))
	}

	// Retrieve the info for all the nodes that have executions for this job
	nodeInfos, err := existingNodeInfos(ctx, b.nodeSelector, nonTerminalCurrentRuntimeIDExecutionsSet)
	if err != nil {
		return err
	}
	metrics.Latency(ctx, processPartDuration, AttrOperationPartGetNodes)

	// Mark executions that are running on nodes that are not healthy as failed
	_, lost := nonTerminalCurrentRuntimeIDExecutionsSet.groupByNodeHealth(nodeInfos)
	lost.markFailed(plan, orchestrator.ExecStoppedByNodeUnhealthyEvent())
	metrics.CountAndHistogram(ctx, executionsLostTotal, executionsLost, float64(len(lost)))

	// Look for new matching nodes and create new executions every time we evaluate the job
	_, err = b.createMissingExecs(ctx, metrics, &job, plan, currentRuntimeIDExecutionsSet)
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
	ctx context.Context, metrics *telemetry.MetricRecorder, job *models.Job, plan *models.Plan, existingExecs execSet) (execSet, error) {
	newExecs := execSet{}

	nodes, rejected, err := b.nodeSelector.MatchingNodes(ctx, job, plan.Eval)
	if err != nil {
		return newExecs, err
	}
	metrics.Histogram(ctx, nodesMatched, float64(len(nodes)))
	metrics.Histogram(ctx, nodesRejected, float64(len(rejected)))

	// map for existing NodeIDs for faster lookup and filtering of existing executions
	existingNodes := make(map[string]struct{})
	for _, exec := range existingExecs {
		existingNodes[exec.NodeID] = struct{}{}
	}

	// Find nodes that need new executions
	var nodesToSchedule []orchestrator.NodeRank
	for _, node := range nodes {
		if _, ok := existingNodes[node.NodeInfo.ID()]; !ok {
			nodesToSchedule = append(nodesToSchedule, node)
		}
	}

	// Apply rate limiting
	execsToCreate := b.rateLimiter.Apply(ctx, plan, len(nodesToSchedule))

	// Create executions up to the limit
	for i := 0; i < execsToCreate; i++ {
		node := nodesToSchedule[i]
		execution := &models.Execution{
			JobID:        job.ID,
			Job:          job,
			JobVersion:   plan.Job.Version,
			EvalID:       plan.EvalID,
			Evaluation:   plan.Eval,
			ID:           idgen.ExecutionIDPrefix + uuid.NewString(),
			Namespace:    job.Namespace,
			ComputeState: models.NewExecutionState(models.ExecutionStateNew),
			DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning),
			NodeID:       node.NodeInfo.ID(),
			RuntimeID:    plan.Eval.RuntimeID,
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

// compile-time assertion that DaemonJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &DaemonJobScheduler{}
