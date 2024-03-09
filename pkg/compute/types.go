//go:generate mockgen --source types.go --destination mocks.go --package compute
package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
)

// Endpoint is the frontend and entry point to the compute node. Requesters, whether through API, CLI or other means, do
// interact with the frontend service to submit jobs, ask for bids, accept or reject bids, etc.
type Endpoint interface {
	// AskForBid asks for a bid for a given job, which will assign executionID to the job and return a bid
	// is interested in bidding on.
	AskForBid(context.Context, AskForBidRequest) (AskForBidResponse, error)
	// BidAccepted accepts a bid for a given executionID, which will trigger executing the job in the backend.
	// The execution can be synchronous or asynchronous, depending on the backend implementation.
	BidAccepted(context.Context, BidAcceptedRequest) (BidAcceptedResponse, error)
	// BidRejected rejects a bid for a given executionID.
	BidRejected(context.Context, BidRejectedRequest) (BidRejectedResponse, error)
	// CancelExecution cancels a job for a given executionID.
	CancelExecution(context.Context, CancelExecutionRequest) (CancelExecutionResponse, error)
	// ExecutionLogs returns the address of a suitable log server
	ExecutionLogs(ctx context.Context, request ExecutionLogsRequest) (<-chan *concurrency.AsyncResult[models.ExecutionLog], error)
}

// Executor Backend service that is responsible for running and publishing executions.
// Implementations can be synchronous or asynchronous by using Callbacks.
type Executor interface {
	// Run triggers the execution of a job.
	Run(ctx context.Context, localExecutionState store.LocalExecutionState) error
	// Cancel cancels the execution of a job.
	Cancel(ctx context.Context, localExecutionState store.LocalExecutionState) error
}

// Callback Callbacks are used to notify the caller of the result of a job execution.
type Callback interface {
	OnBidComplete(ctx context.Context, result BidResult)
	OnRunComplete(ctx context.Context, result RunResult)
	OnCancelComplete(ctx context.Context, result CancelResult)
	OnComputeFailure(ctx context.Context, err ComputeError)
}

// ManagementEndpoint is the transport-based interface for compute nodes to
// register with the requester node, update information and perform heartbeats.
type ManagementEndpoint interface {
	// Register registers a compute node with the requester node.
	Register(context.Context, requests.RegisterRequest) (*requests.RegisterResponse, error)
	// UpdateInfo sends an update of node info to the requester node
	UpdateInfo(context.Context, requests.UpdateInfoRequest) (*requests.UpdateInfoResponse, error)
}

///////////////////////////////////
// Endpoint request/response models
///////////////////////////////////

type RoutingMetadata struct {
	SourcePeerID string
	TargetPeerID string
}

type ExecutionMetadata struct {
	ExecutionID string
	JobID       string
}

func NewExecutionMetadata(execution *models.Execution) ExecutionMetadata {
	return ExecutionMetadata{
		ExecutionID: execution.ID,
		JobID:       execution.Job.ID,
	}
}

type AskForBidRequest struct {
	RoutingMetadata
	// Execution specifies the job to be executed.
	Execution *models.Execution
	// WaitForApproval specifies whether the compute node should wait for the requester to approve the bid.
	// if set to true, the compute node will not start the execution until the requester approves the bid.
	// If set to false, the compute node will automatically start the execution after bidding and when resources are available.
	WaitForApproval bool
}

type AskForBidResponse struct {
	ExecutionMetadata
}

type BidAcceptedRequest struct {
	RoutingMetadata
	ExecutionID   string
	Accepted      bool
	Justification string
}

type BidAcceptedResponse struct {
	ExecutionMetadata
}

type BidRejectedRequest struct {
	RoutingMetadata
	ExecutionID   string
	Justification string
}

type BidRejectedResponse struct {
	ExecutionMetadata
}

type CancelExecutionRequest struct {
	RoutingMetadata
	ExecutionID   string
	Justification string
}

type CancelExecutionResponse struct {
	ExecutionMetadata
}

type ExecutionLogsRequest struct {
	RoutingMetadata
	ExecutionID string
	Tail        bool
	Follow      bool
}

type ExecutionLogsResponse struct {
	Address           string
	ExecutionFinished bool
}

///////////////////////////////////
// Callback result models
///////////////////////////////////

// BidResult is the result of the compute node bidding on a job that is returned
// to the caller through a Callback.
type BidResult struct {
	RoutingMetadata
	ExecutionMetadata
	Accepted bool
	Reason   string
}

// RunResult Result of a job execution that is returned to the caller through a Callback.
type RunResult struct {
	RoutingMetadata
	ExecutionMetadata
	PublishResult    *models.SpecConfig
	RunCommandResult *models.RunCommandResult
}

// CancelResult Result of a job cancel that is returned to the caller through a Callback.
type CancelResult struct {
	RoutingMetadata
	ExecutionMetadata
}

type ComputeError struct {
	RoutingMetadata
	ExecutionMetadata
	Err string
}

func (e ComputeError) Error() string {
	return e.Err
}
