package wasm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/c2h5oh/datasize"
	"github.com/rs/zerolog/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/sys"
	"go.uber.org/multierr"
	"golang.org/x/exp/maps"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	wasmlogs "github.com/bacalhau-project/bacalhau/pkg/logger/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/bacalhau-project/bacalhau/pkg/util/filefs"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/bacalhau-project/bacalhau/pkg/util/mountfs"
	"github.com/bacalhau-project/bacalhau/pkg/util/touchfs"
)

type Executor struct {
	StorageProvider storage.StorageProvider
	logManagers     generic.SyncMap[string, *wasmlogs.LogManager]
}

func NewExecutor(_ context.Context, storageProvider storage.StorageProvider) (*Executor, error) {
	return &Executor{
		StorageProvider: storageProvider,
	}, nil
}

func (e *Executor) IsInstalled(context.Context) (bool, error) {
	// WASM executor runs natively in Go and so is always available
	return true, nil
}

func (e *Executor) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.Executor.HasStorageLocally")
	defer span.End()

	s, err := e.StorageProvider.Get(ctx, volume.StorageSource)
	if err != nil {
		return false, err
	}

	return s.HasStorageLocally(ctx, volume)
}

func (e *Executor) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.Executor.GetVolumeSize")
	defer span.End()

	storageProvider, err := e.StorageProvider.Get(ctx, volume.StorageSource)
	if err != nil {
		return 0, err
	}
	return storageProvider.GetVolumeSize(ctx, volume)
}

// GetBidStrategy implements executor.Executor
func (*Executor) GetSemanticBidStrategy(context.Context) (bidstrategy.SemanticBidStrategy, error) {
	return semantic.NewChainedSemanticBidStrategy(), nil
}

func (*Executor) GetResourceBidStrategy(context.Context) (bidstrategy.ResourceBidStrategy, error) {
	return resource.NewChainedResourceBidStrategy(), nil
}

