package ranking

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/rs/zerolog/log"
)

// featureNodeRanker is a generic ranker that can rank nodes based on what
// features (engines, publishers, verifiers, storage sources) are installed.
type featureNodeRanker[Key model.ProviderKey] struct {
	getJobRequirement   func(model.Job) []Key
	getNodeProvidedKeys func(model.ComputeNodeInfo) []Key
}

func NewEnginesNodeRanker() *featureNodeRanker[model.Engine] {
	return &featureNodeRanker[model.Engine]{
		getJobRequirement:   func(job model.Job) []model.Engine { return []model.Engine{job.Spec.Engine} },
		getNodeProvidedKeys: func(ni model.ComputeNodeInfo) []model.Engine { return ni.ExecutionEngines },
	}
}

func NewVerifiersNodeRanker() *featureNodeRanker[model.Verifier] {
	return &featureNodeRanker[model.Verifier]{
		getJobRequirement:   func(j model.Job) []model.Verifier { return []model.Verifier{j.Spec.Verifier} },
		getNodeProvidedKeys: func(ni model.ComputeNodeInfo) []model.Verifier { return ni.Verifiers },
	}
}

func NewPublishersNodeRanker() *featureNodeRanker[model.Publisher] {
	return &featureNodeRanker[model.Publisher]{
		getJobRequirement:   func(j model.Job) []model.Publisher { return []model.Publisher{j.Spec.PublisherSpec.Type} },
		getNodeProvidedKeys: func(ni model.ComputeNodeInfo) []model.Publisher { return ni.Publishers },
	}
}

func NewStoragesNodeRanker() *featureNodeRanker[model.StorageSourceType] {
	return &featureNodeRanker[model.StorageSourceType]{
		getJobRequirement: func(j model.Job) []model.StorageSourceType {
			specs := j.Spec.AllStorageSpecs()
			types := make([]model.StorageSourceType, 0, len(specs))
			for _, spec := range specs {
				if spec != nil && model.IsValidStorageSourceType(spec.StorageSource) {
					types = append(types, spec.StorageSource)
				}
			}
			return types
		},
		getNodeProvidedKeys: func(ni model.ComputeNodeInfo) []model.StorageSourceType { return ni.StorageSources },
	}
}

// rankNode ranks a single node based on the features the compute node is accepting.
// - Rank 10: Node is supporting the type(s) the job is requiring.
// - Rank 0: We don't have information on what the node supports.
// - Rank -1: Node is not supporting a type the job is requiring.
func (s *featureNodeRanker[Key]) rankNode(ctx context.Context, node model.NodeInfo, requiredKeys []Key) int {
	if node.ComputeNodeInfo == nil {
		// Node supported types are not set, or the node was discovered not
		// through nodeInfoPublisher (e.g. identity protocol). We will give the
		// node the benefit of the doubt and ask it to bid.
		return requester.RankPossible
	}

	providedKeys := s.getNodeProvidedKeys(*node.ComputeNodeInfo)
	for _, requiredKey := range requiredKeys {
		found := false
		for _, providedKey := range providedKeys {
			if providedKey == requiredKey {
				found = true
				break
			}
		}

		log.Ctx(ctx).Trace().Stringer("Requirement", requiredKey).Bool("Supported", found).Send()
		if !found {
			// Target wasn't found – we can end early as we won't use this node.
			return requester.RankUnsuitable
		}
	}

	// Node provides all the specified required types.
	return requester.RankPreferred
}

func (s *featureNodeRanker[Key]) RankNodes(
	ctx context.Context,
	job model.Job,
	nodes []model.NodeInfo,
) ([]requester.NodeRank, error) {
	ranks := make([]requester.NodeRank, len(nodes))
	requiredKeys := s.getJobRequirement(job)

	for i, node := range nodes {
		ctx := log.Ctx(ctx).With().Stringer("TargetNode", node.PeerInfo).Logger().WithContext(ctx) //nolint:govet
		rank := s.rankNode(ctx, node, requiredKeys)

		log.Ctx(ctx).Trace().Int("Rank", rank).Msg("Rank completed")
		ranks[i] = requester.NodeRank{
			NodeInfo: node,
			Rank:     rank,
		}
	}
	return ranks, nil
}
