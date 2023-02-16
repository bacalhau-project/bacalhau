package wasm

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"golang.org/x/exp/maps"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/filefs"
	"github.com/filecoin-project/bacalhau/pkg/util/mountfs"
	"github.com/filecoin-project/bacalhau/pkg/util/touchfs"
	"github.com/rs/zerolog/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

type Executor struct {
	StorageProvider storage.StorageProvider
}

func NewExecutor(
	_ context.Context,
	storageProvider storage.StorageProvider,
) (*Executor, error) {
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
		log.Ctx(ctx).Debug().Msgf("Using input '%s' at '%s'", input.Path, volume.Source)

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
		log.Ctx(ctx).Debug().Msgf("Collecting output '%s' at '%s'", output.Name, srcd)

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

//nolint:funlen  // Will be made shorter when we do more module linking
func (e *Executor) RunShard(
	ctx context.Context,
	shard model.JobShard,
	jobResultsDir string,
) (*model.RunCommandResult, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.Executor.RunShard")
	defer span.End()

	cache := wazero.NewCompilationCache()
	engineConfig := wazero.NewRuntimeConfig().WithCompilationCache(cache)

	// Apply memory limits to the runtime. We have to do this in multiples of
	// the WASM page size of 64kb, so round up to the nearest page size if the
	// limit is not specified as a multiple of that.
	if shard.Job.Spec.Resources.Memory != "" {
		memoryLimit, err := datasize.ParseString(shard.Job.Spec.Resources.Memory)
		if err != nil {
			return executor.FailResult(err)
		}

		const pageSize = 65536
		pageLimit := memoryLimit.Bytes()/pageSize + system.Min(memoryLimit.Bytes()%pageSize, 1)
		engineConfig = engineConfig.WithMemoryLimitPages(uint32(pageLimit))
	}

	engine := wazero.NewRuntimeWithConfig(ctx, engineConfig)

	wasmSpec := shard.Job.Spec.Wasm
	contextStorageSpec := shard.Job.Spec.Wasm.EntryModule
	module, err := LoadRemoteModule(ctx, engine, e.StorageProvider, contextStorageSpec)
	if err != nil {
		return executor.FailResult(err)
	}
	defer module.Close(ctx)

	shardStorageSpec, err := job.GetShardStorageSpec(ctx, shard, e.StorageProvider)
	if err != nil {
		return executor.FailResult(err)
	}

	fs, err := e.makeFsFromStorage(ctx, jobResultsDir, shardStorageSpec, shard.Job.Spec.Outputs)
	if err != nil {
		return executor.FailResult(err)
	}

	// Configure the modules. We will write STDOUT and STDERR to a buffer so
	// that we can later include them in the job results. We don't want to
	// execute any start functions automatically as we will do it manually
	// later. Finally, add the filesystem which contains our input and output.
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	args := []string{module.Name()}
	args = append(args, wasmSpec.Parameters...)

	config := wazero.NewModuleConfig().
		WithStartFunctions().
		WithStdout(stdout).
		WithStderr(stderr).
		WithArgs(args...).
		WithFS(fs)

	keys := maps.Keys(wasmSpec.EnvironmentVariables)
	sort.Strings(keys)
	for _, key := range keys {
		// Make sure we add the environment variables in a consistent order
		config = config.WithEnv(key, wasmSpec.EnvironmentVariables[key])
	}

	entryPoint := wasmSpec.EntryPoint
	importedModules := []wazero.CompiledModule{}

	// Load and instantiate imported modules
	for _, wasmSpec := range wasmSpec.ImportModules {
		importedWasi, importErr := LoadRemoteModule(ctx, engine, e.StorageProvider, wasmSpec)
		if importErr != nil {
			return executor.FailResult(importErr)
		}
		importedModules = append(importedModules, importedWasi)

		_, instantiateErr := engine.InstantiateModule(ctx, importedWasi, config)
		if instantiateErr != nil {
			return executor.FailResult(instantiateErr)
		}
	}

	wasi, err := wasi_snapshot_preview1.NewBuilder(engine).Compile(ctx)
	if err != nil {
		return executor.FailResult(err)
	}
	defer wasi.Close(ctx)

	_, err = engine.InstantiateModule(ctx, wasi, config)
	if err != nil {
		return executor.FailResult(err)
	}

	// Now instantiate the module and run the entry point.
	instance, err := engine.InstantiateModule(ctx, module, config)
	if err != nil {
		return executor.FailResult(err)
	}

	// Check that all WASI modules conform to our requirements.
	importedModules = append(importedModules, wasi)
	err = ValidateModuleAgainstJob(module, shard.Job.Spec, importedModules...)
	if err != nil {
		return executor.FailResult(err)
	}

	// The function should exit which results in a sys.ExitError. So we capture
	// the exit code for inclusion in the job output, and ignore the return code
	// from the function (most WASI compilers will not give one). Some compilers
	// though do not set an exit code, so we use a default of -1.
	log.Ctx(ctx).Debug().Msgf("Running WASM %q from job %q", entryPoint, shard.Job.Metadata.ID)
	entryFunc := instance.ExportedFunction(entryPoint)
	exitCode := int(-1)
	_, wasmErr := entryFunc.Call(ctx)
	if wasmErr != nil {
		errExit, ok := wasmErr.(*sys.ExitError)
		if ok {
			exitCode = int(errExit.ExitCode())
			wasmErr = nil
		}
	}

	return executor.WriteJobResults(jobResultsDir, stdout, stderr, exitCode, wasmErr)
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
