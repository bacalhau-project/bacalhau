package requester

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/external"
)

// Endpoint is the frontend and entry point to the requester node for the end users to submit, update and cancel jobs.
type Endpoint interface {
	// SubmitJob submits a new job to the network.
	SubmitJob(context.Context, model.JobCreatePayload) (*model.Job, error)
	// ApproveJob approves or rejects the running of a job.
	ApproveJob(context.Context, bidstrategy.ModerateJobRequest) error
	// CancelJob cancels an existing job.
	CancelJob(context.Context, CancelJobRequest) (CancelJobResult, error)
	// VerifyExecutions approves or rejects the publishing of an execution.
	VerifyExecutions(context.Context, external.ExternalVerificationResponse) error
	// ReadLogs retrieves the logs for an execution
	ReadLogs(context.Context, ReadLogsRequest) (ReadLogsResponse, error)
}

// Scheduler distributes jobs to the compute nodes and tracks the executions.
type Scheduler interface {
	StartJob(context.Context, StartJobRequest) error
	CancelJob(context.Context, CancelJobRequest) (CancelJobResult, error)
	VerifyExecutions(context.Context, []verifier.VerifierResult) (succeeded, failed []verifier.VerifierResult)
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

// NodeRank represents a node and its rank. The higher the rank, the more preferable a node is to execute the job.
// A negative rank means the node is not suitable to execute the job.
type NodeRank struct {
	NodeInfo model.NodeInfo
	Rank     int
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
