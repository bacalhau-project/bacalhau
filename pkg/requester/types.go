//go:generate mockgen --source types.go --destination mocks.go --package requester
package requester

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
)

// RegistrationEndpoint is the transport-based interface for compute nodes to
// register with the requester node.
type RegistrationEndpoint interface {
	// Register registers a compute node with the requester node.
	Register(context.Context, requests.RegisterRequest) error
}

// Endpoint is the frontend and entry point to the requester node for the end users to submit, update and cancel jobs.
type Endpoint interface {
	// SubmitJob submits a new job to the network.
	SubmitJob(context.Context, model.JobCreatePayload) (*model.Job, error)
	// CancelJob cancels an existing job.
	CancelJob(context.Context, CancelJobRequest) (CancelJobResult, error)
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
	Tail        bool
	Follow      bool
}

type ReadLogsResponse struct {
	Address           string
	ExecutionComplete bool
}
