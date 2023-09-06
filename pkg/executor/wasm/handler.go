package wasm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"

	"github.com/rs/zerolog"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/sys"
	"go.uber.org/atomic"
	"golang.org/x/exp/maps"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	wasmlogs "github.com/bacalhau-project/bacalhau/pkg/logger/wasm"
	models "github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

type executionHandler struct {
	// runtime configured with resource-limits
	runtime wazero.Runtime
	// arguments used to instantiate and run the wasm module
	arguments *wasmmodels.EngineArguments
	// virtual filesystem exposed to wasm module
	fs fs.FS
	// wasm modules imported by main wasm module
	inputs []storage.PreparedStorage

	executionID string
	resultsDir  string
	limits      executor.OutputLimits

	// cancellation
	cancel func()

	// bacalhau logging
	logger zerolog.Logger

	// wasm logging
	logManager *wasmlogs.LogManager

	// synchronization
	// blocks until the container starts
	activeCh chan bool
	// blocks until the run method returns
	waitCh chan bool
	// true until the run method returns
	running *atomic.Bool

	// results
	result *models.RunCommandResult
}

func (h *executionHandler) run(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error().
				Str("recover", fmt.Sprintf("%v", r)).
				Msg("execution recovered from panic")
			// TODO don't do this.
			h.result = &models.RunCommandResult{}
		}
	}()

	var wasmCtx context.Context
	wasmCtx, h.cancel = context.WithCancel(ctx)
	defer func() {
		h.running.Store(false)
		close(h.waitCh)
		h.cancel()
	}()

	tracingEngine := tracedRuntime{h.runtime}
	defer closer.ContextCloserWithLogOnError(ctx, "engine", tracingEngine)
	stdout, stderr := h.logManager.GetWriters()
	// Configure the modules. We don't want to execute any start functions
	// automatically as we will do it manually later. Finally, add the
	// filesystem which contains our input and output.
	args := append([]string{""}, h.arguments.Parameters...)
	config := wazero.NewModuleConfig().
		WithStartFunctions().
		WithStdout(stdout).
		WithStderr(stderr).
		WithArgs(args...).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithFS(h.fs)
	keys := maps.Keys(h.arguments.EnvironmentVariables)
	sort.Strings(keys)
	for _, key := range keys {
		// Make sure we add the environment variables in a consistent order
		config = config.WithEnv(key, h.arguments.EnvironmentVariables[key])
	}

	h.logger.Info().Msg("instantiating wasm modules")
	loader := NewModuleLoader(tracingEngine, config, h.inputs...)
	// TODO we have been ignoring errors from this method for ages. Now that we actually check them tests fail! nice..
	// v1.0.3: https://github.com/bacalhau-project/bacalhau/blob/v1.0.3/pkg/executor/wasm/executor.go#L243
	// current: https://github.com/bacalhau-project/bacalhau/blob/ff1bd9cb1c09fa3652c4a68943a97476340dbe33/pkg/executor/wasm/executor.go#L216
	for _, importModule := range h.arguments.ImportModules {
		_, err := loader.InstantiateRemoteModule(ctx, importModule)
		if err != nil {
			h.logger.Warn().
				Str("input_source", importModule.InputSource.Source.Type).
				Str("input_alias", importModule.InputSource.Alias).
				Str("input_target", importModule.InputSource.Target).
				Str("volume_type", importModule.Volume.Type.String()).
				Str("volume_source", importModule.Volume.Source).
				Str("volume_target", importModule.Volume.Target).
				Msg("failed to instantiate import module")
			// lets just ignore the error like we have always done!
			h.result = executor.NewFailedResult(
				fmt.Errorf("failed to instantiate import module (%s): %w",
					importModule.InputSource.Source.Type, err).Error())
			return
		}
	}

	// Load and instantiate the entry module.
	entryModule := h.arguments.EntryModule
	instance, err := loader.InstantiateRemoteModule(ctx, entryModule)
	if err != nil {
		h.logger.Warn().
			Str("input_source", entryModule.InputSource.Source.Type).
			Str("input_alias", entryModule.InputSource.Alias).
			Str("input_target", entryModule.InputSource.Target).
			Str("volume_type", entryModule.Volume.Type.String()).
			Str("volume_source", entryModule.Volume.Source).
			Str("volume_target", entryModule.Volume.Target).
			Msg("failed to instantiate entry module")
		h.result = executor.NewFailedResult(
			fmt.Errorf("failed to instantiate entry module module (%s): %w",
				entryModule.InputSource.Source.Type, err).Error())
		return
	}

	// invoke the function entry point
	entryFunc := instance.ExportedFunction(h.arguments.EntryPoint)
	h.logger.Info().Msg("running execution")

	// TODO this is a little racey
	h.running.Store(true)
	close(h.activeCh)

	// The function should exit which results in a sys.ExitError. So we capture
	// the exit code for inclusion in the job output, and ignore the return code
	// from the function (most WASI compilers will not give one). Some compilers
	// though do not set an exit code, so we use a default of -1.
	_, wasmErr := entryFunc.Call(wasmCtx)
	exitCode := int64(0)
	var errExit *sys.ExitError
	if errors.As(wasmErr, &errExit) {
		exitCode = int64(errExit.ExitCode())
		wasmErr = nil
		h.logger.Info().Int64("exit_code", exitCode).Msg("execution ended")
	}
	if wasmErr != nil {
		// in the event that an error is returned without an exist code we'll assume the operation
		// failed and set the exit code to 1
		exitCode = 1
		h.logger.Warn().Int64("exit_code", exitCode).Err(wasmErr).Msg("execution ended")
	}
	// execution has finished and there's nothing else to read from so inform
	// the logs that it is time to drain any remaining items.
	h.logManager.Drain()
	stdoutReader, stderrReader := h.logManager.GetDefaultReaders(false)

	h.result = executor.WriteJobResults(h.resultsDir, stdoutReader, stderrReader, int(exitCode), wasmErr, h.limits)
}

func (h *executionHandler) active() bool {
	return h.running.Load()
}

func (h *executionHandler) kill(ctx context.Context) error {
	h.cancel()
	return nil
}

func (h *executionHandler) outputStream(ctx context.Context, withHistory, follow bool) (io.ReadCloser, error) {
	return h.logManager.GetMuxedReader(follow), nil
}
