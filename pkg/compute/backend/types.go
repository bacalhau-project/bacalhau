package backend

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type Service interface {
	Run(ctx context.Context, execution store.Execution) error
	Publish(ctx context.Context, execution store.Execution) error
	Cancel(ctx context.Context, execution store.Execution) error
}

type Callback interface {
	OnRunSuccess(ctx context.Context, executionID string, result RunResult)
	OnRunFailure(ctx context.Context, executionID string, err error)
	OnPublishSuccess(ctx context.Context, executionID string, result PublishResult)
	OnPublishFailure(ctx context.Context, executionID string, err error)
	OnCancelSuccess(ctx context.Context, executionID string, result CancelResult)
	OnCancelFailure(ctx context.Context, executionID string, err error)
}

type RunResult struct {
	ResultProposal   []byte
	RunCommandResult *model.RunCommandResult
}

type PublishResult struct {
	PublishResult model.StorageSpec
}

type CancelResult struct {
}
