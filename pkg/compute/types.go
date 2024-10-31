//go:generate mockgen --source types.go --destination mocks.go --package compute
package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

// Endpoint is the frontend and entry point to the compute node. Requesters, whether through API, CLI or other means, do
// interact with the frontend service to submit jobs, ask for bids, accept or reject bids, etc.
type Endpoint interface {
	// AskForBid asks for a bid for a given job, which will assign executionID to the job and return a bid
	// is interested in bidding on.
	AskForBid(context.Context, messages.AskForBidRequest) (messages.AskForBidResponse, error)
	// BidAccepted accepts a bid for a given executionID, which will trigger executing the job in the backend.
	// The execution can be synchronous or asynchronous, depending on the backend implementation.
	BidAccepted(context.Context, messages.BidAcceptedRequest) (messages.BidAcceptedResponse, error)
	// BidRejected rejects a bid for a given executionID.
	BidRejected(context.Context, messages.BidRejectedRequest) (messages.BidRejectedResponse, error)
	// CancelExecution cancels a job for a given executionID.
	CancelExecution(context.Context, messages.CancelExecutionRequest) (messages.CancelExecutionResponse, error)
	// ExecutionLogs returns the address of a suitable log server
	ExecutionLogs(ctx context.Context, request messages.ExecutionLogsRequest) (<-chan *concurrency.AsyncResult[models.ExecutionLog], error)
}

// Executor Backend service that is responsible for running and publishing executions.
// Implementations can be synchronous or asynchronous by using Callbacks.
type Executor interface {
	// Run triggers the execution of a job.
	Run(ctx context.Context, execution *models.Execution) error
	// Cancel cancels the execution of a job.
	Cancel(ctx context.Context, execution *models.Execution) error
}

// Callback Callbacks are used to notify the caller of the result of a job execution.
type Callback interface {
	OnBidComplete(ctx context.Context, result messages.BidResult)
	OnRunComplete(ctx context.Context, result messages.RunResult)
	OnCancelComplete(ctx context.Context, result messages.CancelResult)
	OnComputeFailure(ctx context.Context, err messages.ComputeError)
}

// ManagementEndpoint is the transport-based interface for compute nodes to
// register with the requester node, update information and perform heartbeats.
type ManagementEndpoint interface {
	// Register registers a compute node with the requester node.
	Register(context.Context, messages.RegisterRequest) (*messages.RegisterResponse, error)
	// UpdateInfo sends an update of node info to the requester node
	UpdateInfo(context.Context, messages.UpdateInfoRequest) (*messages.UpdateInfoResponse, error)
	// UpdateResources updates the resources currently in use by a specific node
	UpdateResources(context.Context, messages.UpdateResourcesRequest) (*messages.UpdateResourcesResponse, error)
}
