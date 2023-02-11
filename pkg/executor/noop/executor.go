package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type ExecutorHandlerIsInstalled func(ctx context.Context) (bool, error)
type ExecutorHandlerHasStorageLocally func(ctx context.Context, volume model.StorageSpec) (bool, error)
type ExecutorHandlerGetVolumeSize func(ctx context.Context, volume model.StorageSpec) (uint64, error)
type ExecutorHandlerJobHandler func(ctx context.Context, shard model.JobShard, resultsDir string) (*model.RunCommandResult, error)

type ExecutorConfigExternalHooks struct {
	IsInstalled       ExecutorHandlerIsInstalled
	HasStorageLocally ExecutorHandlerHasStorageLocally
	GetVolumeSize     ExecutorHandlerGetVolumeSize
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

func (e *NoopExecutor) RunShard(
	ctx context.Context,
	shard model.JobShard,
	jobResultsDir string,
) (*model.RunCommandResult, error) {
	e.Jobs = append(e.Jobs, *shard.Job)
	if e.Config.ExternalHooks.JobHandler != nil {
		handler := e.Config.ExternalHooks.JobHandler
		return handler(ctx, shard, jobResultsDir)
	}
	return &model.RunCommandResult{}, nil
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*NoopExecutor)(nil)
