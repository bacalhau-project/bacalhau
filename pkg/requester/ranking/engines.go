package ranking

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/rs/zerolog/log"
)

type EnginesNodeRanker struct {
}

func NewEnginesNodeRanker() *EnginesNodeRanker {
	return &EnginesNodeRanker{}
}

// RankNodes ranks nodes based on the engines the compute nodes are accepting:
// - Rank 10: Node is supporting the engine the job is using.
// - Rank -1: Node is not supporting the engine the job is using.
// - Rank 0: Node supported engines are not set, or the node was discovered not through nodeInfoPublisher (e.g. identity protocol)
func (s *EnginesNodeRanker) RankNodes(ctx context.Context, job model.Job, nodes []model.NodeInfo) ([]requester.NodeRank, error) {
	ranks := make([]requester.NodeRank, len(nodes))
	for i, node := range nodes {
		rank := 0
		if len(node.ComputeNodeInfo.ExecutionEngines) != 0 {
			for _, engine := range node.ComputeNodeInfo.ExecutionEngines {
				if engine == job.Spec.Engine {
					rank = 10
					break
				}
			}
			// engine wasn't found
			if rank == 0 {
				log.Ctx(ctx).Trace().Msgf("filtering node %s doesn't support engine %s", node.PeerInfo.ID, job.Spec.Engine)
				rank = -1
			}
		}
		ranks[i] = requester.NodeRank{
			NodeInfo: node,
			Rank:     rank,
		}
	}
	return ranks, nil
}
