package ranking

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

var developmentVersion = types.BuildVersionInfo{
	Major: "0", Minor: "0", GitVersion: version.DevelopmentGitVersion,
}

var nilVersion = types.BuildVersionInfo{}

type MinVersionNodeRankerParams struct {
	MinVersion types.BuildVersionInfo
}
type MinVersionNodeRanker struct {
	minVersion types.BuildVersionInfo
}

func NewMinVersionNodeRanker(params MinVersionNodeRankerParams) *MinVersionNodeRanker {
	return &MinVersionNodeRanker{
		minVersion: params.MinVersion,
	}
}

func (s *MinVersionNodeRanker) RankNodes(ctx context.Context, job models.Job, nodes []models.NodeInfo) ([]orchestrator.NodeRank, error) {
	ranks := make([]orchestrator.NodeRank, len(nodes))
	for i, node := range nodes {
		rank := orchestrator.RankPreferred
		reason := "Bacalhau version compatible"
		// TODO: nodes discovered through identity protocol will have nil version
		//  this is a temporary fix to avoid filtering them out until we no longer depend on identity protocol for node discovery in our tests.
		if s.match(nilVersion, node.BacalhauVersion) {
			rank = orchestrator.RankPossible
			reason = "Bacalhau version unknown"
		} else if !s.isCompatibleVersion(node.BacalhauVersion) {
			rank = orchestrator.RankUnsuitable
			reason = "Bacalhau version is incompatible"
		}
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

func (s *MinVersionNodeRanker) isCompatibleVersion(nodeVersion models.BuildVersionInfo) bool {
	// we assume development version is always latest and compatible
	if s.match(developmentVersion, nodeVersion) {
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
func (s *MinVersionNodeRanker) match(v1 types.BuildVersionInfo, v2 models.BuildVersionInfo) bool {
	return v1.Major == v2.Major && v1.Minor == v2.Minor && v1.GitVersion == v2.GitVersion
}
