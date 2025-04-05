package wasm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"time"

	"github.com/dylibso/observe-sdk/go/adapter/opentelemetry"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/sys"
	"go.uber.org/atomic"
	"golang.org/x/exp/maps"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	wasmlogs "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/util/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

// executionHandler manages the lifecycle of a single WASM execution.
// It handles loading modules, setting up the runtime environment,
// executing the WASM code, and collecting results.
type executionHandler struct {
	// runtime configured with resource-limits
	runtime wazero.Runtime
	// spec contains the WASM engine specification
	spec wasmmodels.EngineSpec
	// virtual filesystem exposed to wasm module
	fs fs.FS

	// request contains all the information needed for execution
	request *executor.RunCommandRequest

	// cancellation
	cancel func()

	// logging
	logger     zerolog.Logger       // bacalhau logging
	logManager *wasmlogs.LogManager // wasm logging

	// synchronization channels
	activeCh chan bool    // blocks until the container starts
	waitCh   chan bool    // blocks until the run method returns
	running  *atomic.Bool // true until the run method returns

	// results
	result *models.RunCommandResult
}

// newExecutionHandler creates a new execution handler for the given request
func newExecutionHandler(
	ctx context.Context,
	request *executor.RunCommandRequest,
	runtime wazero.Runtime,
	fs fs.FS,
) (*executionHandler, error) {
	// Decode WASM engine spec
	wasmSpec, err := wasmmodels.DecodeSpec(request.EngineParams)
	if err != nil {
		return nil, NewSpecError(err)
	}

	// Create a new log manager and obtain writers for WASM configuration
	wasmLogs, err := wasmlogs.NewLogManager(ctx, request.ExecutionID)
	if err != nil {
		return nil, NewLogError(err)
	}

	return &executionHandler{
		runtime: runtime,
		spec:    wasmSpec,
		fs:      fs,

		request: request,

		logger: log.With().
			Str("execution", request.ExecutionID).
			Str("job", request.JobID).
			Str("entrypoint", wasmSpec.Entrypoint).
			Logger(),
		logManager: wasmLogs,

		activeCh: make(chan bool),
		waitCh:   make(chan bool),
		running:  atomic.NewBool(false),
	}, nil
}

// run executes the WASM module and handles its lifecycle.
// It sets up the runtime environment, loads dependencies,
// executes the main function, and collects results.
func (h *executionHandler) run(ctx context.Context) {
	ActiveExecutions.Inc(ctx)
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error().
				Str("recover", fmt.Sprintf("%v", r)).
				Msg("execution recovered from panic")

			// The recover was originally here for a bug we think is now fixed, but given
			// the propensity for panics in this area, we're being extra-cautious and
			// ensuring we can handle any future panics that arise.
			h.result = executor.NewFailedResult(
				fmt.Sprintf("WASM executor failed with an internal error: %v", r),
			)
		}

		ActiveExecutions.Dec(ctx)
	}()

	// Set up execution context with cancellation
	var wasmCtx context.Context
	wasmCtx, h.cancel = context.WithCancel(ctx)
	defer func() {
		h.running.Store(false)
		close(h.waitCh)
		h.cancel()
	}()

	// Set up tracing if available
	tracingEngine := h.setupTracing(ctx)
	defer closer.ContextCloserWithLogOnError(ctx, "engine", tracingEngine)

	// Set up logging and module configuration
	stdout, stderr := h.logManager.GetWriters()
	config := h.createModuleConfig(stdout, stderr)

	// Load and instantiate modules
	instance, err := h.loadModules(ctx, tracingEngine, config)
	if err != nil {
		return
	}

	// Execute the main function
	h.executeMainFunction(wasmCtx, instance)
}

// setupTracing initializes OpenTelemetry tracing for the WASM execution
func (h *executionHandler) setupTracing(ctx context.Context) tracedRuntime {
	adapter := opentelemetry.NewOTelAdapter(&opentelemetry.OTelConfig{
		ServiceName:        "bacalhau",
		EmitTracesInterval: time.Second * 1,
		TraceBatchMax:      10,
		// the remaining fields are completed from a system-configured client
		// by using the `UseCustomClient` method on the adapter below
	})

	traceClient, err := telemetry.GetTraceClient()
	if err != nil {
		h.logger.Err(err).Msg("Failed to set up tracing")
		return tracedRuntime{Runtime: h.runtime}
	}
	if traceClient != nil {
		adapter.UseCustomClient(traceClient)
	}

	adapter.Start(ctx)
	return tracedRuntime{Runtime: h.runtime, adapter: adapter}
}

