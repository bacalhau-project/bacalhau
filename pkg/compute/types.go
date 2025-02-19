//go:generate mockgen --source types.go --destination mocks.go --package compute
package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

// Endpoint is the frontend and entry point to the compute node. Requesters, whether through API, CLI or other means, do
// interact with the frontend service to submit jobs, ask for bids, accept or reject bids, etc.
type Endpoint interface {
	// AskForBid asks for a bid for a given job, which will assign executionID to the job and return a bid
	// is interested in bidding on.
	AskForBid(context.Context, legacy.AskForBidRequest) (legacy.AskForBidResponse, error)
	// BidAccepted accepts a bid for a given executionID, which will trigger executing the job in the backend.
	// The execution can be synchronous or asynchronous, depending on the backend implementation.
	BidAccepted(context.Context, legacy.BidAcceptedRequest) (legacy.BidAcceptedResponse, error)
	// BidRejected rejects a bid for a given executionID.
	BidRejected(context.Context, legacy.BidRejectedRequest) (legacy.BidRejectedResponse, error)
	// CancelExecution cancels a job for a given executionID.
	CancelExecution(context.Context, legacy.CancelExecutionRequest) (legacy.CancelExecutionResponse, error)
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
	OnBidComplete(ctx context.Context, result legacy.BidResult)
	OnRunComplete(ctx context.Context, result legacy.RunResult)
	OnComputeFailure(ctx context.Context, err legacy.ComputeError)
}

// EnvVarResolver is the interface for resolving environment variable references
type EnvVarResolver interface {
	// Validate checks if the value has valid syntax for this resolver
	Validate(name string, value string) error

	// Value returns the resolved value from the source
	Value(value string) (string, error)
}

// PortAllocator manages network port allocation for container executions, ensuring safe
// concurrent allocation and preventing port conflicts between different jobs.
type PortAllocator interface {
	// AllocatePorts handles port allocation for a job execution's network configuration.
	// For each port mapping:
	// - Static ports (explicitly specified) are validated for availability
	// - Dynamic ports are automatically allocated from a configured range
	// - Target ports default to the host port if unspecified
	// Returns the complete list of port mappings or an error if allocation fails.
	AllocatePorts(execution *models.Execution) ([]models.PortMapping, error)

	// ReleasePorts releases all allocated ports for a given execution.
	ReleasePorts(execution *models.Execution)
}
