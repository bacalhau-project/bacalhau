package scheduler

import (
	"context"
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// OpsJobScheduler is a scheduler for batch jobs that run until completion
type OpsJobScheduler struct {
	jobStore       jobstore.Store
	planner        orchestrator.Planner
	nodeDiscoverer orchestrator.NodeDiscoverer
	nodeRanker     orchestrator.NodeRanker
}

type OpsJobSchedulerParams struct {
	JobStore       jobstore.Store
	Planner        orchestrator.Planner
	NodeDiscoverer orchestrator.NodeDiscoverer
	NodeRanker     orchestrator.NodeRanker
}

func NewOpsJobScheduler(params OpsJobSchedulerParams) *OpsJobScheduler {
	return &OpsJobScheduler{
		jobStore:       params.JobStore,
		planner:        params.Planner,
		nodeDiscoverer: params.NodeDiscoverer,
		nodeRanker:     params.NodeRanker,
	}
}

func (b *OpsJobScheduler) Process(ctx context.Context, evaluation *models.Evaluation) error {
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
	nodeInfos, err := existingNodeInfos(ctx, b.nodeDiscoverer, nonTerminalExecs)
	if err != nil {
		return err
	}

	// Mark executions that are running on nodes that are not healthy as failed
	nonTerminalExecs, lost := nonTerminalExecs.filterByNodeHealth(nodeInfos)
	lost.markStopped(execLost, plan)

	// create new executions only if this is the first time we are evaluating this job
	var newExecs execSet
	allFailed := existingExecs.filterFailed().union(lost)
	if len(existingExecs) == 0 {
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
	return b.planner.Process(ctx, plan)
}

func (b *OpsJobScheduler) createMissingExecs(
	ctx context.Context, job *models.Job, plan *models.Plan) (execSet, error) {
	newExecs := execSet{}
	nodes, err := b.selectNodes(ctx, job)
	if err != nil {
		return newExecs, err
	}
	for _, node := range nodes {
		execution := &models.Execution{
			JobID:        job.ID,
			Job:          job,
			ID:           "e-" + uuid.NewString(),
			ComputeState: models.NewExecutionState(models.ExecutionStateNew),
			DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning),
			NodeID:       node.PeerInfo.ID.String(),
		}
		newExecs[execution.ID] = execution
	}
	for _, exec := range newExecs {
		plan.AppendExecution(exec)
	}
	return newExecs, nil
}

func (b *OpsJobScheduler) selectNodes(ctx context.Context, job *models.Job) ([]models.NodeInfo, error) {
	nodeIDs, err := b.nodeDiscoverer.FindNodes(ctx, *job)
	if err != nil {
		return nil, err
	}
	log.Ctx(ctx).Debug().Int("Discovered", len(nodeIDs)).Msg("Found nodes for job")

	rankedNodes, err := b.nodeRanker.RankNodes(ctx, *job, nodeIDs)
	if err != nil {
		return nil, err
	}

	// filter nodes with rank below 0
	var filteredNodes []models.NodeInfo
	for _, nodeRank := range rankedNodes {
		if nodeRank.MeetsRequirement() {
			filteredNodes = append(filteredNodes, nodeRank.NodeInfo)
		}
	}
	log.Ctx(ctx).Debug().Int("Ranked", len(filteredNodes)).Msg("Ranked nodes for job")

	if len(filteredNodes) == 0 {
		return nil, orchestrator.NewErrNoMatchingNodes()
	}

	return filteredNodes, nil
}

func (b *OpsJobScheduler) handleFailure(nonTerminalExecs execSet, failed execSet, plan *models.Plan, err error) {
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

// compile-time assertion that OpsJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &OpsJobScheduler{}
