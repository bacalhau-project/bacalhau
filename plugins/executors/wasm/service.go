package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"bacalhau-exec-wasm/executor"

	wasmlogs "github.com/bacalhau-project/bacalhau/pkg/logger/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/filefs"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/bacalhau-project/bacalhau/pkg/util/mountfs"
	"github.com/bacalhau-project/bacalhau/pkg/util/touchfs"
	"github.com/rs/zerolog/log"
	"github.com/tetratelabs/wazero"

	"go.uber.org/atomic"
	"golang.org/x/exp/constraints"
)

var ErrAlreadyStarted = fmt.Errorf("execution already started")
var ErrNotFound = fmt.Errorf("execution not found")
var ErrAlreadyComplete = fmt.Errorf("execution already completed")

type WasmExecutor struct {
	handlers generic.SyncMap[string, *executionHandler]

	executor.UnimplementedExecutorServer
}

func NewWasmExecutor() (*WasmExecutor, error) {
	return &WasmExecutor{}, nil
}

func (e *WasmExecutor) IsInstalled(ctx context.Context, request *executor.IsInstalledRequest) (*executor.IsInstalledResponse, error) {
	// WASM executor runs natively in Go and so is always available
	return &executor.IsInstalledResponse{Installed: true}, nil
}

func (*WasmExecutor) ShouldBid(ctx context.Context, request *executor.ShouldBidRequest) (*executor.ShouldBidResponse, error) {
	return &executor.ShouldBidResponse{ShouldBid: true}, nil
}

func (*WasmExecutor) ShouldBidBasedOnUsage(
	ctx context.Context,
	request *executor.ShouldBidBasedOnUsageRequest,
) (*executor.ShouldBidResponse, error) {
	return &executor.ShouldBidResponse{ShouldBid: true}, nil
}

// Wazero: is compliant to WebAssembly Core Specification 1.0 and 2.0.
//
// WebAssembly1:  linear memory objects have sizes measured in pages. Each page is 65536 (2^16) bytes.
// In WebAssembly version 1, a linear memory can have at most 65536 pages, for a total of 2^32 bytes (4 gibibytes).

const WASM_ARCH = 32
const WASM_PAGE_SIZE = 65536
const WASM_MAX_PAGES_LIMIT = 1 << (WASM_ARCH / 2)

// Start initiates an execution based on the provided RunCommandRequest.
func (e *WasmExecutor) Start(ctx context.Context, request *executor.RunCommandRequest) (*executor.StartResponse, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.Executor.Start")
	defer span.End()

	if handler, found := e.handlers.Get(request.ExecutionID); found {
		if handler.active() {
			return nil, fmt.Errorf("starting execution (%s): %w", request.ExecutionID, ErrAlreadyStarted)
		} else {
			return nil, fmt.Errorf("starting execution (%s): %w", request.ExecutionID, ErrAlreadyComplete)
		}
	}

	// Apply memory limits to the runtime. We have to do this in multiples of
	// the WASM page size of 64kb, so round up to the nearest page size if the
	// limit is not specified as a multiple of that.
	engineConfig := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)
	if request.Resources.Memory > 0 {
		requestedPages := request.Resources.Memory/WASM_PAGE_SIZE + intMin(request.Resources.Memory%WASM_PAGE_SIZE, 1)
		if requestedPages > WASM_MAX_PAGES_LIMIT {
			err := fmt.Errorf("requested memory exceeds the wasm limit - %d > 4GB", request.Resources.Memory)
			log.Err(err).Msgf("requested memory exceeds maximum limit: %d > %d", requestedPages, WASM_MAX_PAGES_LIMIT)
			return nil, err
		}
		engineConfig = engineConfig.WithMemoryLimitPages(uint32(requestedPages))
	}

	engineParams, err := DecodeArguments(request.EngineParams)
	if err != nil {
		return nil, fmt.Errorf("decoding wasm arguments: %w", err)
	}

	rootFs, err := e.makeFsFromStorage(ctx, request.ResultsDir, request.Inputs, request.Outputs)
	if err != nil {
		return nil, err
	}

	// Create a new log manager and obtain some writers that we can pass to the wasm
	// configuration
	wasmLogs, err := wasmlogs.NewLogManager(ctx, request.ExecutionID)
	if err != nil {
		return nil, err
	}

	handler := &executionHandler{
		runtime:     wazero.NewRuntimeWithConfig(ctx, engineConfig),
		arguments:   engineParams,
		fs:          rootFs,
		inputs:      request.Inputs,
		executionID: request.ExecutionID,
		resultsDir:  request.ResultsDir,
		limits:      request.OutputLimits,
		logger: log.With().
			Str("execution", request.ExecutionID).
			Str("job", request.JobID).
			Str("entrypoint", *engineParams.EntryPoint).
			Logger(),
		logManager: wasmLogs,
		activeCh:   make(chan bool),
		waitCh:     make(chan bool),
		running:    atomic.NewBool(false),
	}

	// register the handler for this executionID
	e.handlers.Put(request.ExecutionID, handler)
	go handler.run(ctx)

	resp := &executor.StartResponse{
		//TODO(ross)
	}

	return resp, nil
}

