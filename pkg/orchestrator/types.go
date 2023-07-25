package orchestrator

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/rs/zerolog"
)

// NodeRank represents a node and its rank. The higher the rank, the more preferable a node is to execute the job.
// A negative rank means the node is not suitable to execute the job.
type NodeRank struct {
	NodeInfo model.NodeInfo
	Rank     int
	Reason   string
}

const (
	// The node is known to be not suitable to execute the job.
	RankUnsuitable int = -1
	// The node's suitability to execute the job is not known, so we could ask
	// it to bid and hope that it is able to accept.
	RankPossible int = 0
	// The node is known to be suitable to execute the job, so we should prefer
	// using it if we can.
	RankPreferred int = 10
)

// Returns whether the node meets the requirements to run the job.
func (r NodeRank) MeetsRequirement() bool {
	return r.Rank > RankUnsuitable
}

func (r NodeRank) MarshalZerologObject(e *zerolog.Event) {
	e.Stringer("Node", r.NodeInfo.PeerInfo.ID).
		Bool("MeetsRequirement", r.MeetsRequirement()).
		Str("Reason", r.Reason)
}

type RetryRequest struct {
	JobID string
}
