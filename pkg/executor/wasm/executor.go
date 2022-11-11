package wasm

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/filecoin-project/bacalhau/pkg/executor"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/mountfs"
	"github.com/filecoin-project/bacalhau/pkg/util/touchfs"
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

func (e *Executor) IsInstalled(context.Context) (bool, error) {
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

func (e *Executor) loadRemoteModule(ctx context.Context, spec model.StorageSpec) (wazero.CompiledModule, error) {
	volume, err := e.getVolume(ctx, spec)
	if err != nil {
		return nil, err
	}

	programPath := volume.Source
	info, err := os.Stat(programPath)
	if err != nil {
		return nil, err
	}

	// We expect the input to be a single WASM file. It is common however for
	// IPFS implementations to wrap files into a directory. So we make a special
	// case here – if the input is a single file in a directory, we will assume
	// this is the program file and load it.
	if info.IsDir() {
		files, err := os.ReadDir(programPath)
		if err != nil {
			return nil, err
		}

		if len(files) != 1 {
			return nil, fmt.Errorf("should be %d file in %s but there are %d", 1, programPath, len(files))
		}
		programPath = filepath.Join(programPath, files[0].Name())
	}

	log.Ctx(ctx).Info().Msgf("Loading WASM module from '%s'", programPath)
	return LoadModule(ctx, e.Engine, programPath)
}

// makeFsFromStorage sets up a virtual filesystem (represented by an fs.FS) that
// will be the filesystem exposed to our WASM. The strategy for this is to:
//
//   - mount each input at the name specified by Path
//   - make a directory in the job results directory for each output and mount that
//     at the name specified by Name
func (e *Executor) makeFsFromStorage(ctx context.Context, jobResultsDir string, inputs, outputs []model.StorageSpec) (fs.FS, error) {
	var err error
	fs := mountfs.New()

	for _, input := range inputs {
		var volume *storage.StorageVolume
		volume, err = e.getVolume(ctx, input)
		if err != nil {
			return nil, err
		}

		log.Ctx(ctx).Info().Msgf("Using input '%s' at '%s'", input.Path, volume.Source)

		err = fs.Mount(input.Path, os.DirFS(volume.Source))
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
		log.Ctx(ctx).Info().Msgf("Collecting output '%s' at '%s'", output.Name, srcd)

		err = os.Mkdir(srcd, util.OS_ALL_R|util.OS_ALL_X|util.OS_USER_W)
		if err != nil {
			return nil, err
		}

		err = fs.Mount(output.Name, touchfs.New(srcd))
		if err != nil {
			return nil, err
		}
	}

	return fs, nil
}

func failResult(err error) (*model.RunCommandResult, error) {
	return &model.RunCommandResult{ErrorMsg: err.Error()}, err
}

//nolint:funlen  // Will be made shorter when we do more module linking
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

	wasmSpec := shard.Job.Spec.Wasm
	contextStorageSpec := shard.Job.Spec.Contexts[0]
	module, err := e.loadRemoteModule(ctx, contextStorageSpec)
	if err != nil {
		return failResult(err)
	}
	defer module.Close(ctx)

	shardStorageSpec, err := job.GetShardStorageSpec(ctx, shard, e.StorageProvider)
	if err != nil {
		return failResult(err)
	}

	fs, err := e.makeFsFromStorage(ctx, jobResultsDir, shardStorageSpec, shard.Job.Spec.Outputs)
	if err != nil {
		return failResult(err)
	}

	// Configure the modules. We will write STDOUT and STDERR to a buffer so
	// that we can later include them in the job results. We don't want to
	// execute any start functions automatically as we will do it manually
	// later. Finally, add the filesystem which contains our input and output.
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	args := []string{module.Name()}
	args = append(args, wasmSpec.Parameters...)

	namespace := e.Engine.NewNamespace(ctx)
	config := wazero.NewModuleConfig().
		WithStartFunctions().
		WithStdout(stdout).
		WithStderr(stderr).
		WithArgs(args...).
		WithFS(fs)
	for _, key := range keys(wasmSpec.EnvironmentVariables) {
		// Make sure we add the environment variables in a consistent order
		config = config.WithEnv(key, wasmSpec.EnvironmentVariables[key])
	}
	entryPoint := wasmSpec.EntryPoint

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
	_, wasmErr := entryFunc.Call(ctx)
	if wasmErr != nil {
		errExit, ok := wasmErr.(*sys.ExitError)
		if ok {
			exitCode = int(errExit.ExitCode())
			wasmErr = nil
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

	result := &model.RunCommandResult{
		STDOUT:   stdout.String(),
		STDERR:   stderr.String(),
		ExitCode: exitCode,
	}
	if wasmErr != nil {
		result.ErrorMsg = wasmErr.Error()
	}
	return result, wasmErr
}

func (e *Executor) CancelShard(ctx context.Context, shard model.JobShard) error {
	// TODO: Implement CancelShard for WASM executor #1060
	return nil
}

func keys(m map[string]string) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
