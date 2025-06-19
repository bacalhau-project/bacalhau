package selector

import (
	"context"
	"fmt"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
)

type NodeSelector struct {
	discoverer  nodes.Lookup
	ranker      orchestrator.NodeRanker
	constraints orchestrator.NodeSelectionConstraints
}

func NewNodeSelector(
	discoverer nodes.Lookup,
	ranker orchestrator.NodeRanker,
	constraints orchestrator.NodeSelectionConstraints,
) *NodeSelector {
	return &NodeSelector{
		discoverer:  discoverer,
		ranker:      ranker,
		constraints: constraints,
	}
}

func (n NodeSelector) AllNodes(ctx context.Context) ([]models.NodeInfo, error) {
	nodeStates, err := n.discoverer.List(ctx, nodes.HealthyNodeFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list discovered nodes: %w", err)
	}
	// extract slice of models.NodeInfo from slice of models.NodeConnectionState
	nodeInfos := make([]models.NodeInfo, 0, len(nodeStates))
	for _, ns := range nodeStates {
		nodeInfos = append(nodeInfos, ns.Info)
	}
	return nodeInfos, nil
}

func (n NodeSelector) MatchingNodes(
	ctx context.Context,
	job *models.Job,
) (matchingNodes, rejectedNodes []orchestrator.NodeRank, err error) {
	matchingNodes, rejectedNodes, err = n.rankAndFilterNodes(ctx, job)
	if err != nil {
		return
	}

	sort.Slice(matchingNodes, func(i, j int) bool {
		return matchingNodes[i].Rank > matchingNodes[j].Rank
	})
	return matchingNodes, rejectedNodes, nil
}

func (n NodeSelector) rankAndFilterNodes(
	ctx context.Context,
	job *models.Job,
) (selected, rejected []orchestrator.NodeRank, err error) {
	listed, err := n.discoverer.List(ctx)
	if err != nil {
		return nil, nil, err
	}

	// filter node states to return a slice of nodes that are:
	// - compute nodes
	// - approved to executor jobs
	// - connected (alive)
	nodeStates := lo.Filter(listed, func(nodeState models.NodeState, index int) bool {
		if nodeState.Info.NodeType != models.NodeTypeCompute {
			return false
		}

		if n.constraints.RequireApproval && nodeState.Membership != models.NodeMembership.APPROVED {
			return false
		}

		if n.constraints.RequireConnected && nodeState.ConnectionState.Status != models.NodeStates.CONNECTED {
			return false
		}

		return true
	})

	// extract the nodeInfo from the slice of node states for ranking
	nodeInfos := make([]models.NodeInfo, 0, len(nodeStates))
	for _, ns := range nodeStates {
		nodeInfos = append(nodeInfos, ns.Info)
	}

	rankedNodes, err := n.ranker.RankNodes(ctx, *job, nodeInfos)
	if err != nil {
		return nil, nil, err
	}

	// filter nodes with rank below 0
	for _, nodeRank := range rankedNodes {
		if nodeRank.MeetsRequirement() {
			selected = append(selected, nodeRank)
		} else {
			rejected = append(rejected, nodeRank)
		}
	}
	log.Ctx(ctx).Debug().Int("Matched", len(selected)).Int("Rejected", len(rejected)).Msg("Matched nodes for job")
	return selected, rejected, nil
}

// compile-time interface assertions
var _ orchestrator.NodeSelector = (*NodeSelector)(nil)
