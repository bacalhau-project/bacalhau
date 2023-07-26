package scheduler

import (
	"context"
	"fmt"
	"sort"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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
	jobState, err := b.jobStore.GetJobState(ctx, evaluation.JobID)
	if err != nil {
		return fmt.Errorf("failed to retrieve job state for job %s when evaluating %s: %w",
			evaluation.JobID, evaluation, err)
	}

	// Plan to hold the actions to be taken
	plan := models.NewPlan(evaluation, &job, jobState.Version)

	existingExecs := execSetFromSliceOfValues(jobState.Executions)
	nonTerminalExecs := existingExecs.filterNonTerminal()

	// early exit if the job is stopped
	if jobState.State.IsTerminal() {
		b.markStopped(nonTerminalExecs, execNotNeeded, plan)
		return b.planner.Process(ctx, plan)
	}

	// Retrieve the info for all the nodes that have executions for this job
	nodeInfos, err := b.existingNodeInfos(ctx, nonTerminalExecs)
	if err != nil {
		return err
	}

	// Mark executions that are running on nodes that are not healthy as failed
	nonTerminalExecs, lost := b.filterUnhealthyExecs(nonTerminalExecs, nodeInfos)
	b.markStopped(lost, execLost, plan)

	// Approve/Reject nodes
	desiredRemainingCount := math.Max(0, job.Spec.Deal.Concurrency-existingExecs.countCompleted())
	running, toApprove, toReject, pending := b.filterPendingApproval(nonTerminalExecs, desiredRemainingCount)
	b.markApproved(toApprove, plan)
	b.markStopped(toReject, execRejected, plan)

	// create new executions if needed
	activeExecutionCount := len(running) + len(toApprove) + len(pending)
	remainingExecutionCount := desiredRemainingCount - activeExecutionCount
	if remainingExecutionCount > 0 {
		allFailed := existingExecs.filterFailed().union(lost)
		var placementErr error
		if len(allFailed) > 0 && !b.retryStrategy.ShouldRetry(ctx, orchestrator.RetryRequest{JobID: job.ID()}) {
			placementErr = fmt.Errorf("exceeded max retries for job %s", job.ID())
		} else {
			_, placementErr = b.createMissingExecs(ctx, remainingExecutionCount, &job, plan)
		}
		if placementErr != nil {
			b.handleFailure(nonTerminalExecs, allFailed, plan, placementErr)
			return b.planner.Process(ctx, plan)
		}
	}

	// stop executions if we over-subscribed and exceeded the desired number of executions
	_, overSubscriptions := b.filterOverSubscribedExecs(running, desiredRemainingCount)
	b.markStopped(overSubscriptions, execNotNeeded, plan)

	// mark job as completed
	if desiredRemainingCount <= 0 {
		plan.MarkJobCompleted()
	}
	return b.planner.Process(ctx, plan)
}

