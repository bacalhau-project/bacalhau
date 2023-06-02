package ranking

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Chain assigns a random rank to each node to allow the requester to select random top nodes
// for job execution.
type Chain struct {
	rankers []requester.NodeRanker
}

func NewChain() *Chain {
	return &Chain{}
}

// Add ranker to the chain
func (c *Chain) Add(ranker ...requester.NodeRanker) {
	c.rankers = append(c.rankers, ranker...)
}

func (c *Chain) RankNodes(ctx context.Context, job model.Job, nodes []model.NodeInfo) ([]requester.NodeRank, error) {
	// initialize map of node ranks
	ranksMap := make(map[peer.ID]*requester.NodeRank, len(nodes))
	for _, node := range nodes {
		ranksMap[node.PeerInfo.ID] = &requester.NodeRank{NodeInfo: node, Rank: requester.RankPossible}
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
			if !ranksMap[nodeRank.NodeInfo.PeerInfo.ID].MeetsRequirement() || !nodeRank.MeetsRequirement() {
				ranksMap[nodeRank.NodeInfo.PeerInfo.ID].Rank = requester.RankUnsuitable
			} else {
				ranksMap[nodeRank.NodeInfo.PeerInfo.ID].Rank += nodeRank.Rank
			}
		}
	}

	nodeRanks := make([]requester.NodeRank, 0, len(ranksMap))
	for _, nodeRank := range ranksMap {
		nodeRanks = append(nodeRanks, *nodeRank)
	}
	return nodeRanks, nil
}
