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
	"go.uber.org/atomic"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/system"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	wasmlogs "github.com/bacalhau-project/bacalhau/pkg/logger/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/filefs"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/bacalhau-project/bacalhau/pkg/util/mountfs"
	"github.com/bacalhau-project/bacalhau/pkg/util/touchfs"
)

type Executor struct {
	// handlers is a map of executionID to its handler.
	handlers generic.SyncMap[string, *executionHandler]
}

func NewExecutor() (*Executor, error) {
	return &Executor{}, nil
}

func (e *Executor) IsInstalled(context.Context) (bool, error) {
	// WASM executor runs natively in Go and so is always available
	return true, nil
}

func (*Executor) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	return bidstrategy.NewBidResponse(true, "not place additional requirements on WASM jobs"), nil
}

func (*Executor) ShouldBidBasedOnUsage(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	usage models.Resources,
) (bidstrategy.BidStrategyResponse, error) {
	return bidstrategy.NewBidResponse(true, "not place additional requirements on WASM jobs"), nil
}

// Wazero: is compliant to WebAssembly Core Specification 1.0 and 2.0.
//
// WebAssembly1:  linear memory objects have sizes measured in pages. Each page is 65536 (2^16) bytes.
// In WebAssembly version 1, a linear memory can have at most 65536 pages, for a total of 2^32 bytes (4 gibibytes).

const WasmArch = 32
const WasmPageSize = 65536
const WasmMaxPagesLimit = 1 << (WasmArch / 2)

// Start initiates an execution based on the provided RunCommandRequest.
func (e *Executor) Start(ctx context.Context, request *executor.RunCommandRequest) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.Executor.Start")
	defer span.End()

	if handler, found := e.handlers.Get(request.ExecutionID); found {
		if handler.active() {
			return fmt.Errorf("starting execution (%s): %w", request.ExecutionID, executor.ErrAlreadyStarted)
		} else {
			return fmt.Errorf("starting execution (%s): %w", request.ExecutionID, executor.ErrAlreadyComplete)
		}
	}

	// Apply memory limits to the runtime. We have to do this in multiples of
	// the WASM page size of 64kb, so round up to the nearest page size if the
	// limit is not specified as a multiple of that.
	engineConfig := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)
	if request.Resources.Memory > 0 {
		requestedPages := request.Resources.Memory/WasmPageSize + math.Min(request.Resources.Memory%WasmPageSize, 1)
		if requestedPages > WasmMaxPagesLimit {
			err := fmt.Errorf("requested memory exceeds the wasm limit - %d > 4GB", request.Resources.Memory)
			log.Err(err).Msgf("requested memory exceeds maximum limit: %d > %d", requestedPages, WasmMaxPagesLimit)
			return err
		}
		engineConfig = engineConfig.WithMemoryLimitPages(uint32(requestedPages))
	}

	engineParams, err := wasmmodels.DecodeArguments(request.EngineParams)
	if err != nil {
		return fmt.Errorf("decoding wasm arguments: %w", err)
	}

	rootFs, err := e.makeFsFromStorage(ctx, request.ResultsDir, request.Inputs, request.Outputs)
	if err != nil {
		return err
	}

	// Create a new log manager and obtain some writers that we can pass to the wasm
	// configuration
	wasmLogs, err := wasmlogs.NewLogManager(ctx, request.ExecutionID)
	if err != nil {
		return err
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
			Str("entrypoint", engineParams.EntryPoint).
			Logger(),
		logManager: wasmLogs,
		activeCh:   make(chan bool),
		waitCh:     make(chan bool),
		running:    atomic.NewBool(false),
	}

	// register the handler for this executionID
	e.handlers.Put(request.ExecutionID, handler)
	go handler.run(ctx)
	return nil
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
		errCh <- fmt.Errorf("waiting on execution (%s): %w", executionID, executor.ErrNotFound)
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
func (e *Executor) Cancel(ctx context.Context, executionID string) error {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return fmt.Errorf("canceling execution (%s): %w", executionID, executor.ErrNotFound)
	}
	return handler.kill(ctx)
}

// GetOutputStream provides a stream of output logs for a specific execution.
// Parameters 'withHistory' and 'follow' control whether to include past logs
// and whether to keep the stream open for new logs, respectively.
// It returns an error if the execution is not found.
func (e *Executor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return nil, fmt.Errorf("getting outputs for execution (%s): %w", executionID, executor.ErrNotFound)
	}
	return handler.outputStream(ctx, withHistory, follow)
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

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