func (b *BatchJobScheduler) createMissingExecs(
	ctx context.Context, remainingExecutionCount int, job *model.Job, plan *models.Plan) (execSet, error) {
	newExecs := execSet{}
	for i := 0; i < remainingExecutionCount; i++ {
		execution := &model.ExecutionState{
			JobID:            job.Metadata.ID,
			ComputeReference: "e-" + uuid.NewString(),
			State:            model.ExecutionStateNew,
			DesiredState:     model.ExecutionDesiredStatePending,
		}
		newExecs[execution.ComputeReference] = execution
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
func (b *BatchJobScheduler) placeExecs(ctx context.Context, execs execSet, job *model.Job) error {
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

// existingNodeInfos returns a map of nodeID to NodeInfo for all the nodes that have executions for this job
func (b *BatchJobScheduler) existingNodeInfos(ctx context.Context, existingExecutions execSet) (map[string]*model.NodeInfo, error) {
	out := make(map[string]*model.NodeInfo)
	checked := make(map[string]struct{})

	// TODO: implement a better way to retrieve node info instead of listing all nodes
	nodesMap := make(map[string]*model.NodeInfo)
	discoveredNodes, err := b.nodeDiscoverer.ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	for i, node := range discoveredNodes {
		nodesMap[node.PeerInfo.ID.String()] = &discoveredNodes[i]
	}

	for _, execution := range existingExecutions {
		// keep track of the nodes that we already checked, including the nodes
		// that no longer exist in the node discoverer
		if _, ok := checked[execution.NodeID]; ok {
			continue
		}
		nodeInfo, ok := nodesMap[execution.NodeID]
		if ok {
			out[execution.NodeID] = nodeInfo
		}
		checked[execution.NodeID] = struct{}{}
	}
	return out, nil
}

func (b *BatchJobScheduler) selectNodes(ctx context.Context, job *model.Job, desiredCount int) ([]model.NodeInfo, error) {
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
	selectedInfos := generic.Map(selectedNodes, func(nr orchestrator.NodeRank) model.NodeInfo { return nr.NodeInfo })
	return selectedInfos, nil
}

// filterUnhealthyExecs marks executions that are running on nodes that are not healthy as failed
func (b *BatchJobScheduler) filterUnhealthyExecs(execs execSet, nodeInfos map[string]*model.NodeInfo) (execSet, execSet) {
	healthy := make(execSet)
	lost := make(execSet)

	for _, exec := range execs {
		if _, ok := nodeInfos[exec.NodeID]; !ok {
			lost[exec.ComputeReference] = exec
			log.Debug().Msgf("Execution %s is running on node %s which is not healthy", exec.ComputeReference, exec.NodeID)
		} else {
			healthy[exec.ComputeReference] = exec
		}
	}
	return healthy, lost
}

func (b *BatchJobScheduler) filterPendingApproval(set execSet, desiredCount int) (execSet, execSet, execSet, execSet) {
	running := set.filterRunning()
	toApprove := make(execSet)
	toReject := make(execSet)
	pending := make(execSet)

	//TODO: we are approving the oldest executions first, we should probably
	// approve the ones with highest rank first
	orderedExecs := set.ordered()

	// Approve/Reject nodes
	for _, exec := range orderedExecs {
		// nothing left to approve
		if (len(running) + len(toApprove)) >= desiredCount {
			break
		}
		if exec.State == model.ExecutionStateAskForBidAccepted {
			toApprove[exec.ComputeReference] = exec
		}
	}

	// reject the rest
	totalRunningCount := len(running) + len(toApprove)
	for _, exec := range orderedExecs {
		if running.has(exec.ComputeReference) || toApprove.has(exec.ComputeReference) {
			continue
		}
		if totalRunningCount >= desiredCount {
			toReject[exec.ComputeReference] = exec
		} else {
			pending[exec.ComputeReference] = exec
		}
	}
	return running, toApprove, toReject, pending
}

func (b *BatchJobScheduler) filterOverSubscribedExecs(running execSet, desiredCount int) (execSet, execSet) {
	remaining := make(execSet)
	overSubscriptions := make(execSet)

	count := 0
	for _, exec := range running {
		if count >= desiredCount {
			overSubscriptions[exec.ComputeReference] = exec
		} else {
			remaining[exec.ComputeReference] = exec
		}
		count++
	}
	return remaining, overSubscriptions
}

// markStopped
func (b *BatchJobScheduler) markStopped(set execSet, comment string, plan *models.Plan) {
	for _, exec := range set {
		plan.AppendStoppedExecution(exec, comment)
	}
}

// markStopped
func (b *BatchJobScheduler) markApproved(set execSet, plan *models.Plan) {
	for _, exec := range set {
		plan.AppendApprovedExecution(exec)
	}
}

func (b *BatchJobScheduler) handleFailure(nonTerminalExecs execSet, failed execSet, plan *models.Plan, err error) {
	// TODO: allow scheduling retries in a later time if don't find nodes instead of failing the job
	// mark all non-terminal executions as failed
	b.markStopped(nonTerminalExecs, jobFailed, plan)

	// mark the job as failed, using the error message of the latest failed execution, if any, or use
	// the error message passed by the scheduler
	latestErr := err.Error()
	if len(failed) > 0 {
		latestErr = failed.latest().Status
	}
	plan.MarkJobFailed(latestErr)
}

// compile-time assertion that BatchJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &BatchJobScheduler{}
