package selector

import (
	"context"
	"errors"
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
	return n.nodeDiscoverer.ListNodes(ctx)
}

func (n NodeSelector) AllMatchingNodes(ctx context.Context,
	job *models.Job,
	options ...orchestrator.NodeSelectionOption) ([]models.NodeInfo, error) {
	filteredNodes, _, err := n.rankAndFilterNodes(ctx, job, options...)
	if err != nil {
		return nil, err
	}

	nodeInfos := generic.Map(filteredNodes, func(nr orchestrator.NodeRank) models.NodeInfo { return nr.NodeInfo })
	return nodeInfos, nil
}

func (n NodeSelector) TopMatchingNodes(ctx context.Context,
	job *models.Job, desiredCount int,
	options ...orchestrator.NodeSelectionOption) ([]models.NodeInfo, error) {
	possibleNodes, rejectedNodes, err := n.rankAndFilterNodes(ctx, job, options...)
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
	options ...orchestrator.NodeSelectionOption) (selected, rejected []orchestrator.NodeRank, err error) {
	listed, err := n.nodeDiscoverer.ListNodes(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Apply constraints on the state of the nodes we want to select, but allow
	// the caller to override them.
	constraints := &orchestrator.NodeSelectionConstraint{
		RequireApproval:  true,
		RequireConnected: true,
	}
	for _, opt := range options {
		opt(constraints)
	}

	nodeIDs := lo.Filter(listed, func(nodeInfo models.NodeInfo, index int) bool {
		if nodeInfo.NodeType != models.NodeTypeCompute {
			return false
		}

		if constraints.RequireApproval && nodeInfo.Approval != models.NodeApprovals.APPROVED {
			return false
		}

		if constraints.RequireConnected && nodeInfo.State != models.NodeStates.CONNECTED {
			return false
		}

		return true
	})

	if len(nodeIDs) == 0 {
		return nil, nil, errors.New("unable to find any connected and approved nodes")
	}

	rankedNodes, err := n.nodeRanker.RankNodes(ctx, *job, nodeIDs)
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
