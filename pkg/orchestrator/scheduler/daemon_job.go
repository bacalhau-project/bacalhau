package scheduler

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
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

func (b *DaemonJobScheduler) Process(ctx context.Context, evaluation *models.Evaluation) error {
	ctx = log.Ctx(ctx).With().Str("JobID", evaluation.JobID).Str("EvalID", evaluation.ID).Logger().WithContext(ctx)

	job, err := b.jobStore.GetJob(ctx, evaluation.JobID)
	if err != nil {
		return fmt.Errorf("failed to retrieve job %s: %w", evaluation.JobID, err)
	}
	// Retrieve the job state
	jobExecutions, err := b.jobStore.GetExecutions(ctx, evaluation.JobID)
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
	_, lost := nonTerminalExecs.filterByNodeHealth(nodeInfos)
	lost.markStopped(execLost, plan)

	// Look for new matching nodes and create new executions everytime we evaluate the job
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
	nodes, err := b.nodeSelector.AllMatchingNodes(ctx, job)
	if err != nil {
		return newExecs, err
	}

	// map for existing NodeIDs for faster lookup and filtering of existing executions
	existingNodes := make(map[string]struct{})
	for _, exec := range existingExecs {
		existingNodes[exec.NodeID] = struct{}{}
	}

	for _, node := range nodes {
		if _, ok := existingNodes[node.PeerInfo.ID.String()]; ok {
			// there is already a healthy execution on this node
			continue
		}
		execution := &models.Execution{
			JobID:        job.ID,
			Job:          job,
			ID:           "e-" + uuid.NewString(),
			Namespace:    job.Namespace,
			ComputeState: models.NewExecutionState(models.ExecutionStateNew),
			DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning),
			NodeID:       node.PeerInfo.ID.String(),
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
