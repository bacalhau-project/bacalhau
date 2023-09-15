package selector

import (
	"context"
	"sort"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/rs/zerolog/log"
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

func (n NodeSelector) AllMatchingNodes(ctx context.Context, job *models.Job) ([]models.NodeInfo, error) {
	filteredNodes, err := n.rankAndFilterNodes(ctx, job)
	if err != nil {
		return nil, err
	}

	nodeInfos := generic.Map(filteredNodes, func(nr orchestrator.NodeRank) models.NodeInfo { return nr.NodeInfo })
	return nodeInfos, nil
}
func (n NodeSelector) TopMatchingNodes(ctx context.Context, job *models.Job, desiredCount int) ([]models.NodeInfo, error) {
	filteredNodes, err := n.rankAndFilterNodes(ctx, job)
	if err != nil {
		return nil, err
	}
	if len(filteredNodes) < desiredCount {
		// TODO: evaluate if we should run the job if some nodes where found
		err = orchestrator.NewErrNotEnoughNodes(desiredCount, filteredNodes)
		return nil, err
	}

	sort.Slice(filteredNodes, func(i, j int) bool {
		return filteredNodes[i].Rank > filteredNodes[j].Rank
	})

	selectedNodes := filteredNodes[:math.Min(len(filteredNodes), desiredCount)]
	selectedInfos := generic.Map(selectedNodes, func(nr orchestrator.NodeRank) models.NodeInfo { return nr.NodeInfo })
	return selectedInfos, nil
}

func (n NodeSelector) rankAndFilterNodes(ctx context.Context, job *models.Job) ([]orchestrator.NodeRank, error) {
	nodeIDs, err := n.nodeDiscoverer.FindNodes(ctx, *job)
	if err != nil {
		return nil, err
	}

	rankedNodes, err := n.nodeRanker.RankNodes(ctx, *job, nodeIDs)
	if err != nil {
		return nil, err
	}

	// filter nodes with rank bellow 0
	var filteredNodes []orchestrator.NodeRank
	for _, nodeRank := range rankedNodes {
		if nodeRank.MeetsRequirement() {
			filteredNodes = append(filteredNodes, nodeRank)
		}
	}
	log.Ctx(ctx).Debug().Int("Matched", len(filteredNodes)).Msg("Matched nodes for job")
	return filteredNodes, nil
}

// compile-time interface assertions
var _ orchestrator.NodeSelector = (*NodeSelector)(nil)
