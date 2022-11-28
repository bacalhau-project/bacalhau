package backend

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

// Service Backend service that is responsible for running and publishing executions.
// Implementations can be synchronous or asynchronous by using Callbacks.
type Service interface {
	// Run triggers the execution of a job.
	Run(ctx context.Context, execution store.Execution) error
	// Publish publishes the result of a job execution.
	Publish(ctx context.Context, execution store.Execution) error
	// Cancel cancels the execution of a job.
	Cancel(ctx context.Context, execution store.Execution) error
}

// Callback Callbacks are used to notify the caller of the result of a job execution.
type Callback interface {
	OnRunSuccess(ctx context.Context, executionID string, result RunResult)
	OnRunFailure(ctx context.Context, executionID string, err error)
	OnPublishSuccess(ctx context.Context, executionID string, result PublishResult)
	OnPublishFailure(ctx context.Context, executionID string, err error)
	OnCancelSuccess(ctx context.Context, executionID string, result CancelResult)
	OnCancelFailure(ctx context.Context, executionID string, err error)
}

// RunResult Result of a job execution that is returned to the caller through a Callback.
type RunResult struct {
	ResultProposal   []byte
	RunCommandResult *model.RunCommandResult
}

// PublishResult Result of a job publish that is returned to the caller through a Callback.
type PublishResult struct {
	PublishResult model.StorageSpec
}

// CancelResult Result of a job cancel that is returned to the caller through a Callback.
type CancelResult struct {
}
