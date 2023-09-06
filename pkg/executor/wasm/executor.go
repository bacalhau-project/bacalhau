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
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
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
	return semantic.NewChainedSemanticBidStrategy().ShouldBid(ctx, request)
}

func (*Executor) ShouldBidBasedOnUsage(
	ctx context.Context,
	request bidstrategy.BidStrategyRequest,
	usage models.Resources,
) (bidstrategy.BidStrategyResponse, error) {
	return resource.NewChainedResourceBidStrategy().ShouldBidBasedOnUsage(ctx, request, usage)
}

// Start initiates the execution of a WebAssembly (WASM) task using the provided RunCommandRequest.
// It first checks if the execution with the given ExecutionID is already active or completed,
// returning an error if so.
//
// This method performs several steps to initiate the execution:
// 1. Decodes the engine parameters specific to the WASM engine.
// 2. Configures memory limits for the WASM runtime, based on the resources specified in the request.
// 3. Creates a virtual filesystem from storage to be used by the WASM execution.
// 4. Sets up a new log manager specific to the execution for capturing logs.
//
// After preparing the environment, it creates an executionHandler and associates it with the ExecutionID.
// The executionHandler is responsible for running the WASM task, and it's executed in a separate goroutine.
//
// Parameters:
// - ctx: The context for the operation, used for timeouts and cancellations.
// - request: The RunCommandRequest object containing details such as ExecutionID, resources, and inputs/outputs.
//
// Returns:
// - An error if the initialization fails, or if an execution with the given ExecutionID is already active or completed.
func (e *Executor) Start(ctx context.Context, request *executor.RunCommandRequest) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.Executor.Start")
	defer span.End()

	if handler, found := e.handlers.Get(request.ExecutionID); found {
		if handler.active() {
			return fmt.Errorf("starting execution (%s): %w", request.ExecutionID, executor.AlreadyStartedErr)
		} else {
			return fmt.Errorf("starting execution (%s): %w", request.ExecutionID, executor.AlreadyCompleteErr)
		}
	}

	engineParams, err := wasmmodels.DecodeArguments(request.EngineParams)
	if err != nil {
		return fmt.Errorf("decoding wasm arguments: %w", err)
	}

	// Apply memory limits to the runtime. We have to do this in multiples of
	// the WASM page size of 64kb, so round up to the nearest page size if the
	// limit is not specified as a multiple of that.
	engineConfig := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)
	if request.Resources.Memory > 0 {
		const pageSize = 65536
		pageLimit := request.Resources.Memory/pageSize + math.Min(request.Resources.Memory%pageSize, 1)
		engineConfig = engineConfig.WithMemoryLimitPages(uint32(pageLimit))
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

// Wait waits for the completion of a specific execution identified by its executionID.
// It returns a read-only channel through which the result of the execution will be sent.
// If an execution with the given executionID is not found, an error is returned.
//
// Internally, this method spawns a new Goroutine to wait for the execution to complete,
// which adds a TODO note suggesting future optimization to handle a large number of concurrent
// Wait calls more efficiently.
//
// Parameters:
// - ctx: The context for the operation, used for timeouts and cancellations.
// - executionID: The unique identifier for the execution to wait on.
//
// Returns:
// - A read-only channel emitting the result of the execution.
// - An error if the execution with the given executionID is not found.
func (e *Executor) Wait(ctx context.Context, executionID string) (<-chan *models.RunCommandResult, error) {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return nil, fmt.Errorf("waiting on execution (%s): %w", executionID, executor.NotFoundErr)
	}
	ch := make(chan *models.RunCommandResult)
	go e.doWait(ctx, ch, handler)
	return ch, nil
}

// doWait is an internal method that performs the actual waiting for a given execution to complete.
// It listens for the completion signal from the executionHandler's wait channel or a cancellation signal
// from the context, and then forwards the result through the provided output channel.
//
// Parameters:
// - ctx: The context for the operation, used for timeouts and cancellations.
// - out: The channel through which the execution result will be sent.
// - handle: The executionHandler responsible for the execution.
//
// Note: This method is intended to be run in a separate Goroutine.
func (e *Executor) doWait(ctx context.Context, out chan *models.RunCommandResult, handle *executionHandler) {
	defer close(out)
	select {
	case <-ctx.Done():
		return
	case <-handle.waitCh:
		out <- handle.result
	}
}

// Cancel attempts to terminate an ongoing execution identified by its executionID.
// It looks up the handler for the execution and invokes its kill method to cancel it.
// If the execution with the given ID is not found, an error is returned.
//
// Parameters:
// - ctx: The context for the operation, used for timeouts and cancellations.
// - executionID: The unique identifier for the execution to cancel.
//
// Returns:
// - An error if the execution with the given executionID is not found or if canceling fails.
func (e *Executor) Cancel(ctx context.Context, executionID string) error {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return fmt.Errorf("canceling execution (%s): %w", executionID, executor.NotFoundErr)
	}
	return handler.kill(ctx)
}

// GetOutputStream retrieves the output stream for a specific execution identified by its executionID.
// The method allows configuring whether to include past output (history) and whether to keep the stream open for new output (follow).
// If an execution with the given executionID is not found, an error is returned.
//
// Parameters:
// - ctx: The context for the operation, used for timeouts and cancellations.
// - executionID: The unique identifier for the execution whose output stream to get.
// - withHistory: Whether to include the historical output in the returned stream.
// - follow: Whether to keep the stream open for new output.
//
// Returns:
// - An io.ReadCloser that provides access to the output stream.
// - An error if the execution with the given executionID is not found.
func (e *Executor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	handler, found := e.handlers.Get(executionID)
	if !found {
		return nil, fmt.Errorf("getting outputs for execution (%s): %w", executionID, executor.NotFoundErr)
	}
	return handler.outputStream(ctx, withHistory, follow)
}

// Run initiates and waits for the completion of an execution in one call.
// This is essentially a convenience method that combines the Start and Wait methods.
//
// Parameters:
// - ctx: The context for the operation, used for timeouts and cancellations.
// - request: The RunCommandRequest object containing details about the execution.
//
// Returns:
// - A pointer to a RunCommandResult object, containing the result of the execution.
// - An error if either starting or waiting for the execution fails, or if the context is cancelled.
//
// Steps:
//  1. Starts the execution using the Start method. If it fails, returns an error.
//  2. Waits for the execution to complete using the Wait method. If it fails, returns an error.
//  3. Listens on the channel returned by Wait, and returns the result when available.
//     If the context is cancelled, it returns a context error.
func (e *Executor) Run(
	ctx context.Context,
	request *executor.RunCommandRequest,
) (*models.RunCommandResult, error) {
	if err := e.Start(ctx, request); err != nil {
		return nil, err
	}
	res, err := e.Wait(ctx, request.ExecutionID)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-res:
		return out, nil
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
