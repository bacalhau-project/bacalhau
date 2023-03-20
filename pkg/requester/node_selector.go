package requester

import (
	"context"
	"sort"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type NodeSelectorParams struct {
	NodeDiscoverer NodeDiscoverer
	NodeRanker     NodeRanker
}

type NodeSelector struct {
	nodeDiscoverer NodeDiscoverer
	nodeRanker     NodeRanker
}

func NewNodeSelector(params NodeSelectorParams) *NodeSelector {
	return &NodeSelector{
		nodeDiscoverer: params.NodeDiscoverer,
		nodeRanker:     params.NodeRanker,
	}
}

func (s *NodeSelector) SelectNodes(ctx context.Context, job model.Job, minCount, desiredCount int) ([]NodeRank, error) {
	nodeIDs, err := s.nodeDiscoverer.FindNodes(ctx, job)
	if err != nil {
		return nil, err
	}
	log.Ctx(ctx).Debug().Msgf("found %d nodes for job %s", len(nodeIDs), job.ID())

	rankedNodes, err := s.nodeRanker.RankNodes(ctx, job, nodeIDs)
	if err != nil {
		return nil, err
	}

	// filter nodes with rank below 0
	var filteredNodes []NodeRank
	for _, node := range rankedNodes {
		if node.Rank >= 0 {
			filteredNodes = append(filteredNodes, node)
		}
	}
	rankedNodes = filteredNodes
	log.Ctx(ctx).Debug().Msgf("ranked %d nodes for job %s", len(rankedNodes), job.ID())

	if len(rankedNodes) < minCount {
		err = NewErrNotEnoughNodes(minCount, len(rankedNodes))
		return nil, err
	}

	sort.Slice(rankedNodes, func(i, j int) bool {
		return rankedNodes[i].Rank > rankedNodes[j].Rank
	})

	selectedNodes := rankedNodes[:system.Min(len(rankedNodes), desiredCount)]
	return selectedNodes, nil
}
