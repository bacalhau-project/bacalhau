package wasm

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

type Executor struct {
	Engine          wazero.Runtime
	StorageProvider storage.StorageProvider
}

func NewExecutor(
	ctx context.Context,
	storageProvider storage.StorageProvider,
) (*Executor, error) {
	// TODO: add host-specific config about WASM runtime and mem limits
	engine := wazero.NewRuntime(ctx)

	executor := &Executor{
		Engine:          engine,
		StorageProvider: storageProvider,
	}

	return executor, nil
}

func (e *Executor) IsInstalled(ctx context.Context) (bool, error) {
	// WASM executor runs natively in Go and so is always available
	return true, nil
}

func (e *Executor) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/executor/wasm/Executor.HasStorageLocally")
	defer span.End()

	s, err := e.StorageProvider.GetStorage(ctx, volume.StorageSource)
	if err != nil {
		return false, err
	}

	return s.HasStorageLocally(ctx, volume)
}

func (e *Executor) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/executor/wasm/Executor.GetVolumeSize")
	defer span.End()

	storageProvider, err := e.StorageProvider.GetStorage(ctx, volume.StorageSource)
	if err != nil {
		return 0, err
	}
	return storageProvider.GetVolumeSize(ctx, volume)
}

func (e *Executor) getVolume(ctx context.Context, spec model.StorageSpec) (*storage.StorageVolume, error) {
	log.Ctx(ctx).Info().Msgf("Getting object %v", spec)

	storage, err := e.StorageProvider.GetStorage(ctx, spec.StorageSource)
	if err != nil {
		return nil, err
	}

	volume, err := storage.PrepareStorage(ctx, spec)
	if err != nil {
		return nil, err
	}

	return &volume, nil
}

func (e *Executor) loadRemoteModule(ctx context.Context, spec model.StorageSpec, programName string) (wazero.CompiledModule, error) {
	volume, err := e.getVolume(ctx, spec)
	if err != nil {
		return nil, err
	}

	log.Ctx(ctx).Info().Msgf("Loading WASM module from remote '%s'", volume.Target)
	programPath := filepath.Join(volume.Source, filepath.Base(programName))
	return LoadModule(ctx, e.Engine, programPath)
}
	}

	return module, nil
}

func failResult(err error) (*model.RunCommandResult, error) {
	return &model.RunCommandResult{ErrorMsg: err.Error()}, err
}

func (e *Executor) RunShard(
	ctx context.Context,
	shard model.JobShard,
	jobResultsDir string,
) (*model.RunCommandResult, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/executor/wasm/Executor.RunShard")
	defer span.End()

	// Go and get the actual WASM we are going to run.
	if len(shard.Job.Spec.Contexts) < 1 {
		err := fmt.Errorf("WASM job expects one context containing code to run")
		return failResult(err)
	}

	contextStorageSpec := shard.Job.Spec.Contexts[0]
	module, err := e.loadRemoteModule(ctx, contextStorageSpec, shard.Job.Spec.Language.ProgramPath)
	if err != nil {
		return failResult(err)
	}
	defer module.Close(ctx)

	// Configure the modules. We will write STDOUT and STDERR to a buffer so
	// that we can later include them in the job results. We don't want to
	// execute any start functions automatically as we will do it manually
	// later. Finally, add the filesystem which contains our input and output.
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	namespace := e.Engine.NewNamespace(ctx)
	config := wazero.NewModuleConfig().
		WithStartFunctions().
		WithStdout(stdout).
		WithStderr(stderr)
	entryPoint := shard.Job.Spec.Language.Command

	log.Ctx(ctx).Info().Msgf("Compilation of WASI runtime for job '%s'", shard.Job.ID)
	wasi, err := wasi_snapshot_preview1.NewBuilder(e.Engine).Compile(ctx)
	if err != nil {
		return failResult(err)
	}
	defer wasi.Close(ctx)

	log.Ctx(ctx).Info().Msgf("Instantiating WASI runtime for job '%s'", shard.Job.ID)
	_, err = namespace.InstantiateModule(ctx, wasi, config)
	if err != nil {
		return failResult(err)
	}

	// Now instantiate the module and run the entry point.
	log.Ctx(ctx).Info().Msgf("Instantiation of module for job '%s'", shard.Job.ID)
	instance, err := namespace.InstantiateModule(ctx, module, config)
	if err != nil {
		return failResult(err)
	}

	// Check that it conforms to our requirements.
	err = ValidateModuleAgainstJob(module, shard.Job.Spec, wasi)
	if err != nil {
		return failResult(err)
	}

	// The function should exit which results in a sys.ExitError. So we capture
	// the exit code for inclusion in the job output, and ignore the return code
	// from the function (most WASI compilers will not give one). Some compilers
	// though do not set an exit code, so we use a default of -1.
	log.Ctx(ctx).Info().Msgf("Running WASM '%s' from job '%s'", entryPoint, shard.Job.ID)
	entryFunc := instance.ExportedFunction(entryPoint)
	exitCode := int(-1)
	_, err = entryFunc.Call(ctx)
	if err != nil {
		errExit, ok := err.(*sys.ExitError)
		if ok {
			exitCode = int(errExit.ExitCode())
		} else {
			return failResult(err)
		}
	}

	for filename, contents := range map[string][]byte{
		"stdout":   stdout.Bytes(),
		"stderr":   stderr.Bytes(),
		"exitCode": []byte(fmt.Sprint(exitCode)),
	} {
		err = os.WriteFile(filepath.Join(jobResultsDir, filename), contents, os.ModePerm)
		if err != nil {
			return failResult(err)
		}
	}

	return &model.RunCommandResult{
		STDOUT:   stdout.String(),
		STDERR:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}
