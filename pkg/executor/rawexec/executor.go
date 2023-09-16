package rawexec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
)

const EngineName = "RawExec"

type EngineSpec struct {
	Command string
	Args    []string
	Env     []string
	Dir     string
}

func DecodeSpec(spec *models.SpecConfig) (EngineSpec, error) {
	if !spec.IsType(EngineName) {
		return EngineSpec{}, errors.New("invalid rawexec engine type. expected " + EngineName)
	}

	if spec.Params == nil {
		return EngineSpec{}, errors.New("invalid rawexec engine params. cannot be nil")
	}

	paramsBytes, err := json.Marshal(spec.Params)
	if err != nil {
		return EngineSpec{}, fmt.Errorf("failed to encode rawexec engine spec: %w", err)
	}

	var c EngineSpec
	if err := json.Unmarshal(paramsBytes, &c); err != nil {
		return EngineSpec{}, fmt.Errorf("failed to decode rawexec engine spec: %w", err)
	}
	return c, nil
}

type Executor struct {
	handlers generic.SyncMap[string, *executionHandler]
}

func (e *Executor) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (e *Executor) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	return bidstrategy.BidStrategyResponse{
		ShouldBid:  true,
		ShouldWait: false,
		Reason:     "",
	}, nil
}

func (e *Executor) ShouldBidBasedOnUsage(ctx context.Context, request bidstrategy.BidStrategyRequest, usage models.Resources) (bidstrategy.BidStrategyResponse, error) {
	return bidstrategy.BidStrategyResponse{
		ShouldBid:  true,
		ShouldWait: false,
		Reason:     "",
	}, nil
}

func (e *Executor) Start(ctx context.Context, request *executor.RunCommandRequest) error {
	log.Ctx(ctx).Info().
		Str("executionID", request.ExecutionID).
		Str("jobID", request.JobID).
		Msg("starting execution")

	if handler, found := e.handlers.Get(request.ExecutionID); found {
		if handler.active() {
			return fmt.Errorf("starting execution (%s): %w", request.ExecutionID, executor.ErrAlreadyStarted)
		} else {
			return fmt.Errorf("starting execution (%s): %w", request.ExecutionID, executor.ErrAlreadyComplete)
		}
	}

	execArgs, err := DecodeSpec(request.EngineParams)
	if err != nil {
		return fmt.Errorf("decoding raw exec engine spec: %w", err)
	}

	absPath, err := exec.LookPath(execArgs.Command)
	if err != nil {
		return fmt.Errorf("finding binary for command: %w", err)
	}

	/*
		The distinction between the Path field of the *Cmd struct and the first element in the Args
		slice is subtle but important:

		- Path: This specifies the path of the executable that will be run.
		- Args: This is a slice of strings representing the arguments to the command.
				The first element in this slice is, by convention, the command itself (i.e., how it's called).

		The reason for this convention is that in Unix-like operating systems, when a new process is spawned,
		it's given an array of arguments (argv in C). The program can use this to determine how it was called.
		For example, some programs behave differently depending on the name they're invoked with.

		In Go's os/exec package, this distinction allows for flexibility.
	*/

	args := append([]string{absPath}, execArgs.Args...)
	handler := &executionHandler{
		logger: log.With().
			Str("execution", request.ExecutionID).
			Str("job", request.JobID).
			Logger(),
		cmd: exec.Cmd{
			Path: absPath,
			Args: args,
			Env:  execArgs.Env,
			Dir:  execArgs.Dir,
		},
		executionID: request.ExecutionID,
		resultsDir:  request.ResultsDir,
		limits:      request.OutputLimits,
		activeCh:    make(chan bool),
		waitCh:      make(chan bool),
		running:     atomic.NewBool(false),
	}

	// register the handler for this executionID
	e.handlers.Put(request.ExecutionID, handler)
	// run the container.
	go handler.run(ctx)

	return nil
}

func (e *Executor) Wait(ctx context.Context, executionID string) (<-chan *models.RunCommandResult, <-chan error) {
	handler, found := e.handlers.Get(executionID)
	resultCh := make(chan *models.RunCommandResult, 1)
	errCh := make(chan error, 1)

	if !found {
		errCh <- fmt.Errorf("waiting on execution (%s): %w", executionID, executor.ErrNotFound)
		return resultCh, errCh
	}

	go e.doWait(ctx, resultCh, errCh, handler)
	return resultCh, errCh
}

func (e *Executor) Run(ctx context.Context, request *executor.RunCommandRequest) (*models.RunCommandResult, error) {
	if err := e.Start(ctx, request); err != nil {
		return nil, err
	}
	resCh, errCh := e.Wait(ctx, request.ExecutionID)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-resCh:
		return out, nil
	case err := <-errCh:
		return nil, err
	}
}

func (e *Executor) Cancel(ctx context.Context, executionID string) error {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return fmt.Errorf("canceling execution (%s): %w", executionID, executor.ErrNotFound)
	}
	return handler.kill(ctx)
}

func (e *Executor) doWait(ctx context.Context, out chan *models.RunCommandResult, errCh chan error, handle *executionHandler) {
	log.Info().Str("executionID", handle.executionID).Msg("waiting on execution")
	defer close(out)
	defer close(errCh)

	select {
	case <-ctx.Done():
		errCh <- ctx.Err() // Send the cancellation error to the error channel
		return
	case <-handle.waitCh:
		if handle.result != nil {
			log.Info().Str("executionID", handle.executionID).Msg("received results from execution")
			out <- handle.result
		} else {
			// NB(forrest): this shouldn't happen with the wasm and docker executors, but handling it as it
			// represents a significant error in executor logic, which may occur in future pluggable executor impls.
			errCh <- fmt.Errorf("execution (%s) result is nil", handle.executionID)
		}
	}
}

func (e *Executor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	//TODO implement me
	panic("implement me")
}