// makeFsFromStorage sets up a virtual filesystem (represented by an fs.FS) that
// will be the filesystem exposed to our WASM. The strategy for this is to:
//
//   - mount each input at the name specified by Path
//   - make a directory in the job results directory for each output and mount that
//     at the name specified by Name
func (e *Executor) makeFsFromStorage(ctx context.Context, jobResultsDir string, inputs, outputs []model.StorageSpec) (fs.FS, error) {
	var err error
	rootFs := mountfs.New()

	volumes, err := storage.ParallelPrepareStorage(ctx, e.StorageProvider, inputs)
	if err != nil {
		return nil, err
	}

	for input, volume := range volumes {
		log.Ctx(ctx).Debug().
			Str("input", input.Path).
			Str("source", volume.Source).
			Msg("Using input")

		var stat os.FileInfo
		stat, err = os.Stat(volume.Source)
		if err != nil {
			return nil, err
		}

		var inputFs fs.FS
		if stat.IsDir() {
			inputFs = os.DirFS(volume.Source)
		} else {
			inputFs = filefs.New(volume.Source)
		}

		err = rootFs.Mount(input.Path, inputFs)
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

//nolint:funlen
func (e *Executor) Run(ctx context.Context, executionID string, job model.Job, jobResultsDir string) (*model.RunCommandResult, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.Executor.Run")
	defer span.End()

	engineConfig := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)

	// Apply memory limits to the runtime. We have to do this in multiples of
	// the WASM page size of 64kb, so round up to the nearest page size if the
	// limit is not specified as a multiple of that.
	if job.Spec.Resources.Memory != "" {
		memoryLimit, err := datasize.ParseString(job.Spec.Resources.Memory)
		if err != nil {
			return executor.FailResult(err)
		}

		const pageSize = 65536
		pageLimit := memoryLimit.Bytes()/pageSize + system.Min(memoryLimit.Bytes()%pageSize, 1)
		engineConfig = engineConfig.WithMemoryLimitPages(uint32(pageLimit))
	}

	engine := tracedRuntime{wazero.NewRuntimeWithConfig(ctx, engineConfig)}
	defer closer.ContextCloserWithLogOnError(ctx, "engine", engine)

	rootFs, err := e.makeFsFromStorage(ctx, jobResultsDir, job.Spec.Inputs, job.Spec.Outputs)
	if err != nil {
		return executor.FailResult(err)
	}

	// Create a new log manager and obtain some writers that we can pass to the wasm
	// configuration
	logs, err := wasmlogs.NewLogManager(ctx, executionID)
	if err != nil {
		return executor.FailResult(err)
	}
	stdout, stderr := logs.GetWriters()

	// Store the LogManager for the lifetime of the execution, making sure to tidy up
	// once complete.
	e.logManagers.Put(executionID, logs)
	defer func() {
		log.Ctx(ctx).Debug().Str("Execution", executionID).Msg("cleaning up logmanager for execution")
		logs.Close()
		e.logManagers.Delete(executionID)
		log.Ctx(ctx).Debug().Str("Execution", executionID).Msg("logmanager being removed")
	}()

	// Configure the modules. We don't want to execute any start functions
	// automatically as we will do it manually later. Finally, add the
	// filesystem which contains our input and output.

	args := append([]string{job.Spec.Wasm.EntryModule.Name}, job.Spec.Wasm.Parameters...)

	config := wazero.NewModuleConfig().
		WithStartFunctions().
		WithStdout(stdout).
		WithStderr(stderr).
		WithArgs(args...).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithFS(rootFs)

	keys := maps.Keys(job.Spec.Wasm.EnvironmentVariables)
	sort.Strings(keys)
	for _, key := range keys {
		// Make sure we add the environment variables in a consistent order
		config = config.WithEnv(key, job.Spec.Wasm.EnvironmentVariables[key])
	}

	// Load and instantiate imported modules
	loader := NewModuleLoader(engine, config, e.StorageProvider)
	for _, importModule := range job.Spec.Wasm.ImportModules {
		_, ierr := loader.InstantiateRemoteModule(ctx, importModule)
		err = multierr.Append(err, ierr)
	}

	// Load and instantiate the entry module.
	instance, err := loader.InstantiateRemoteModule(ctx, job.Spec.Wasm.EntryModule)
	if err != nil {
		return executor.FailResult(err)
	}

	// The function should exit which results in a sys.ExitError. So we capture
	// the exit code for inclusion in the job output, and ignore the return code
	// from the function (most WASI compilers will not give one). Some compilers
	// though do not set an exit code, so we use a default of -1.
	log.Ctx(ctx).Debug().
		Str("entryPoint", job.Spec.Wasm.EntryPoint).
		Str("job", job.ID()).
		Str("execution", executionID).
		Msg("Running WASM job")
	entryFunc := instance.ExportedFunction(job.Spec.Wasm.EntryPoint)
	exitCode := -1
	_, wasmErr := entryFunc.Call(ctx)

	var errExit *sys.ExitError
	if errors.As(wasmErr, &errExit) {
		exitCode = int(errExit.ExitCode())
		wasmErr = nil
	}

	// execution has finished and there's nothing else to read from so inform
	// the logs that it is time to drain any remaining items.
	logs.Drain()

	stdoutReader, stderrReader := logs.GetDefaultReaders(false)
	return executor.WriteJobResults(jobResultsDir, stdoutReader, stderrReader, exitCode, wasmErr)
}

func (e *Executor) GetOutputStream(ctx context.Context, executionID string, withHistory bool, follow bool) (io.ReadCloser, error) {
	logs, present := e.logManagers.Get(executionID)
	if !present {
		log.Ctx(ctx).Debug().Str("Execution", executionID).Msg("logmanager for wasm execution was already removed")
		return nil, fmt.Errorf("logmanager has completed, no logs available")
	}

	return logs.GetMuxedReader(follow), nil
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
