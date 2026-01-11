package wasm

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/tetratelabs/wazero"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/util/filefs"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/util/mountfs"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/util/touchfs"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

// Executor handles the execution of WebAssembly modules.
// It manages the lifecycle of WASM executions, including starting, waiting for completion,
// and handling cancellation.
type Executor struct {
	// handlers is a map of executionID to its handler.
	handlers generic.SyncMap[string, *executionHandler]
}

// NewExecutor creates a new WASM executor instance.
func NewExecutor() (*Executor, error) {
	return &Executor{}, nil
}

// IsInstalled checks if the WASM executor is available.
// Since WASM executor runs natively in Go, it's always available.
func (e *Executor) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

// ShouldBid determines if the executor should bid on a job.
// WASM jobs don't have additional requirements, so it always returns true.
func (*Executor) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	return bidstrategy.NewBidResponse(true, "not place additional requirements on WASM jobs"), nil
}

// ShouldBidBasedOnUsage determines if the executor should bid on a job based on resource usage.
// WASM jobs don't have additional requirements, so it always returns true.
func (*Executor) ShouldBidBasedOnUsage(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	usage models.Resources,
) (bidstrategy.BidStrategyResponse, error) {
	return bidstrategy.NewBidResponse(true, "not place additional requirements on WASM jobs"), nil
}

// Start initiates an execution based on the provided RunCommandRequest.
// It sets up the WASM runtime with appropriate memory limits and filesystem mounts,
// then starts the execution in a separate goroutine.
func (e *Executor) Start(ctx context.Context, request *executor.RunCommandRequest) error {
	if handler, found := e.handlers.Get(request.ExecutionID); found {
		if handler.active() {
			return executor.NewExecutorError(executor.ExecutionAlreadyStarted, fmt.Sprintf("starting execution (%s)", request.ExecutionID))
		} else {
			return executor.NewExecutorError(executor.ExecutionAlreadyComplete, fmt.Sprintf("starting execution (%s)", request.ExecutionID))
		}
	}

	// Configure runtime with memory limits
	engineConfig, err := e.configureRuntime(request.Resources.Memory)
	if err != nil {
		return err
	}

	rootFs, err := e.makeFsFromStorage(
		ctx,
		compute.ExecutionResultsDir(request.ExecutionDir),
		request.Inputs,
		request.Outputs,
	)
	if err != nil {
		return err
	}

	handler, err := newExecutionHandler(ctx,
		request,
		wazero.NewRuntimeWithConfig(ctx, engineConfig),
		rootFs)

	if err != nil {
		return err
	}

	// register the handler for this executionID
	e.handlers.Put(request.ExecutionID, handler)
	go handler.run(ctx)
	return nil
}

// configureRuntime sets up the WASM runtime with appropriate memory limits
func (e *Executor) configureRuntime(memoryLimit uint64) (wazero.RuntimeConfig, error) {
	engineConfig := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)

	// Apply memory limits to the runtime. We have to do this in multiples of
	// the WASM page size of 64kb, so round up to the nearest page size if the
	// limit is not specified as a multiple of that.
	if memoryLimit > 0 {
		requestedPages := memoryLimit/WasmPageSize + math.Min(memoryLimit%WasmPageSize, 1)
		if requestedPages > WasmMaxPagesLimit {
			maxBytes := uint64(WasmMaxPagesLimit) * WasmPageSize
			return nil, NewMemoryLimitError(memoryLimit, maxBytes)
		}
		engineConfig = engineConfig.WithMemoryLimitPages(uint32(requestedPages))
	}
	return engineConfig, nil
}

// Wait initiates a wait for the completion of a specific execution using its
// executionID. The function returns two channels: one for the result and another
// for any potential error. If the executionID is not found, an error is immediately
// sent to the error channel. Otherwise, an internal goroutine (doWait) is spawned
// to handle the asynchronous waiting. Callers should use the two returned channels
// to wait for the result of the execution or an error. This can be due to issues
// either beginning the wait or in getting the response. This approach allows the
// caller to synchronize Wait with calls to Start, waiting for the execution to complete.
func (e *Executor) Wait(ctx context.Context, executionID string) (<-chan *models.RunCommandResult, <-chan error) {
	handler, found := e.handlers.Get(executionID)
	outCh := make(chan *models.RunCommandResult, 1)
	errCh := make(chan error, 1)

	if !found {
		errCh <- executor.NewExecutorError(executor.ExecutionNotFound, fmt.Sprintf("waiting on execution (%s)", executionID))
		return outCh, errCh
	}

	go e.doWait(ctx, outCh, errCh, handler)
	return outCh, errCh
}

