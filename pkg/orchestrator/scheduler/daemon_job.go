package scheduler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

// DaemonJobScheduler is a scheduler for batch jobs that run until completion
type DaemonJobScheduler struct {
	jobStore     jobstore.Store
	planner      orchestrator.Planner
	nodeSelector orchestrator.NodeSelector
	constraints  orchestrator.NodeSelectionConstraints
}

func NewDaemonJobScheduler(
	store jobstore.Store,
	planner orchestrator.Planner,
	selector orchestrator.NodeSelector,
	constraints orchestrator.NodeSelectionConstraints,
) *DaemonJobScheduler {
	return &DaemonJobScheduler{
		jobStore:     store,
		planner:      planner,
		nodeSelector: selector,
		constraints:  constraints,
	}
}

func (b *DaemonJobScheduler) Process(ctx context.Context, evaluation *models.Evaluation) error {
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
	nodeInfos, err := existingNodeInfos(ctx, b.nodeSelector, nonTerminalExecs)
	if err != nil {
		return err
	}

	// Mark executions that are running on nodes that are not healthy as failed
	_, lost := nonTerminalExecs.filterByNodeHealth(nodeInfos)
	lost.markStopped(orchestrator.ExecStoppedByNodeUnhealthyEvent(), plan)

	// Look for new matching nodes and create new executions every time we evaluate the job
	_, err = b.createMissingExecs(ctx, &job, plan, existingExecs)
	if err != nil {
		return fmt.Errorf("failed to find/create missing executions: %w", err)
	}

	plan.MarkJobRunningIfEligible()
	return b.planner.Process(ctx, plan)
}

func (b *DaemonJobScheduler) createMissingExecs(
	ctx context.Context, job *models.Job, plan *models.Plan, existingExecs execSet) (execSet, error) {
	newExecs := execSet{}

	// Require nodes to be approved and connected to schedule work.
	nodes, err := b.nodeSelector.AllMatchingNodes(
		ctx,
		job,
		&orchestrator.NodeSelectionConstraints{RequireApproval: true, RequireConnected: true},
	)
	if err != nil {
		return newExecs, err
	}

	// map for existing NodeIDs for faster lookup and filtering of existing executions
	existingNodes := make(map[string]struct{})
	for _, exec := range existingExecs {
		existingNodes[exec.NodeID] = struct{}{}
	}

	for _, node := range nodes {
		if _, ok := existingNodes[node.ID()]; ok {
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
			NodeID:       node.ID(),
		}
		execution.Normalize()
		newExecs[execution.ID] = execution
	}
	for _, exec := range newExecs {
		plan.AppendExecution(exec)
	}
	return newExecs, nil
}

// compile-time assertion that DaemonJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &DaemonJobScheduler{}
