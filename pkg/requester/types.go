//go:generate mockgen --source types.go --destination mocks.go --package requester
package requester

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/rs/zerolog"
)

// Endpoint is the frontend and entry point to the requester node for the end users to submit, update and cancel jobs.
type Endpoint interface {
	// SubmitJob submits a new job to the network.
	SubmitJob(context.Context, model.JobCreatePayload) (*model.Job, error)
	// CancelJob cancels an existing job.
	CancelJob(context.Context, CancelJobRequest) (CancelJobResult, error)
	// ReadLogs retrieves the logs for an execution
	ReadLogs(context.Context, ReadLogsRequest) (ReadLogsResponse, error)
}

// Scheduler distributes jobs to the compute nodes and tracks the executions.
type Scheduler interface {
	StartJob(context.Context, StartJobRequest) error
	CancelJob(context.Context, CancelJobRequest) (CancelJobResult, error)
}

type Queue interface {
	Scheduler

	EnqueueJob(context.Context, model.Job) error
}

// NodeDiscoverer discovers nodes in the network that are suitable to execute a job.
type NodeDiscoverer interface {
	ListNodes(ctx context.Context) ([]model.NodeInfo, error)
	FindNodes(ctx context.Context, job model.Job) ([]model.NodeInfo, error)
}

// NodeRanker ranks nodes based on their suitability to execute a job.
type NodeRanker interface {
	RankNodes(ctx context.Context, job model.Job, nodes []model.NodeInfo) ([]NodeRank, error)
}

// NodeSelector chooses appropriate nodes for to execute a job.
type NodeSelector interface {
	// SelectNodes returns the nodes that should be used to execute the passed job.
	SelectNodes(context.Context, *model.Job) ([]model.NodeInfo, error)
	// SelectNodesForRetry returns the nodes that should be used to retry the
	// passed failed executions in the context of the passed job. If no nodes
	// are returned, the executions do not need to be retried just yet.
	SelectNodesForRetry(context.Context, *model.Job, *model.JobState) ([]model.NodeInfo, error)
	// SelectBids returns the pending bids on the passed job that should be
	// accepted or rejected.
	SelectBids(context.Context, *model.Job, *model.JobState) (accept, reject []model.ExecutionState)
	// CanCompleteJob returns whether the passed job is ready to be declared
	// complete. The returned job state should be used to update the state of
	// the job, and may be JobStateCompleted or JobStateCompletedPartially.
	CanCompleteJob(context.Context, *model.Job, *model.JobState) (bool, model.JobStateType)
}

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

// StartJobRequest triggers the scheduling of a job.
type StartJobRequest struct {
	Job model.Job
}

type CancelJobRequest struct {
	JobID         string
	Reason        string
	UserTriggered bool
}

type CancelJobResult struct{}

type ReadLogsRequest struct {
	JobID       string
	ExecutionID string
	WithHistory bool
	Follow      bool
}

type ReadLogsResponse struct {
	Address           string
	ExecutionComplete bool
}
