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
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// BatchJobScheduler is a scheduler for batch jobs that run until completion
type BatchJobScheduler struct {
	jobStore       jobstore.Store
	planner        orchestrator.Planner
	nodeDiscoverer requester.NodeDiscoverer
	nodeRanker     requester.NodeRanker
}

type BatchJobSchedulerParams struct {
	JobStore       jobstore.Store
	Planner        orchestrator.Planner
	NodeDiscoverer requester.NodeDiscoverer
	NodeRanker     requester.NodeRanker
}

func NewBatchJobScheduler(params *BatchJobSchedulerParams) *BatchJobScheduler {
	return &BatchJobScheduler{
		jobStore:       params.JobStore,
		planner:        params.Planner,
		nodeDiscoverer: params.NodeDiscoverer,
		nodeRanker:     params.NodeRanker,
	}
}

func (b *BatchJobScheduler) Process(ctx context.Context, evaluation *models.Evaluation) error {
	ctx = log.Ctx(ctx).With().Str("JobID", evaluation.JobID).Str("EvalID", evaluation.ID).Logger().WithContext(ctx)

	// Plan to hold the actions to be taken
	plan := models.NewPlan(evaluation)

	// Retrieve the job state
	jobState, err := b.jobStore.GetJobState(ctx, evaluation.JobID)
	if err != nil {
		return fmt.Errorf("failed to retrieve job state for job %s when evaluating %s: %w",
			evaluation.JobID, evaluation, err)
	}
	nonTerminalExecutions := jobState.NonTerminalExecutions()

	// early exit if the job is stopped
	if jobState.State.IsTerminal() {
		for _, execution := range nonTerminalExecutions {
			plan.AppendStoppedExecution(execution, model.ExecutionStateCancelled, execNotNeeded)
		}
		return b.planner.Process(ctx, plan)
	}

	// Retrieve the info for all the nodes that have executions for this job
	nodeInfos, err := b.existingNodeInfos(ctx, nonTerminalExecutions)
	if err != nil {
		return err
	}

	// Mark executions that are running on nodes that are not healthy as failed
	err = b.failUnhealthyExecs(ctx, nonTerminalExecutions, nodeInfos, plan)
	if err != nil {
		return err
	}

	job, err := b.jobStore.GetJob(ctx, evaluation.JobID)
	if err != nil {
		return fmt.Errorf("failed to retrieve job %s: %w", evaluation.JobID, err)
	}

	// create new executions if needed
	activeExecutionsCount := len(nonTerminalExecutions) - plan.StoppedExecutionsCount()
	desiredExecutionCount := job.Spec.Deal.Concurrency - jobState.CompletedCount() - activeExecutionsCount
	if desiredExecutionCount > 0 {
		selectedNodes, err := b.selectNodes(ctx, &job, desiredExecutionCount)
		if err != nil {
			return err
		}
		for i := 0; i < desiredExecutionCount; i++ {
			executionID := model.ExecutionID{
				JobID:       job.Metadata.ID,
				NodeID:      selectedNodes[i].PeerInfo.ID.String(),
				ExecutionID: "e-" + uuid.NewString(),
			}
			execution := &model.ExecutionState{
				JobID:            executionID.JobID,
				NodeID:           executionID.NodeID,
				ComputeReference: executionID.ExecutionID,
				State:            model.ExecutionStateAskForBid,
			}
			plan.AppendExecution(execution)
		}
	}
	// stop executions if we over-subscribed and exceeded the desired number of executions
	if desiredExecutionCount < 0 {
		candidateExecutions := lo.Filter(nonTerminalExecutions, func(execution *model.ExecutionState, _ int) bool {
			return !plan.IsExecutionStopped(execution)
		})
		// TODO: keep track of execution ranks and kill the worst ones
		// Using version as indicator of which execution has made more progress
		sort.Slice(candidateExecutions, func(i, j int) bool {
			return candidateExecutions[i].Version < candidateExecutions[j].Version
		})
		for i := 0; i < math.Abs(desiredExecutionCount); i++ {
			plan.AppendStoppedExecution(candidateExecutions[i], model.ExecutionStateCancelled, execNotNeeded)
		}
	}

	return b.planner.Process(ctx, plan)
}

// failUnhealthyExecs marks executions that are running on nodes that are not healthy as failed
func (b *BatchJobScheduler) failUnhealthyExecs(
	ctx context.Context, existingExecutions []*model.ExecutionState, nodeInfos map[string]*model.NodeInfo, plan *models.Plan) error {
	for _, execution := range existingExecutions {
		if _, ok := nodeInfos[execution.NodeID]; !ok {
			plan.AppendStoppedExecution(execution, model.ExecutionStateFailed, execLost)
		}
	}
	return nil
}

// existingNodeInfos returns a map of nodeID to NodeInfo for all the nodes that have executions for this job
func (b *BatchJobScheduler) existingNodeInfos(ctx context.Context, existingExecutions []*model.ExecutionState) (map[string]*model.NodeInfo, error) {
	out := make(map[string]*model.NodeInfo)
	checked := make(map[string]struct{})

	// TODO: implement a better way to retrieve node info instead of listing all nodes
	nodesMap := make(map[string]*model.NodeInfo)
	discoveredNodes, err := b.nodeDiscoverer.ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	for _, node := range discoveredNodes {
		nodesMap[string(node.PeerInfo.ID)] = &node
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
	var filteredNodes []requester.NodeRank
	for _, nodeRank := range rankedNodes {
		if nodeRank.MeetsRequirement() {
			filteredNodes = append(filteredNodes, nodeRank)
		}
	}
	log.Ctx(ctx).Debug().Int("Ranked", len(filteredNodes)).Msg("Ranked nodes for job")

	if len(filteredNodes) < desiredCount {
		// TODO: evaluate if we should run the job if some nodes where found
		err = requester.NewErrNotEnoughNodes(desiredCount, rankedNodes)
		return nil, err
	}

	sort.Slice(filteredNodes, func(i, j int) bool {
		return filteredNodes[i].Rank > filteredNodes[j].Rank
	})

	selectedNodes := filteredNodes[:math.Min(len(filteredNodes), desiredCount)]
	selectedInfos := generic.Map(selectedNodes, func(nr requester.NodeRank) model.NodeInfo { return nr.NodeInfo })
	return selectedInfos, nil
}

// compile-time assertion that BatchJobScheduler satisfies the Scheduler interface
var _ orchestrator.Scheduler = &BatchJobScheduler{}
