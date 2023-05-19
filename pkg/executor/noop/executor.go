package noop

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type ExecutorHandlerIsInstalled func(ctx context.Context) (bool, error)
type ExecutorHandlerHasStorageLocally func(ctx context.Context, volume model.StorageSpec) (bool, error)
type ExecutorHandlerGetVolumeSize func(ctx context.Context, volume model.StorageSpec) (uint64, error)
type ExecutorHandlerGetBidStrategy func(ctx context.Context) (bidstrategy.BidStrategy, error)
type ExecutorHandlerJobHandler func(ctx context.Context, job model.Job, resultsDir string) (*model.RunCommandResult, error)

func ErrorJobHandler(err error) ExecutorHandlerJobHandler {
	return func(ctx context.Context, job model.Job, resultsDir string) (*model.RunCommandResult, error) {
		return nil, err
	}
}

func DelayedJobHandler(sleep time.Duration) ExecutorHandlerJobHandler {
	return func(ctx context.Context, job model.Job, resultsDir string) (*model.RunCommandResult, error) {
		time.Sleep(sleep)
		return nil, nil
	}
}

type ExecutorConfigExternalHooks struct {
	IsInstalled       ExecutorHandlerIsInstalled
	HasStorageLocally ExecutorHandlerHasStorageLocally
	GetVolumeSize     ExecutorHandlerGetVolumeSize
	GetBidStrategy    ExecutorHandlerGetBidStrategy
	JobHandler        ExecutorHandlerJobHandler
}

type ExecutorConfig struct {
	ExternalHooks ExecutorConfigExternalHooks
}

type NoopExecutor struct {
	Jobs   []model.Job
	Config ExecutorConfig
}

func NewNoopExecutor() *NoopExecutor {
	Executor := &NoopExecutor{
		Jobs: []model.Job{},
	}
	return Executor
}

func NewNoopExecutorWithConfig(config ExecutorConfig) *NoopExecutor {
	e := NewNoopExecutor()
	e.Config = config
	return e
}

func (e *NoopExecutor) IsInstalled(ctx context.Context) (bool, error) {
	if e.Config.ExternalHooks.IsInstalled != nil {
		handler := e.Config.ExternalHooks.IsInstalled
		return handler(ctx)
	}
	return true, nil
}

func (e *NoopExecutor) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	if e.Config.ExternalHooks.HasStorageLocally != nil {
		handler := e.Config.ExternalHooks.HasStorageLocally
		return handler(ctx, volume)
	}
	return true, nil
}

func (e *NoopExecutor) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	if e.Config.ExternalHooks.GetVolumeSize != nil {
		handler := e.Config.ExternalHooks.GetVolumeSize
		return handler(ctx, volume)
	}
	return 0, nil
}

func (e *NoopExecutor) GetSemanticBidStrategy(ctx context.Context) (bidstrategy.SemanticBidStrategy, error) {
	if e.Config.ExternalHooks.GetBidStrategy != nil {
		handler := e.Config.ExternalHooks.GetBidStrategy
		return handler(ctx)
	}
	return semantic.NewChainedSemanticBidStrategy(), nil
}

func (e *NoopExecutor) GetResourceBidStrategy(ctx context.Context) (bidstrategy.ResourceBidStrategy, error) {
	if e.Config.ExternalHooks.GetBidStrategy != nil {
		handler := e.Config.ExternalHooks.GetBidStrategy
		return handler(ctx)
	}
	return resource.NewChainedResourceBidStrategy(), nil
}

func (e *NoopExecutor) Run(
	ctx context.Context,
	executionID string,
	job model.Job,
	jobResultsDir string,
) (*model.RunCommandResult, error) {
	e.Jobs = append(e.Jobs, job)
	if e.Config.ExternalHooks.JobHandler != nil {
		handler := e.Config.ExternalHooks.JobHandler
		return handler(ctx, job, jobResultsDir)
	}
	return &model.RunCommandResult{}, nil
}

func (e *NoopExecutor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented for NoopExecutor")
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*NoopExecutor)(nil)
