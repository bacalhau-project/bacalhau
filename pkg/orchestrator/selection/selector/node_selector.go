package selector

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
)

type NodeSelectorParams struct {
	NodeDiscoverer orchestrator.NodeDiscoverer
	NodeRanker     orchestrator.NodeRanker
}

type NodeSelector struct {
	nodeDiscoverer orchestrator.NodeDiscoverer
	nodeRanker     orchestrator.NodeRanker
}

func NewNodeSelector(params NodeSelectorParams) *NodeSelector {
	return &NodeSelector{
		nodeDiscoverer: params.NodeDiscoverer,
		nodeRanker:     params.NodeRanker,
	}
}

func (n NodeSelector) AllNodes(ctx context.Context) ([]models.NodeInfo, error) {
	nodeStates, err := n.nodeDiscoverer.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list discovered nodes: %w", err)
	}
	// extract slice of models.NodeInfo from slice of routing.NodeLiveness
	nodeInfos := make([]models.NodeInfo, 0, len(nodeStates))
	for _, ns := range nodeStates {
		nodeInfos = append(nodeInfos, ns.Info)
	}
	return nodeInfos, nil
}

func (n NodeSelector) AllMatchingNodes(ctx context.Context,
	job *models.Job,
	constraints *orchestrator.NodeSelectionConstraints) ([]models.NodeInfo, error) {
	filteredNodes, _, err := n.rankAndFilterNodes(ctx, job, constraints)
	if err != nil {
		return nil, err
	}

	nodeInfos := generic.Map(filteredNodes, func(nr orchestrator.NodeRank) models.NodeInfo { return nr.NodeInfo })
	return nodeInfos, nil
}

func (n NodeSelector) TopMatchingNodes(ctx context.Context,
	job *models.Job, desiredCount int,
	constraints *orchestrator.NodeSelectionConstraints) ([]models.NodeInfo, error) {
	possibleNodes, rejectedNodes, err := n.rankAndFilterNodes(ctx, job, constraints)
	if err != nil {
		return nil, err
	}

	if len(possibleNodes) < desiredCount {
		// TODO: evaluate if we should run the job if some nodes where found
		err = orchestrator.NewErrNotEnoughNodes(desiredCount, append(possibleNodes, rejectedNodes...))
		return nil, err
	}

	sort.Slice(possibleNodes, func(i, j int) bool {
		return possibleNodes[i].Rank > possibleNodes[j].Rank
	})

	selectedNodes := possibleNodes[:math.Min(len(possibleNodes), desiredCount)]
	selectedInfos := generic.Map(selectedNodes, func(nr orchestrator.NodeRank) models.NodeInfo { return nr.NodeInfo })
	return selectedInfos, nil
}

func (n NodeSelector) rankAndFilterNodes(ctx context.Context,
	job *models.Job,
	constraints *orchestrator.NodeSelectionConstraints) (selected, rejected []orchestrator.NodeRank, err error) {
	listed, err := n.nodeDiscoverer.List(ctx)
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

		if constraints.RequireApproval && nodeState.Approval != models.NodeApprovals.APPROVED {
			return false
		}

		if constraints.RequireConnected && nodeState.Liveness != models.NodeStates.CONNECTED {
			return false
		}

		return true
	})

	if len(nodeStates) == 0 {
		return nil, nil, errors.New("unable to find any connected and approved nodes")
	}

	// extract the nodeInfo from the slice of node states for ranking
	nodeInfos := make([]models.NodeInfo, 0, len(nodeStates))
	for _, ns := range nodeStates {
		nodeInfos = append(nodeInfos, ns.Info)
	}

	rankedNodes, err := n.nodeRanker.RankNodes(ctx, *job, nodeInfos)
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