// createModuleConfig creates the WASM module configuration with logging and environment setup
func (h *executionHandler) createModuleConfig(stdout, stderr io.Writer) wazero.ModuleConfig {
	args := append([]string{""}, h.spec.Parameters...)
	config := wazero.NewModuleConfig().
		WithStartFunctions().
		WithStdout(stdout).
		WithStderr(stderr).
		WithArgs(args...).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithFS(h.fs)

	// Add environment variables in a consistent order
	keys := maps.Keys(h.request.Env)
	sort.Strings(keys)
	for _, key := range keys {
		config = config.WithEnv(key, h.request.Env[key])
	}

	return config
}

// loadModules loads and instantiates all required WASM modules
func (h *executionHandler) loadModules(ctx context.Context, engine tracedRuntime, config wazero.ModuleConfig) (api.Module, error) {
	h.logger.Info().Msg("instantiating wasm modules")
	loader := NewModuleLoader(engine, config, h.fs)

	// in wasm, if network type is undefined, we default to host
	if h.request.Network == nil || h.request.Network.Type == models.NetworkDefault {
		h.request.Network = &models.NetworkConfig{Type: models.NetworkHost}
	}

	// Load HTTP module if networking is enabled
	if h.request.Network != nil && h.request.Network.Type != models.NetworkNone {
		// Configure HTTP module parameters
		httpParams := http.Params{
			Network: h.request.Network,
		}

		// Instantiate HTTP module
		if err := http.InstantiateModule(ctx, engine.Runtime, httpParams); err != nil {
			h.result = executor.NewFailedResult(fmt.Sprintf("failed to load HTTP module: %s", err))
			return nil, err
		}
	}

	// Load import modules first
	for _, importModule := range h.spec.ImportModules {
		if _, err := loader.InstantiateModule(ctx, importModule); err != nil {
			h.result = executor.NewFailedResult(fmt.Sprintf("failed to load import module %s: %s", importModule, err))
			return nil, err
		}
	}

	// Load and instantiate the entry module
	instance, err := loader.InstantiateModule(ctx, h.spec.EntryModule)
	if err != nil {
		h.result = executor.NewFailedResult(fmt.Sprintf("failed to load entry module %s: %s", h.spec.EntryModule, err))
		return nil, err
	}

	// Verify entry point exists
	if err = h.verifyEntryPoint(instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// verifyEntryPoint checks if the specified entry point exists in the module
func (h *executionHandler) verifyEntryPoint(instance api.Module) error {
	definitions := instance.ExportedFunctionDefinitions()
	_, found := definitions[h.spec.Entrypoint]

	if !found {
		h.result = executor.NewFailedResult(
			fmt.Sprintf("unable to find the entrypoint '%s' in the WASM module", h.spec.Entrypoint),
		)
		return NewEntrypointError(h.spec.Entrypoint)
	}
	return nil
}

// executeMainFunction runs the main WASM function and handles its completion
func (h *executionHandler) executeMainFunction(ctx context.Context, instance api.Module) {
	// Calling instance.ExportedFunction with an invalid name returns an item that
	// is not null. Or rather, the returned item is not null, but something internal
	// when calling Call() _is_ null, causing a panic.
	//
	// To avoid this, we need to check the keys of the definitions map and
	// see if the entry point is there and if not we will not attempt to look
	// for it.
	definitions := instance.ExportedFunctionDefinitions()
	_, found := definitions[h.spec.Entrypoint]

	if !found {
		h.result = executor.NewFailedResult(
			fmt.Sprintf("unable to find the entrypoint '%s' in the WASM module", h.spec.Entrypoint),
		)
		return
	}

	entryFunc := instance.ExportedFunction(h.spec.Entrypoint)
	h.logger.Info().Msg("running execution")

	// TODO(forrest): this is a bit of a race condition as the operation has not started when these lines are called.
	h.running.Store(true)
	close(h.activeCh)

	// The function should exit which results in a sys.ExitError. So we capture
	// the exit code for inclusion in the job output, and ignore the return code
	// from the function (most WASI compilers will not give one). Some compilers
	// though do not set an exit code, so we use a default of -1.
	_, wasmErr := entryFunc.Call(ctx)
	exitCode := int64(-1)
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

	// Drain any remaining logs
	h.logManager.Drain()

	// Collect results
	stdoutReader, stderrReader := h.logManager.GetDefaultReaders(false)
	resultsDir := compute.ExecutionResultsDir(h.request.ResultsDir, h.request.ExecutionID)
	h.result = executor.WriteJobResults(resultsDir, stdoutReader, stderrReader, int(exitCode), wasmErr, h.request.OutputLimits)
}

// active returns whether the execution is currently running
func (h *executionHandler) active() bool {
	return h.running.Load()
}

// kill cancels the execution
func (h *executionHandler) kill(ctx context.Context) error {
	h.cancel()
	return nil
}

// outputStream provides a stream of execution logs
func (h *executionHandler) outputStream(ctx context.Context, request messages.ExecutionLogsRequest) (io.ReadCloser, error) {
	return h.logManager.GetMuxedReader(request.Follow), nil
}
