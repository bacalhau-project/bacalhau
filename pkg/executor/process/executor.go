package process

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
)

type RunningProcess struct {
	cmd    *exec.Cmd
	stdout bytes.Buffer
	stderr bytes.Buffer
}

type Executor struct {
	commands generic.SyncMap[string, *RunningProcess]
}

// Cancel implements executor.Executor.
func (e *Executor) Cancel(ctx context.Context, executionID string) error {
	rp, found := e.commands.Get(executionID)
	if !found {
		return fmt.Errorf("failed to find running process for execution %s", executionID)
	}

	// The command has already gone
	if rp.cmd == nil {
		return nil
	}

	// Kill the process!
	// TODO: This does not kill any child processes and so we may want a
	// staged approach where we SIGINT, SIGTERM, SIGKILL instead. This let's
	// the process handle the first two options, but with the SIGKILL being
	// uncatchable.
	_ = rp.cmd.Process.Kill()

	return nil
}

// GetOutputStream implements executor.Executor.
func (*Executor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	panic("unimplemented")
}

// IsInstalled implements executor.Executor.
func (*Executor) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

// Run implements executor.Executor.
func (e *Executor) Run(ctx context.Context, args *executor.RunCommandRequest) (*models.RunCommandResult, error) {
	if err := e.Start(ctx, args); err != nil {
		return nil, err
	}

	resCh, errCh := e.Wait(ctx, args.ExecutionID)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-resCh:
		return out, nil
	case err := <-errCh:
		return nil, err
	}
}

// Start implements executor.Executor.
func (e *Executor) Start(ctx context.Context, request *executor.RunCommandRequest) error {
	args, err := EngineSpecFromDict(request.EngineParams.Params)
	if err != nil {
		return err
	}

	rp := &RunningProcess{}

	proc := exec.Command(args.Name, args.Arguments...)
	proc.Stdout = bufio.NewWriter(&rp.stdout)
	proc.Stderr = bufio.NewWriter(&rp.stderr)
	rp.cmd = proc

	err = proc.Start()
	if err != nil {
		return err
	}

	e.commands.Put(request.ExecutionID, rp)

	return nil
}

// Wait will wait until the Run method completes
func (e *Executor) Wait(ctx context.Context, executionID string) (<-chan *models.RunCommandResult, <-chan error) {
	outputChannel := make(chan *models.RunCommandResult, 1)
	errorChannel := make(chan error, 1)

	rp, found := e.commands.Get(executionID)
	if !found {
		errorChannel <- fmt.Errorf("failed to find running process for execution %s", executionID)
	}

	go func(rp *RunningProcess) {
		err := rp.cmd.Wait()

		if err != nil {
			errorChannel <- err
		} else {
			outputChannel <- &models.RunCommandResult{
				ExitCode: rp.cmd.ProcessState.ExitCode(),
				STDOUT:   rp.stdout.String(),
				STDERR:   rp.stderr.String(),
			}
		}
	}(rp)

	return outputChannel, errorChannel
}

func (*Executor) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	return semantic.NewChainedSemanticBidStrategy().ShouldBid(ctx, request)
}

func (*Executor) ShouldBidBasedOnUsage(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	usage models.Resources,
) (bidstrategy.BidStrategyResponse, error) {
	return resource.NewChainedResourceBidStrategy().ShouldBidBasedOnUsage(ctx, request, usage)
}

func NewExecutor() (*Executor, error) {
	return &Executor{}, nil
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
