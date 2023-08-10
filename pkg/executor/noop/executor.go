package noop

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
)

type ExecutorHandlerIsInstalled func(ctx context.Context) (bool, error)
type ExecutorHandlerHasStorageLocally func(ctx context.Context, volume models.StorageSpec) (bool, error)
type ExecutorHandlerGetVolumeSize func(ctx context.Context, volume models.StorageSpec) (uint64, error)
type ExecutorHandlerGetBidStrategy func(ctx context.Context) (bidstrategy.BidStrategy, error)
type ExecutorHandlerJobHandler func(ctx context.Context, jobID string, resultsDir string) (*models.RunCommandResult, error)

func ErrorJobHandler(err error) ExecutorHandlerJobHandler {
	return func(ctx context.Context, jobID string, resultsDir string) (*models.RunCommandResult, error) {
		return nil, err
	}
}

func DelayedJobHandler(sleep time.Duration) ExecutorHandlerJobHandler {
	return func(ctx context.Context, jobID string, resultsDir string) (*models.RunCommandResult, error) {
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
	Jobs   []string
	Config ExecutorConfig
}

func (e *NoopExecutor) Cancel(ctx context.Context, id string) error {
	return nil
}

func NewNoopExecutor() *NoopExecutor {
	Executor := &NoopExecutor{
		Jobs: []string{},
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

func (e *NoopExecutor) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	if e.Config.ExternalHooks.GetBidStrategy != nil {
		handler := e.Config.ExternalHooks.GetBidStrategy
		strategy, err := handler(ctx)
		if err != nil {
			return bidstrategy.BidStrategyResponse{}, err
		}
		return strategy.ShouldBid(ctx, request)
	}
	return semantic.NewChainedSemanticBidStrategy().ShouldBid(ctx, request)
}

func (e *NoopExecutor) ShouldBidBasedOnUsage(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	usage models.Resources,
) (bidstrategy.BidStrategyResponse, error) {
	if e.Config.ExternalHooks.GetBidStrategy != nil {
		handler := e.Config.ExternalHooks.GetBidStrategy
		strategy, err := handler(ctx)
		if err != nil {
			return bidstrategy.BidStrategyResponse{}, err
		}
		return strategy.ShouldBidBasedOnUsage(ctx, request, usage)
	}
	// TODO(forrest): [correctness] this returns the correct response, but could be made specific to this method.
	return semantic.NewChainedSemanticBidStrategy().ShouldBid(ctx, request)
}

func (e *NoopExecutor) Run(
	ctx context.Context,
	args *executor.RunCommandRequest,
) (*models.RunCommandResult, error) {
	e.Jobs = append(e.Jobs, args.JobID)
	if e.Config.ExternalHooks.JobHandler != nil {
		handler := e.Config.ExternalHooks.JobHandler
		return handler(ctx, args.JobID, args.ResultsDir)
	}
	return &models.RunCommandResult{}, nil
}

func (e *NoopExecutor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented for NoopExecutor")
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*NoopExecutor)(nil)
