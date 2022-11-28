package noop

import (
	"context"
	"fmt"

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
	IsBadActor    bool
	ExternalHooks ExecutorConfigExternalHooks
}

type NoopExecutorProvider struct {
	noopExecutor *NoopExecutor
}

func NewNoopExecutorProvider(noopExecutor *NoopExecutor) *NoopExecutorProvider {
	return &NoopExecutorProvider{
		noopExecutor: noopExecutor,
	}
}

func (p *NoopExecutorProvider) AddExecutor(ctx context.Context, engineType model.Engine, executor executor.Executor) error {
	return fmt.Errorf("noop executor provider does not support adding executors")
}

func (p *NoopExecutorProvider) GetExecutor(ctx context.Context, engineType model.Engine) (executor.Executor, error) {
	// NoopExecutorProvider also support Docker engine in addition to Noop as some tests use `docker run` cli command
	// to submit jobs, and we don't have a noop cli option.
	if engineType != model.EngineNoop && engineType != model.EngineDocker {
		return nil, fmt.Errorf("noop executor doesn't support %s", engineType)
	}
	return p.noopExecutor, nil
}

func (p *NoopExecutorProvider) HasExecutor(ctx context.Context, engineType model.Engine) bool {
	_, err := p.GetExecutor(ctx, engineType)
	return err == nil
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

func (e *NoopExecutor) CancelShard(ctx context.Context, shard model.JobShard) error {
	return nil
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.ExecutorProvider = (*NoopExecutorProvider)(nil)
var _ executor.Executor = (*NoopExecutor)(nil)
