package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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
	// ResultAccepted accepts a result for a given executionID, which will trigger publishing the result to the
	// destination specified in the job.
	ResultAccepted(context.Context, ResultAcceptedRequest) (ResultAcceptedResponse, error)
	// ResultRejected rejects a result for a given executionID.
	ResultRejected(context.Context, ResultRejectedRequest) (ResultRejectedResponse, error)
	// CancelExecution cancels a job for a given executionID.
	CancelExecution(context.Context, CancelExecutionRequest) (CancelExecutionResponse, error)
	// ExecutionLogs returns the address of a suitable log server
	ExecutionLogs(context.Context, ExecutionLogsRequest) (ExecutionLogsResponse, error)
}

// Executor Backend service that is responsible for running and publishing executions.
// Implementations can be synchronous or asynchronous by using Callbacks.
type Executor interface {
	// Run triggers the execution of a job.
	Run(ctx context.Context, execution store.Execution) error
	// Publish publishes the result of a job execution.
	Publish(ctx context.Context, execution store.Execution) error
	// Cancel cancels the execution of a job.
	Cancel(ctx context.Context, execution store.Execution) error
}

// Callback Callbacks are used to notify the caller of the result of a job execution.
type Callback interface {
	OnRunComplete(ctx context.Context, result RunResult)
	OnPublishComplete(ctx context.Context, result PublishResult)
	OnCancelComplete(ctx context.Context, result CancelResult)
	OnComputeFailure(ctx context.Context, err ComputeError)
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

func NewExecutionMetadata(execution store.Execution) ExecutionMetadata {
	return ExecutionMetadata{
		ExecutionID: execution.ID,
		JobID:       execution.Job.Metadata.ID,
	}
}

type AskForBidRequest struct {
	RoutingMetadata
	// Job specifies the job to be executed.
	Job model.Job
}

type AskForBidResponse struct {
	ExecutionMetadata
	Accepted bool
	Reason   string
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

type ResultAcceptedRequest struct {
	RoutingMetadata
	ExecutionID string
}

type ResultAcceptedResponse struct {
	ExecutionMetadata
}

type ResultRejectedRequest struct {
	RoutingMetadata
	ExecutionID   string
	Justification string
}

type ResultRejectedResponse struct {
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
	WithHistory bool
}

type ExecutionLogsResponse struct {
	Address           string
	ExecutionFinished bool
}

///////////////////////////////////
// Callback result models
///////////////////////////////////

// RunResult Result of a job execution that is returned to the caller through a Callback.
type RunResult struct {
	RoutingMetadata
	ExecutionMetadata
	ResultProposal   []byte
	RunCommandResult *model.RunCommandResult
}

// PublishResult Result of a job publish that is returned to the caller through a Callback.
type PublishResult struct {
	RoutingMetadata
	ExecutionMetadata
	PublishResult model.StorageSpec
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
