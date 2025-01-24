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
type DaemonJobScheduler struct {
	jobStore     jobstore.Store
	planner      orchestrator.Planner
	nodeSelector orchestrator.NodeSelector
}

type DaemonJobSchedulerParams struct {
	JobStore     jobstore.Store
	Planner      orchestrator.Planner
	NodeSelector orchestrator.NodeSelector
}

func NewDaemonJobScheduler(params DaemonJobSchedulerParams) *DaemonJobScheduler {
	return &DaemonJobScheduler{
		jobStore:     params.JobStore,
		planner:      params.Planner,
		nodeSelector: params.NodeSelector,
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
	_, lost := nonTerminalExecs.filterByNodeHealth(nodeInfos)
	lost.markStopped(plan, orchestrator.ExecStoppedByNodeUnhealthyEvent())
	metrics.CountAndHistogram(ctx, executionsLostTotal, executionsLost, float64(len(lost)))

	// Look for new matching nodes and create new executions every time we evaluate the job
	_, err = b.createMissingExecs(ctx, metrics, &job, plan, existingExecs)
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

	nodes, rejected, err := b.nodeSelector.MatchingNodes(ctx, job)
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

	for _, node := range nodes {
		if _, ok := existingNodes[node.NodeInfo.ID()]; ok {
			// there is already a healthy execution on this node
			continue
		}
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
