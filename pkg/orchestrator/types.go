package orchestrator

import (
	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type SubmitJobRequest struct {
	Job                  *models.Job
	ClientInstanceID     string
	ClientInstallationID string
	Force                bool
}

type SubmitJobResponse struct {
	JobID        string
	EvaluationID string
	Warnings     []string
}

type DiffJobRequest struct {
	Job *models.Job
}

type DiffJobResponse struct {
	Diff     string
	Warnings []string
}

type StopJobRequest struct {
	JobID         string
	Namespace     string
	Reason        string
	UserTriggered bool
}

type StopJobResponse struct {
	EvaluationID string
}

type RerunJobRequest struct {
	JobIDOrName string
	JobVersion  uint64
	Namespace   string
	Reason      string
}

type RerunJobResponse struct {
	JobID        string
	JobVersion   uint64
	EvaluationID string
	Warnings     []string
}

type ReadLogsRequest struct {
	JobID          string
	Namespace      string
	JobVersion     uint64
	AllJobVersions bool
	ExecutionID    string
	Tail           bool
	Follow         bool
}

type ReadLogsResponse struct {
	Address           string
	ExecutionComplete bool
}

type GetResultsRequest struct {
	JobID     string
	Namespace string
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

	// Retryable should be true only if the system could defer this job until
	// later and the rank could change without any human intervention on the
	// assessed node. I.e. it should only reflect transient things like node
	// usage, capacity or approval status.
	//
	// E.g. if this node is excluded because it does not support a required
	// feature, this could be fixed if the feature was configured at the other
	// node, but Retryable should be false because this is unlikely to happen
	// over the lifetime of the job.
	Retryable bool
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
	JobID     string
	Namespace string
}

type NodeSelectionConstraints struct {
	RequireConnected bool
	RequireApproval  bool
}