// doWait is a helper function that actively waits for an execution to finish. It
// listens on the executionHandler's wait channel for completion signals. Once the
// signal is received, the result is sent to the provided output channel. If there's
// a cancellation request (context is done) before completion, an error is relayed to
// the error channel. If the execution result is nil, an error suggests a potential
// flaw in the executor logic.
func (e *Executor) doWait(ctx context.Context, out chan *models.RunCommandResult, errCh chan error, handle *executionHandler) {
	log.Info().Str("executionID", handle.request.ExecutionID).Msg("waiting on execution")

	defer close(out)
	defer close(errCh)

	select {
	case <-ctx.Done():
		errCh <- ctx.Err() // Send the cancellation error to the error channel
		return
	case <-handle.waitCh:
		log.Info().Str("executionID", handle.request.ExecutionID).Msg("received results from execution")
		if handle.result != nil {
			out <- handle.result
		} else {
			errCh <- fmt.Errorf("execution result is nil")
		}
	}
}

// Cancel tries to cancel a specific execution by its executionID.
// It returns an error if the execution is not found.
func (e *Executor) Cancel(ctx context.Context, executionID string) error {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return executor.NewExecutorError(executor.ExecutionNotFound, fmt.Sprintf("canceling execution (%s)", executionID))
	}
	return handler.kill(ctx)
}

// GetLogStream provides a stream of output logs for a specific execution.
// Parameters 'withHistory' and 'follow' control whether to include past logs
// and whether to keep the stream open for new logs, respectively.
// It returns an error if the execution is not found.
func (e *Executor) GetLogStream(ctx context.Context, request messages.ExecutionLogsRequest) (io.ReadCloser, error) {
	handler, found := e.handlers.Get(request.ExecutionID)
	if !found {
		return nil, executor.NewExecutorError(executor.ExecutionNotFound,
			fmt.Sprintf("getting outputs for execution (%s)", request.ExecutionID))
	}
	return handler.outputStream(ctx, request)
}

// Run initiates and waits for the completion of an execution in one call.
// This method serves as a higher-level convenience function that
// internally calls Start and Wait methods.
// It returns the result of the execution or an error if either starting
// or waiting fails, or if the context is canceled.
func (e *Executor) Run(
	ctx context.Context,
	request *executor.RunCommandRequest,
) (*models.RunCommandResult, error) {
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

// makeFsFromStorage sets up a virtual filesystem (represented by an fs.FS) that
// will be the filesystem exposed to our WASM. The strategy for this is to:
//
//   - mount each input at the name specified by Path
//   - make a directory in the job results directory for each output and mount that
//     at the name specified by Name
func (e *Executor) makeFsFromStorage(
	ctx context.Context,
	jobResultsDir string,
	volumes []storage.PreparedStorage,
	outputs []*models.ResultPath) (fs.FS, error) {
	var err error
	rootFs := mountfs.New()

	// Validate input sources
	for _, v := range volumes {
		if v.Volume.Target == "" {
			return nil, NewInputConfigError("input source has no target path")
		}
		if v.Volume.Source == "" {
			return nil, NewInputConfigError("input source has no source path")
		}

		log.Ctx(ctx).Debug().
			Str("target", v.Volume.Target).
			Str("source", v.Volume.Source).
			Msg("Using input")

		var stat os.FileInfo
		stat, err = os.Stat(v.Volume.Source)
		if err != nil {
			return nil, NewInputConfigError(fmt.Sprintf("input source %q does not exist: %s", v.Volume.Source, err))
		}

		var inputFs fs.FS
		if stat.IsDir() {
			inputFs = os.DirFS(v.Volume.Source)
		} else {
			inputFs = filefs.New(v.Volume.Source)
		}

		err = rootFs.Mount(v.Volume.Target, inputFs)
		if err != nil {
			return nil, NewInputConfigError(fmt.Sprintf("failed to mount input %q: %s", v.Volume.Target, err))
		}
	}

	// Validate output configuration
	for _, output := range outputs {
		if output.Name == "" {
			return nil, NewOutputError("output volume has no name")
		}

		if output.Path == "" {
			return nil, NewOutputError("output volume has no path")
		}

		// Check for conflicts with input paths
		for _, v := range volumes {
			if v.Volume.Target == output.Name {
				return nil, NewOutputError(fmt.Sprintf("output name %q conflicts with input target", output.Name))
			}
		}

		srcDir := filepath.Join(jobResultsDir, output.Name)
		log.Ctx(ctx).Debug().
			Str("output", output.Name).
			Str("dir", srcDir).
			Msg("Collecting output")

		err = os.Mkdir(srcDir, util.OS_ALL_R|util.OS_ALL_X|util.OS_USER_W)
		if err != nil {
			return nil, NewFilesystemError(output.Name, err)
		}

		err = rootFs.Mount(output.Name, touchfs.New(srcDir))
		if err != nil {
			return nil, NewFilesystemError(output.Name, err)
		}
	}

	return rootFs, nil
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
