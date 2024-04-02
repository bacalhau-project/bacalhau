package wasm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"go.opentelemetry.io/otel/attribute"
	"go.ptx.dk/multierrgroup"

	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// ModuleLoader handles the loading of WebAssembly modules from
// storage.PreparedStorage and the automatic resolution of required imports.
//
// ModuleLoader allows WebAssembly module dependencies to be specified within
// the WebAssembly program, allowing the user to deploy self-contained
// WebAssembly blobs. See the introductory talk at https://youtu.be/6zJkMLzXbQc.
//
// This works by using the "module name" field of a WebAssmelby import header,
// (which for user-supplied modules is arbitrary) as a hint to the loader as to
// where the dependency lives and how to retrieve it. The module still needs to
// be specified as input data for the job (a previous implementation of the
// ModuleLoader could load modules from remote locations, but this one cannot).
type ModuleLoader struct {
	runtime  wazero.Runtime
	config   wazero.ModuleConfig
	storages []storage.PreparedStorage

	// Runtime will throw an error if the same module is instantiated more than
	// once. So we use this mutex around checking for modules and instantiating
	mtx sync.Mutex
}

func NewModuleLoader(runtime wazero.Runtime, config wazero.ModuleConfig, storages ...storage.PreparedStorage) *ModuleLoader {
	return &ModuleLoader{runtime: runtime, config: config, storages: storages}
}

// Load compiles and returns a module located at the passed path.
func (loader *ModuleLoader) Load(ctx context.Context, path string) (wazero.CompiledModule, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.ModuleLoader.Load")
	span.SetAttributes(attribute.String("Path", path))
	defer span.End()

	log.Ctx(ctx).Debug().Str("Path", path).Msg("Loading WASM module")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	module, err := loader.runtime.CompileModule(ctx, bytes)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// loadModule loads and compiles all of the modules located by the passed storage specs.
func (loader *ModuleLoader) loadModule(ctx context.Context, m storage.PreparedStorage) (wazero.CompiledModule, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.ModuleLoader.loadModule")
	defer span.End()

	programPath := m.Volume.Source

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

	module, err := loader.Load(ctx, programPath)
	if err != nil {
		return nil, err
	}
	return module, err
}

// InstantiateRemoteModule loads and instantiates the remote module and all of
// its dependencies. It only looks in the job's input storage specs for modules.
//
// This function calls itself reucrsively for any discovered dependencies on the
// loaded modules, so that the returned module has all of its dependencies fully
// instantiated and is ready to use.
func (loader *ModuleLoader) InstantiateRemoteModule(ctx context.Context, m storage.PreparedStorage) (api.Module, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.ModuleLoader.InstantiateRemoteModule")
	span.SetAttributes(attribute.String("ModuleName", m.InputSource.Alias))
	defer span.End()

	if module := loader.runtime.Module(m.InputSource.Alias); module != nil {
		// Module already instantiated.
		return module, nil
	}

	// Get the remote module.
	module, err := loader.loadModule(ctx, m)
	if err != nil {
		return nil, err
	}

	// Examine its imports and recursively load them.
	var wg multierrgroup.Group
	for _, importedFunc := range module.ImportedFunctions() {
		moduleName, _, _ := importedFunc.Import()
		wg.Go(func() error {
			_, err := loader.loadModuleByName(ctx, moduleName)
			return err
		})
	}
	if err = wg.Wait(); err != nil {
		return nil, err
	}

	// We now have all dependencies loaded, so load this module.
	loader.mtx.Lock()
	defer loader.mtx.Unlock()

	if module := loader.runtime.Module(m.InputSource.Alias); module != nil {
		return module, nil
	}
	return loader.runtime.InstantiateModule(ctx, module, loader.config.WithName(m.InputSource.Alias))
}

const unknownModuleErrStr = ("could not find WASM module with name %q. " +
	"expecting a built-in module name (like %q), " +
	"or the Alias of one of the job's InputSources. " +
	"see also: https://docs.bacalhau.org/getting-started/wasm-workload-onboarding")

func (loader *ModuleLoader) loadModuleByName(ctx context.Context, moduleName string) (api.Module, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/executor/wasm.ModuleLoader.loadModuleByName")
	span.SetAttributes(attribute.String("ModuleName", moduleName))
	defer span.End()

	if module, err := func() (api.Module, error) {
		loader.mtx.Lock()
		defer loader.mtx.Unlock()
		if module := loader.runtime.Module(moduleName); module != nil {
			// Module already instantiated.
			return module, nil
		}

		if moduleName == wasi_snapshot_preview1.ModuleName {
			_, err := wasi_snapshot_preview1.NewBuilder(loader.runtime).Instantiate(ctx)
			return loader.runtime.Module(moduleName), err
		}

		return nil, nil
	}(); module != nil || err != nil {
		return module, err
	}

	// check if the module we are dynamically linking was specific in as an input to the job.
	// BUG: this just completely ignores the module name?? and
	// will presumably end up in an error if the first storage is not a WASM
	// module, or if the first storage is a WASM module but not the one we were
	// looking for?
	for _, s := range loader.storages {
		if s.InputSource.Alias == moduleName {
			return loader.InstantiateRemoteModule(ctx, s)
		}
	}

	return nil, fmt.Errorf(unknownModuleErrStr, moduleName, wasi_snapshot_preview1.ModuleName)
}
