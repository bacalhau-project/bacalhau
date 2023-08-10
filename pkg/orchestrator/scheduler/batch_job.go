package scheduler

import (
	"context"
	"fmt"
	"sort"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// BatchJobScheduler is a scheduler for batch jobs that run until completion
type BatchJobScheduler struct {
	jobStore       jobstore.Store
	planner        orchestrator.Planner
	nodeDiscoverer orchestrator.NodeDiscoverer
	nodeRanker     orchestrator.NodeRanker
	retryStrategy  orchestrator.RetryStrategy
}

type BatchJobSchedulerParams struct {
	JobStore       jobstore.Store
	Planner        orchestrator.Planner
	NodeDiscoverer orchestrator.NodeDiscoverer
	NodeRanker     orchestrator.NodeRanker
	RetryStrategy  orchestrator.RetryStrategy
}

func NewBatchJobScheduler(params BatchJobSchedulerParams) *BatchJobScheduler {
	return &BatchJobScheduler{
		jobStore:       params.JobStore,
		planner:        params.Planner,
		nodeDiscoverer: params.NodeDiscoverer,
		nodeRanker:     params.NodeRanker,
		retryStrategy:  params.RetryStrategy,
	}
}

func (b *BatchJobScheduler) Process(ctx context.Context, evaluation *models.Evaluation) error {
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

	// Approve/Reject nodes
	desiredRemainingCount := math.Max(0, job.Count-existingExecs.countCompleted())
	execsByApprovalStatus := nonTerminalExecs.filterByApprovalStatus(desiredRemainingCount)
	execsByApprovalStatus.toApprove.markApproved(plan)
	execsByApprovalStatus.toReject.markStopped(execRejected, plan)

	// create new executions if needed
	remainingExecutionCount := desiredRemainingCount - execsByApprovalStatus.activeCount()
	if remainingExecutionCount > 0 {
		allFailed := existingExecs.filterFailed().union(lost)
		var placementErr error
		if len(allFailed) > 0 && !b.retryStrategy.ShouldRetry(ctx, orchestrator.RetryRequest{JobID: job.ID}) {
			placementErr = fmt.Errorf("exceeded max retries for job %s", job.ID)
		} else {
			_, placementErr = b.createMissingExecs(ctx, remainingExecutionCount, &job, plan)
		}
		if placementErr != nil {
			b.handleFailure(nonTerminalExecs, allFailed, plan, placementErr)
			return b.planner.Process(ctx, plan)
		}
	}

	// stop executions if we over-subscribed and exceeded the desired number of executions
	_, overSubscriptions := execsByApprovalStatus.running.filterByOverSubscriptions(desiredRemainingCount)
	overSubscriptions.markStopped(execNotNeeded, plan)

	// mark job as completed
	if desiredRemainingCount <= 0 {
		plan.MarkJobCompleted()
	}
	return b.planner.Process(ctx, plan)
}

func (b *BatchJobScheduler) createMissingExecs(
	ctx context.Context, remainingExecutionCount int, job *models.Job, plan *models.Plan) (execSet, error) {
	newExecs := execSet{}
	for i := 0; i < remainingExecutionCount; i++ {
		execution := &models.Execution{
			JobID:        job.ID,
			ID:           "e-" + uuid.NewString(),
			ComputeState: models.NewExecutionState(models.ExecutionStateNew),
			DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStatePending),
		}
		newExecs[execution.ID] = execution
	}
	if len(newExecs) > 0 {
		err := b.placeExecs(ctx, newExecs, job)
		if err != nil {
			return newExecs, err
		}
	}
	for _, exec := range newExecs {
		plan.AppendExecution(exec)
	}
	return newExecs, nil
}

// placeExecs places the executions
func (b *BatchJobScheduler) placeExecs(ctx context.Context, execs execSet, job *models.Job) error {
	if len(execs) > 0 {
		selectedNodes, err := b.selectNodes(ctx, job, len(execs))
		if err != nil {
			return err
		}
		i := 0
		for _, exec := range execs {
			exec.NodeID = selectedNodes[i].PeerInfo.ID.String()
			i++
		}
	}
	return nil
}

func (b *BatchJobScheduler) selectNodes(ctx context.Context, job *models.Job, desiredCount int) ([]models.NodeInfo, error) {
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
	var filteredNodes []orchestrator.NodeRank
	for _, nodeRank := range rankedNodes {
		if nodeRank.MeetsRequirement() {
			filteredNodes = append(filteredNodes, nodeRank)
		}
	}
	log.Ctx(ctx).Debug().Int("Ranked", len(filteredNodes)).Msg("Ranked nodes for job")

	if len(filteredNodes) < desiredCount {
		// TODO: evaluate if we should run the job if some nodes where found
		err = orchestrator.NewErrNotEnoughNodes(desiredCount, rankedNodes)
		return nil, err
	}

	sort.Slice(filteredNodes, func(i, j int) bool {
		return filteredNodes[i].Rank > filteredNodes[j].Rank
	})

	selectedNodes := filteredNodes[:math.Min(len(filteredNodes), desiredCount)]
	selectedInfos := generic.Map(selectedNodes, func(nr orchestrator.NodeRank) models.NodeInfo { return nr.NodeInfo })
	return selectedInfos, nil
}

func (b *BatchJobScheduler) handleFailure(nonTerminalExecs execSet, failed execSet, plan *models.Plan, err error) {
	// TODO: allow scheduling retries in a later time if don't find nodes instead of failing the job
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

// compile-time assertion that BatchJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &BatchJobScheduler{}
