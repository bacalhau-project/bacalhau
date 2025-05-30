package noop

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
)

type ExecutorHandlerIsInstalled func(ctx context.Context) (bool, error)
type ExecutorHandlerHasStorageLocally func(ctx context.Context, volume models.InputSource) (bool, error)
type ExecutorHandlerGetVolumeSize func(ctx context.Context, volume models.InputSource) (uint64, error)
type ExecutorHandlerGetBidStrategy func(ctx context.Context) (bidstrategy.BidStrategy, error)
type ExecutorHandlerJobHandler func(ctx context.Context, execContext ExecutionContext) (*models.RunCommandResult, error)

// ExecutionContext provides all the context needed for execution
type ExecutionContext struct {
	*executor.RunCommandRequest
	ExecutionDir string
}

func ErrorJobHandler(err error) ExecutorHandlerJobHandler {
	return func(ctx context.Context, _ ExecutionContext) (*models.RunCommandResult, error) {
		return nil, err
	}
}

func DelayedJobHandler(sleep time.Duration) ExecutorHandlerJobHandler {
	return func(ctx context.Context, _ ExecutionContext) (*models.RunCommandResult, error) {
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
	Jobs     []string
	Config   ExecutorConfig
	handlers generic.SyncMap[string, *executionHandler]
}

type executionHandler struct {
	jobHandler   ExecutorHandlerJobHandler
	request      *executor.RunCommandRequest
	executionDir string
	done         chan bool
	result       *handlerResult
}

// isEmpty is a flag to indicate if the handler is empty
func (e *executionHandler) isEmpty() bool {
	return e.jobHandler == nil
}

type handlerResult struct {
	err    error
	result *models.RunCommandResult
}

func (e *executionHandler) run(ctx context.Context) {
	defer close(e.done)

	if e.isEmpty() {
		e.result = &handlerResult{
			err: nil,
			result: &models.RunCommandResult{
				STDOUT:          "",
				StdoutTruncated: false,
				STDERR:          "",
				StderrTruncated: false,
				ExitCode:        0,
				ErrorMsg:        "",
			},
		}
		return
	}

	execContext := ExecutionContext{
		RunCommandRequest: e.request,
		ExecutionDir:      e.executionDir,
	}

	// Fall back to original handler
	result, err := e.jobHandler(ctx, execContext)
	e.result = &handlerResult{
		err:    err,
		result: result,
	}
}

func (e *NoopExecutor) Start(ctx context.Context, request *executor.RunCommandRequest) error {
	log.Info().Msg("starting execution")
	e.Jobs = append(e.Jobs, request.JobID)

	handler := &executionHandler{
		jobHandler:   e.Config.ExternalHooks.JobHandler,
		request:      request,
		executionDir: request.ExecutionDir,
		done:         make(chan bool),
	}
	e.handlers.Put(request.ExecutionID, handler)
	go handler.run(ctx)
	return nil
}

func (e *NoopExecutor) Wait(ctx context.Context, executionID string) (<-chan *models.RunCommandResult, <-chan error) {
	handler, found := e.handlers.Get(executionID)
	resultC := make(chan *models.RunCommandResult, 1)
	errC := make(chan error, 1)

	if !found {
		errC <- NewNoopExecutorError(executor.ExecutionNotFound, fmt.Sprintf("waiting on execution (%s)", executionID))
		return resultC, errC
	}

	go e.doWait(ctx, resultC, errC, handler)
	return resultC, errC
}

func (e *NoopExecutor) doWait(ctx context.Context, out chan *models.RunCommandResult, errC <-chan error, handler *executionHandler) {
	defer close(out)
	select {
	case <-ctx.Done():
		out <- &models.RunCommandResult{ErrorMsg: ctx.Err().Error()}
	case <-handler.done:
		if handler.isEmpty() {
			out <- &models.RunCommandResult{}
			return
		}
		if handler.result.err != nil {
			out <- &models.RunCommandResult{
				STDOUT:          "",
				StdoutTruncated: false,
				STDERR:          "",
				StderrTruncated: false,
				ExitCode:        0,
				ErrorMsg:        handler.result.err.Error(),
			}
		} else {
			out <- &models.RunCommandResult{}
		}
	}
}

func (e *NoopExecutor) Cancel(ctx context.Context, id string) error {
	log.Info().Msg("cancel execution")
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
	return bidstrategy.NewBidResponse(true, ""), nil
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
	return bidstrategy.NewBidResponse(true, ""), nil
}

func (e *NoopExecutor) Run(
	ctx context.Context,
	args *executor.RunCommandRequest,
) (*models.RunCommandResult, error) {
	e.Jobs = append(e.Jobs, args.JobID)
	if err := e.Start(ctx, args); err != nil {
		return nil, err
	}
	resultC, errC := e.Wait(ctx, args.ExecutionID)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errC:
		return nil, err
	case out := <-resultC:
		return out, nil
	}
}

func (e *NoopExecutor) GetLogStream(ctx context.Context, request messages.ExecutionLogsRequest) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented for NoopExecutor")
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*NoopExecutor)(nil)
