package orchestrator

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/rs/zerolog"
)

type SubmitJobRequest struct {
	Job *models.Job
}

type SubmitJobResponse struct {
	JobID        string
	EvaluationID string
	Warnings     []string
}

type StopJobRequest struct {
	JobID         string
	Reason        string
	UserTriggered bool
}

type StopJobResponse struct {
	EvaluationID string
}

type ReadLogsRequest struct {
	JobID       string
	ExecutionID string
	Tail        bool
	Follow      bool
}

type ReadLogsResponse struct {
	Address           string
	ExecutionComplete bool
}

type GetResultsRequest struct {
	JobID string
}

type GetResultsResponse struct {
	Results []*models.SpecConfig
}

// NodeRank represents a node and its rank. The higher the rank, the more preferable a node is to execute the job.
// A negative rank means the node is not suitable to execute the job.
type NodeRank struct {
	NodeInfo models.NodeInfo
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
	e.Str("Node", r.NodeInfo.ID()).
		Bool("MeetsRequirement", r.MeetsRequirement()).
		Str("Reason", r.Reason)
}

type RetryRequest struct {
	JobID string
}

type NodeSelectionConstraint struct {
	RequireConnected bool
	RequireApproval  bool
}

type NodeSelectionOption func(*NodeSelectionConstraint)

func WithConnected(required bool) NodeSelectionOption {
	return func(c *NodeSelectionConstraint) {
		c.RequireConnected = required
	}
}

func WithApproval(required bool) NodeSelectionOption {
	return func(c *NodeSelectionConstraint) {
		c.RequireApproval = required
	}
}
