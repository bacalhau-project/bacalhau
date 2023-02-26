package ranking

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/version"
	"github.com/rs/zerolog/log"
)

var developmentVersion = model.BuildVersionInfo{
	Major: "0", Minor: "0", GitVersion: version.DevelopmentGitVersion,
}

type MinVersionNodeRankerParams struct {
	MinVersion model.BuildVersionInfo
}
type MinVersionNodeRanker struct {
	minVersion model.BuildVersionInfo
}

func NewMinVersionNodeRanker(params MinVersionNodeRankerParams) *MinVersionNodeRanker {
	return &MinVersionNodeRanker{
		minVersion: params.MinVersion,
	}
}

func (s *MinVersionNodeRanker) RankNodes(ctx context.Context, job model.Job, nodes []model.NodeInfo) ([]requester.NodeRank, error) {
	ranks := make([]requester.NodeRank, len(nodes))
	for i, node := range nodes {
		rank := 10
		if !s.isCompatibleVersion(node.BacalhauVersion) {
			log.Ctx(ctx).Trace().Msgf("filtering node %s with old bacalhau version %+v", node.PeerInfo.ID, node.BacalhauVersion)
			rank = -1
		}
		ranks[i] = requester.NodeRank{
			NodeInfo: node,
			Rank:     rank,
		}
	}
	return ranks, nil
}

func (s *MinVersionNodeRanker) isCompatibleVersion(nodeVersion model.BuildVersionInfo) bool {
	if s.IsDevelopmentVersion(nodeVersion) {
		return true
	}
	if nodeVersion.Major < s.minVersion.Major {
		return false
	}
	if nodeVersion.Major == s.minVersion.Major && nodeVersion.Minor < s.minVersion.Minor {
		return false
	}
	if nodeVersion.Major == s.minVersion.Major && nodeVersion.Minor == s.minVersion.Minor && nodeVersion.GitVersion < s.minVersion.GitVersion {
		return false
	}
	return true
}

func (s *MinVersionNodeRanker) IsDevelopmentVersion(nodeVersion model.BuildVersionInfo) bool {
	if nodeVersion.Major != developmentVersion.Major {
		return false
	}
	if nodeVersion.Minor != developmentVersion.Minor {
		return false
	}
	if nodeVersion.GitVersion != developmentVersion.GitVersion {
		return false
	}
	return true
}
