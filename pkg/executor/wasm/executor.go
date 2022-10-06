package wasm

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type Executor struct {
	Engine          *wasmtime.Engine
	StorageProvider storage.StorageProvider
}

func NewExecutor(
	ctx context.Context,
	storageProvider storage.StorageProvider,
) (*Executor, error) {
	// TODO: add host-specific config about WASM runtime and mem limits
	engine := wasmtime.NewEngine()

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

func (e *Executor) loadRemoteModule(ctx context.Context, spec model.StorageSpec, programName string) (*wasmtime.Module, error) {
	log.Ctx(ctx).Info().Msgf("Getting object %v", spec)
	storage, err := e.StorageProvider.GetStorage(ctx, spec.StorageSource)
	if err != nil {
		return nil, err
	}

	volume, err := storage.PrepareStorage(ctx, spec)
	if err != nil {
		return nil, err
	}

	// Generate a WASM module fm that.
	log.Ctx(ctx).Info().Msgf("Loading WASM module from remote '%s'", volume.Target)
	programPath := filepath.Join(volume.Source, filepath.Base(programName))
	module, err := wasmtime.NewModuleFromFile(e.Engine, programPath)
	if err != nil {
		return nil, err
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

	// Check that it conforms to our requirements.
	err = ValidateModuleAgainstJob(module, shard.Job.Spec)
	if err != nil {
		return failResult(err)
	}

	// Now instantiate the module and run the entry point.
	store := wasmtime.NewStore(e.Engine)
	instance, err := wasmtime.NewInstance(store, module, []wasmtime.AsExtern{})
	if err != nil {
		return failResult(err)
	}

	log.Ctx(ctx).Info().Msgf("Running WASM '%s' from job '%s'", shard.Job.Spec.Language.Command, shard.Job.ID)
	entryPoint := instance.GetFunc(store, shard.Job.Spec.Language.Command)
	returnValue, err := entryPoint.Call(store)
	if err != nil {
		return failResult(err)
	}

	// Current assumption: func returns one i32
	exitCode, ok := returnValue.(int32)
	if !ok {
		err = fmt.Errorf("WASM failed to return an exit code of i32")
		return failResult(err)
	}

	return &model.RunCommandResult{
		ExitCode: int(exitCode),
	}, nil
}
