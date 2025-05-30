package ranking

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	modelsutils "github.com/bacalhau-project/bacalhau/pkg/models/utils"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

// featureNodeRanker is a generic ranker that can rank nodes based on what
// features (engines, publishers, storage sources) are installed.
type featureNodeRanker struct {
	getJobRequirement   func(models.Job) []string
	getNodeProvidedKeys func(models.ComputeNodeInfo) []string
}

func NewEnginesNodeRanker() *featureNodeRanker {
	return &featureNodeRanker{
		getJobRequirement: func(job models.Job) []string {
			return []string{job.Task().Engine.Type}
		},
		getNodeProvidedKeys: func(ni models.ComputeNodeInfo) []string { return ni.ExecutionEngines },
	}
}

func NewPublishersNodeRanker() *featureNodeRanker {
	return &featureNodeRanker{
		getJobRequirement: func(j models.Job) []string {
			// publisher is optional and can be empty
			if j.Task().Publisher.Type == "" {
				return []string{}
			}
			return []string{j.Task().Publisher.Type}
		},
		getNodeProvidedKeys: func(ni models.ComputeNodeInfo) []string { return ni.Publishers },
	}
}

func NewStoragesNodeRanker() *featureNodeRanker {
	return &featureNodeRanker{
		getJobRequirement: func(j models.Job) []string {
			return modelsutils.AllInputSourcesTypes(&j)
		},
		getNodeProvidedKeys: func(ni models.ComputeNodeInfo) []string { return ni.StorageSources },
	}
}

// rankNode ranks a single node based on the features the compute node is accepting.
// - Rank 10: Node is supporting the type(s) the job is requiring.
// - Rank -1: Node is not supporting a type the job is requiring.
func (s *featureNodeRanker) rankNode(ctx context.Context, node models.NodeInfo, requiredKeys []string) (rank int, reason string) {
	providedKeys := s.getNodeProvidedKeys(node.ComputeNodeInfo)
	for _, requiredKey := range requiredKeys {
		found := false
		for _, providedKey := range providedKeys {
			if strings.EqualFold(providedKey, requiredKey) {
				found = true
				break
			}
		}

		if !found {
			// Target wasn't found – we can end early as we won't use this node.
			availableKeys := strings.Join(providedKeys, ", ")
			return orchestrator.RankUnsuitable, fmt.Sprintf("does not support %s, only %s", requiredKey, availableKeys)
		}
	}

	// Node provides all the specified required types.
	return orchestrator.RankPreferred, "provides all the specified required types"
}

func (s *featureNodeRanker) RankNodes(
	ctx context.Context,
	job models.Job,
	nodes []models.NodeInfo,
) ([]orchestrator.NodeRank, error) {
	ranks := make([]orchestrator.NodeRank, len(nodes))
	requiredKeys := s.getJobRequirement(job)

	for i, node := range nodes {
		rank, reason := s.rankNode(ctx, node, requiredKeys)
		ranks[i] = orchestrator.NodeRank{
			NodeInfo:  node,
			Rank:      rank,
			Reason:    reason,
			Retryable: false,
		}
		log.Ctx(ctx).Trace().Object("Rank", ranks[i]).Msg("Ranked node")
	}
	return ranks, nil
}
