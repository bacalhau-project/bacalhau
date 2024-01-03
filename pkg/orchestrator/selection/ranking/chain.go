package ranking

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

// Chain assigns a random rank to each node to allow the orchestrator to select random top nodes
// for job execution.
type Chain struct {
	rankers []orchestrator.NodeRanker
}

func NewChain() *Chain {
	return &Chain{}
}

// Add ranker to the chain
func (c *Chain) Add(ranker ...orchestrator.NodeRanker) {
	c.rankers = append(c.rankers, ranker...)
}

func (c *Chain) RankNodes(ctx context.Context, job models.Job, nodes []models.NodeInfo) ([]orchestrator.NodeRank, error) {
	// initialize map of node ranks
	ranksMap := make(map[string]*orchestrator.NodeRank, len(nodes))
	for _, node := range nodes {
		ranksMap[node.ID()] = &orchestrator.NodeRank{NodeInfo: node, Rank: orchestrator.RankPossible}
	}

	// iterate over the rankers and add their ranks to the map
	// once a node is ranked below zero, it is not considered for job execution and the rank will never be increased above zero
	// by other rankers. It can only go down more
	for _, ranker := range c.rankers {
		nodeRanks, err := ranker.RankNodes(ctx, job, nodes)
		if err != nil {
			return nil, err
		}
		for _, nodeRank := range nodeRanks {
			if !nodeRank.MeetsRequirement() {
				ranksMap[nodeRank.NodeInfo.ID()].Rank = orchestrator.RankUnsuitable
				ranksMap[nodeRank.NodeInfo.ID()].Reason = nodeRank.Reason
			} else if ranksMap[nodeRank.NodeInfo.ID()].MeetsRequirement() {
				ranksMap[nodeRank.NodeInfo.ID()].Rank += nodeRank.Rank
			}
		}
	}

	nodeRanks := make([]orchestrator.NodeRank, 0, len(ranksMap))
	for _, nodeRank := range ranksMap {
		nodeRanks = append(nodeRanks, *nodeRank)
	}
	return nodeRanks, nil
}
