package selection

import (
	"context"
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

const maxRetriesPerNode = 3

type AllNodeSelectorParams struct {
	NodeDiscoverer requester.NodeDiscoverer
	NodeRanker     requester.NodeRanker
}

type allNodeSelector struct {
	nodeDiscoverer requester.NodeDiscoverer
	nodeRanker     requester.NodeRanker
}

// The AllNodeSelector implements a node selection algorithm that executes the
// job on all nodes that meet the job requirements.
func NewAllNodeSelector(params AllNodeSelectorParams) requester.NodeSelector {
	return &allNodeSelector{nodeDiscoverer: params.NodeDiscoverer, nodeRanker: params.NodeRanker}
}

// We want to select all nodes that match the job requirements. We assume from
// this point onwards that there is >= 1 execution for each node that passed the
// node rank so that we don't have to keep track of which nodes passed in a
// store.
func (s *allNodeSelector) SelectNodes(ctx context.Context, job *model.Job) ([]model.NodeInfo, error) {
	ctx = log.Ctx(ctx).With().Str("JobID", job.ID()).Logger().WithContext(ctx)

	nodeIDs, err := s.nodeDiscoverer.FindNodes(ctx, *job)
	if err != nil {
		return nil, err
	}
	log.Ctx(ctx).Debug().Int("Discovered", len(nodeIDs)).Msg("Found nodes for job")

	rankedNodes, err := s.nodeRanker.RankNodes(ctx, *job, nodeIDs)
	if err != nil {
		return nil, err
	}

	// filter nodes that are unsuitable
	filteredNodes := lo.FilterMap(rankedNodes, func(node requester.NodeRank, _ int) (model.NodeInfo, bool) {
		return node.NodeInfo, node.MeetsRequirement()
	})
	return filteredNodes, nil
}

// We asked all nodes that should be capable of running the job to bid, so we
// want to select all the nodes that said they would run the job.
func (*allNodeSelector) SelectBids(ctx context.Context, job *model.Job, jobState *model.JobState) (accept, reject []model.ExecutionState) {
	return jobState.GroupExecutionsByState()[model.ExecutionStateAskForBidAccepted], nil
}

// We specifically want to run the job on *all* nodes – so we need to only retry
// executions on the *same* node on which it failed. There would be no point /
// danger in scheduling the execution to a different node.
//
// So we just select all of the nodes that have *only* failed executions (a node
// that failed and then retried successfully should not be retried again!). This
// only helps work around transient errors (e.g. something random wrong at the
// node) and not permanent failures, so there is a limit to the number of times
// we should retry.
func (s *allNodeSelector) SelectNodesForRetry(ctx context.Context, job *model.Job, jobState *model.JobState) ([]model.NodeInfo, error) {
	// failedNodes counts the number of failures that the node has seen so far.
	// A negative value means the job is successful or in progress.
	failureCounts := make(map[string]int, len(jobState.Executions))
	for _, execution := range jobState.Executions {
		numFails, seen := failureCounts[execution.NodeID]
		if !seen {
			numFails = 0
		}

		if numFails < 0 || !inRetryableState(ctx, execution) {
			// The job is in progress or completed: don't retry this node again.
			failureCounts[execution.NodeID] = -1
		} else {
			// The execution failed – increment counter.
			failureCounts[execution.NodeID] = numFails + 1
		}
	}

	failedNodes := make([]string, 0, len(jobState.Executions))
	for nodeID, failureCount := range failureCounts {
		if failureCount > 0 {
			failedNodes = append(failedNodes, nodeID)
		}
	}

	log.Ctx(ctx).Debug().Int("Retryable", len(failedNodes)).Msg("Found nodes to retry")
	foundNodes, err := s.nodeDiscoverer.FindNodes(ctx, *job)
	if err != nil {
		return nil, err
	}
	log.Ctx(ctx).Debug().Int("Discovered", len(foundNodes)).Msg("Found nodes for job")

	nodeErrors := make([]error, 0, len(failedNodes))
	retryNodes := make([]model.NodeInfo, 0, len(failedNodes))
	for _, failedNode := range failedNodes {
		id, err := peer.Decode(failedNode)
		if err != nil {
			nodeErrors = append(nodeErrors, err)
			continue
		}

		idx := slices.IndexFunc[model.NodeInfo](foundNodes, func(ni model.NodeInfo) bool {
			return ni.PeerInfo.ID == id
		})

		if idx < 0 {
			nodeErrors = append(nodeErrors, requester.NewErrNodeNotFound(id))
		} else if failureCounts[failedNode] >= maxRetriesPerNode {
			nodeErrors = append(nodeErrors, requester.NewErrTooManyRetries(failureCounts[failedNode]))
		} else {
			retryNodes = append(retryNodes, foundNodes[idx])
		}
	}

	if len(nodeErrors) > 0 {
		// A node failed but has disappeared or a node is over the retry limit
		return nil, errors.Join(nodeErrors...)
	}

	return retryNodes, nil
}

// We can complete the job if all the nodes completed successfully. We can
// partially complete the job if at least one did.
func (*allNodeSelector) CanCompleteJob(ctx context.Context, job *model.Job, jobState *model.JobState) (bool, model.JobStateType) {
	completedJobs := jobState.CompletedCount()
	if completedJobs < 1 {
		return false, jobState.State
	}

	uniqueNodes := lo.UniqBy(jobState.Executions, func(state model.ExecutionState) string { return state.NodeID })
	if completedJobs >= len(uniqueNodes) {
		return true, model.JobStateCompleted
	}

	return false, jobState.State
}

// An execution is ready to be retried if it is not currently executing or
// completed, and is in a state where retrying could change the result.
func inRetryableState(ctx context.Context, executionState model.ExecutionState) bool {
	// States are split out below for documentation purposes.
	switch executionState.State {
	case model.ExecutionStateAskForBidRejected:
		// The node may have been overloaded but could run the job later.
		return true
	case model.ExecutionStateBidRejected:
		// We shouldn't ever reject a bid in this algorithm because we want all
		// nodes to run the job – so lets start a new execution to resolve this?
		log.Ctx(ctx).Warn().Stringer("State", executionState.State).Msg("Execution should not be in this state")
		return true
	case model.ExecutionStateFailed:
		// Something hopefully transient went wrong at the node – so try again.
		return true
	case model.ExecutionStateCancelled:
		// Presumably we don't want to retry if the job is canceled.
		return false
	default:
		// The other states should be success or in progress states.
		return false
	}
}

var _ requester.NodeSelector = (*allNodeSelector)(nil)
