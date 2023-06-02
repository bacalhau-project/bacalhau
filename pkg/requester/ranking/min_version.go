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

var nilVersion = model.BuildVersionInfo{}

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
		rank := requester.RankPreferred
		// TODO: nodes discovered through identity protocol will have nil version
		//  this is a temporary fix to avoid filtering them out until we no longer depend on identity protocol for node discovery in our tests.
		if s.match(node.BacalhauVersion, nilVersion) {
			rank = requester.RankPossible
		} else if !s.isCompatibleVersion(node.BacalhauVersion) {
			log.Ctx(ctx).Debug().Msgf("filtering node %s with old bacalhau version %+v", node.PeerInfo.ID, node.BacalhauVersion)
			rank = requester.RankUnsuitable
		}
		ranks[i] = requester.NodeRank{
			NodeInfo: node,
			Rank:     rank,
		}
	}
	return ranks, nil
}

func (s *MinVersionNodeRanker) isCompatibleVersion(nodeVersion model.BuildVersionInfo) bool {
	// we assume development version is always latest and compatible
	if s.match(nodeVersion, developmentVersion) {
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

// match return true if versions are equal. We only compare Major, Minor and GitVersion
func (s *MinVersionNodeRanker) match(v1, v2 model.BuildVersionInfo) bool {
	return v1.Major == v2.Major && v1.Minor == v2.Minor && v1.GitVersion == v2.GitVersion
}