func (e *WasmExecutor) Wait(request *executor.WaitRequest, stream executor.Executor_WaitServer) error {
	ctx := context.TODO() // No context in the GRPC call

	resCh, errCh := e.wait(ctx, request.ExecutionID)
	select {
	case out := <-resCh:
		stream.Send(getRunCommandResponse(out))
	case err := <-errCh:
		stream.Send(&executor.RunCommandResponse{ErrorMsg: err.Error()})
		return err
	}

	return nil
}

// Wait initiates a wait for the completion of a specific execution using its
// executionID.
func (e *WasmExecutor) wait(ctx context.Context, executionID string) (<-chan *models.RunCommandResult, <-chan error) {
	handler, found := e.handlers.Get(executionID)
	outCh := make(chan *models.RunCommandResult, 1)
	errCh := make(chan error, 1)

	if !found {
		errCh <- fmt.Errorf("waiting on execution (%s): %w", executionID, ErrNotFound)
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
func (e *WasmExecutor) doWait(ctx context.Context, out chan *models.RunCommandResult, errCh chan error, handle *executionHandler) {
	log.Info().Str("executionID", handle.executionID).Msg("waiting on execution")

	defer close(out)
	defer close(errCh)

	select {
	case <-ctx.Done():
		errCh <- ctx.Err() // Send the cancellation error to the error channel
		return
	case <-handle.waitCh:
		log.Info().Str("executionID", handle.executionID).Msg("received results from execution")
		if handle.result != nil {
			out <- handle.result
		} else {
			errCh <- fmt.Errorf("execution result is nil")
		}
	}
}

// Cancel tries to cancel a specific execution by its executionID.
// It returns an error if the execution is not found.
func (e *WasmExecutor) Cancel(ctx context.Context, request *executor.CancelCommandRequest) (*executor.CancelCommandResponse, error) {
	handler, found := e.handlers.Get(request.ExecutionID)
	if !found {
		return nil, fmt.Errorf("canceling execution (%s): %w", request.ExecutionID, ErrNotFound)
	}

	return &executor.CancelCommandResponse{}, handler.kill(ctx)
}

// GetOutputStream provides a stream of output logs for a specific execution.
// Parameters 'withHistory' and 'follow' control whether to include past logs
// and whether to keep the stream open for new logs, respectively.
// It returns an error if the execution is not found.
func (e *WasmExecutor) GetOutputStream(request *executor.OutputStreamRequest, stream executor.Executor_GetOutputStreamServer) error {
	handler, found := e.handlers.Get(request.ExecutionID)
	if !found {
		return fmt.Errorf("getting outputs for execution (%s): %w", request.ExecutionID, ErrNotFound)
	}

	reader, err := handler.outputStream(context.TODO(), request.History, request.Follow)
	if err != nil {
		return err
	}

	var buffer []byte = make([]byte, 4096)

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
			var nRead int
			if nRead, err = reader.Read(buffer); err != nil {
				return err
			}

			if nRead > 0 {
				resp := &executor.OutputStreamResponse{
					Data: append([]byte(nil), buffer[0:nRead]...),
				}

				if err = stream.Send(resp); err != nil {
					return err
				}
			}
		}
	}

}

