//go:generate mockgen --source types.go --destination mocks.go --package compute
package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
)

// Executor Backend service that is responsible for running and publishing executions.
// Implementations can be synchronous or asynchronous by using Callbacks.
type Executor interface {
	// Run triggers the execution of a job.
	Run(ctx context.Context, execution *models.Execution) error
	// Cancel cancels the execution of a job.
	Cancel(ctx context.Context, execution *models.Execution) error
}

// ManagementEndpoint is the transport-based interface for compute nodes to
// register with the requester node, update information and perform heartbeats.
type ManagementEndpoint interface {
	// Register registers a compute node with the requester node.
	Register(context.Context, requests.RegisterRequest) (*requests.RegisterResponse, error)
	// UpdateInfo sends an update of node info to the requester node
	UpdateInfo(context.Context, requests.UpdateInfoRequest) (*requests.UpdateInfoResponse, error)
	// UpdateResources updates the resources currently in use by a specific node
	UpdateResources(context.Context, requests.UpdateResourcesRequest) (*requests.UpdateResourcesResponse, error)
}

// /////////////////////////////////
// Endpoint request/response models
// /////////////////////////////////

// BaseRequest is the base request model for all requests.
type BaseRequest struct {
	Events []models.Event
}

// Message returns a request message if available.
func (r BaseRequest) Message() string {
	if len(r.Events) > 0 {
		return r.Events[0].Message
	}
	return ""
}

type AskForBidRequest struct {
	BaseRequest
	// Execution specifies the job to be executed.
	Execution *models.Execution
}

type BidAcceptedRequest struct {
	BaseRequest
	ExecutionID string
	Accepted    bool
}

type BidRejectedRequest struct {
	BaseRequest
	ExecutionID string
}

type CancelExecutionRequest struct {
	BaseRequest
	ExecutionID string
}

type ExecutionLogsResponse struct {
	Address           string
	ExecutionFinished bool
}

// /////////////////////////////////
// Callback result models
// /////////////////////////////////

type BaseResponse struct {
	ExecutionID string
	JobID       string
	JobType     string
	Events      []*models.Event
}

// Message returns a response message if available.
func (r BaseResponse) Message() string {
	if len(r.Events) > 0 {
		return r.Events[0].Message
	}
	return ""
}

// BidResult is the result of the compute node bidding on a job that is returned
// to the caller through a Callback.
type BidResult struct {
	BaseResponse
	Accepted bool
}

// RunResult Result of a job execution that is returned to the caller through a Callback.
type RunResult struct {
	BaseResponse
	PublishResult    *models.SpecConfig
	RunCommandResult *models.RunCommandResult
}

// CancelResult Result of a job cancel that is returned to the caller through a Callback.
type CancelResult struct {
	BaseResponse
}

type ComputeError struct {
	BaseResponse
}

func (e ComputeError) Error() string {
	return e.Message()
}
