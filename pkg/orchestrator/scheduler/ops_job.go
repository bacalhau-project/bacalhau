package scheduler

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

// OpsJobScheduler is a scheduler for batch jobs that run until completion
type OpsJobScheduler struct {
	jobStore jobstore.Store
	planner  orchestrator.Planner
	selector orchestrator.NodeSelector
}

func NewOpsJobScheduler(
	store jobstore.Store,
	planner orchestrator.Planner,
	selector orchestrator.NodeSelector,
) *OpsJobScheduler {
	return &OpsJobScheduler{
		jobStore: store,
		planner:  planner,
		selector: selector,
	}
}

func (b *OpsJobScheduler) Process(ctx context.Context, evaluation *models.Evaluation) error {
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

	// Mark executions that are running on nodes that are not healthy as failed
	nonTerminalExecs, lost := nonTerminalExecs.filterByNodeHealth(nodeInfos)
	lost.markStopped(orchestrator.ExecStoppedByNodeUnhealthyEvent(), plan)

	allFailed := existingExecs.filterFailed().union(lost)

	// Look for matching nodes and create new executions if:
	// - Ops jobs: this is the first time we are evaluating the job
	// - Daemon jobs: every time the job is evaluated
	var newExecs execSet
	if job.Type == models.JobTypeDaemon || len(existingExecs) == 0 {
		newExecs, err = b.createMissingExecs(ctx, &job, plan)
		if err != nil {
			b.handleFailure(nonTerminalExecs, allFailed, plan, err)
			return b.planner.Process(ctx, plan)
		}
	}

	// mark job as completed if there are no more active or new executions
	if len(nonTerminalExecs) == 0 && len(newExecs) == 0 {
		if len(allFailed) > 0 {
			b.handleFailure(nonTerminalExecs, allFailed, plan, errors.New(""))
		} else {
			plan.MarkJobCompleted()
		}
	}

	plan.MarkJobRunningIfEligible()
	return b.planner.Process(ctx, plan)
}

func (b *OpsJobScheduler) createMissingExecs(
	ctx context.Context, job *models.Job, plan *models.Plan) (execSet, error) {
	newExecs := execSet{}
	nodes, err := b.selector.AllMatchingNodes(ctx, job)
	if err != nil {
		return newExecs, err
	}
	if len(nodes) == 0 {
		return nil, orchestrator.ErrNoMatchingNodes{}
	}
	for _, node := range nodes {
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

func (b *OpsJobScheduler) handleFailure(nonTerminalExecs execSet, failed execSet, plan *models.Plan, err error) {
	// mark all non-terminal executions as failed
	nonTerminalExecs.markStopped(plan.Event, plan)

	// mark the job as failed, using the error message of the latest failed execution, if any, or use
	// the error message passed by the scheduler
	plan.MarkJobFailed(plan.Event)
}

// compile-time assertion that OpsJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &OpsJobScheduler{}