// Run initiates and waits for the completion of an execution in one call.
// This method serves as a higher-level convenience function that
// internally calls Start and Wait methods.
// It returns the result of the execution or an error if either starting
// or waiting fails, or if the context is canceled.
func (e *WasmExecutor) Run(
	ctx context.Context,
	request *executor.RunCommandRequest,
) (*executor.RunCommandResponse, error) {
	if _, err := e.Start(ctx, request); err != nil {
		return nil, err
	}

	resCh, errCh := e.wait(ctx, request.ExecutionID)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-resCh:
		return getRunCommandResponse(out), nil
	case err := <-errCh:
		return nil, err
	}
}

// TODO: Would be nice to use structural typing to cast from one type to the other,
// maybe the int32 cast stopping it working atm.
func getRunCommandResponse(result *models.RunCommandResult) *executor.RunCommandResponse {
	return &executor.RunCommandResponse{
		STDOUT:          result.STDOUT,
		STDERR:          result.STDERR,
		StdoutTruncated: result.StdoutTruncated,
		StderrTruncated: result.StderrTruncated,
		ExitCode:        int32(result.ExitCode),
		ErrorMsg:        result.ErrorMsg,
	}
}

// makeFsFromStorage sets up a virtual filesystem (represented by an fs.FS) that
// will be the filesystem exposed to our WASM. The strategy for this is to:
//
//   - mount each input at the name specified by Path
//   - make a directory in the job results directory for each output and mount that
//     at the name specified by Name
func (e *WasmExecutor) makeFsFromStorage(
	ctx context.Context,
	jobResultsDir string,
	volumes []*executor.PreparedStorage,
	outputs []*executor.ResultPath) (fs.FS, error) {
	var err error
	rootFs := mountfs.New()

	for _, v := range volumes {
		log.Ctx(ctx).Debug().
			Str("input", v.InputSource.Target).
			Str("source", v.Volume.Source).
			Msg("Using input")

		var stat os.FileInfo
		stat, err = os.Stat(v.Volume.Source)
		if err != nil {
			return nil, err
		}

		var inputFs fs.FS
		if stat.IsDir() {
			inputFs = os.DirFS(v.Volume.Source)
		} else {
			inputFs = filefs.New(v.Volume.Source)
		}

		err = rootFs.Mount(v.InputSource.Target, inputFs)
		if err != nil {
			return nil, err
		}
	}

	for _, output := range outputs {
		if output.Name == "" {
			return nil, fmt.Errorf("output volume has no name: %+v", output)
		}

		if output.Path == "" {
			return nil, fmt.Errorf("output volume has no path: %+v", output)
		}

		srcd := filepath.Join(jobResultsDir, output.Name)
		log.Ctx(ctx).Debug().
			Str("output", output.Name).
			Str("dir", srcd).
			Msg("Collecting output")

		err = os.Mkdir(srcd, util.OS_ALL_R|util.OS_ALL_X|util.OS_USER_W)
		if err != nil {
			return nil, err
		}

		err = rootFs.Mount(output.Name, touchfs.New(srcd))
		if err != nil {
			return nil, err
		}
	}

	return rootFs, nil
}

func intMin[T constraints.Ordered](a T, b T) T {
	if a < b {
		return a
	}
	return b
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.ExecutorServer = (*WasmExecutor)(nil)
