package wasm

import (
	"context"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"go.ptx.dk/multierrgroup"
)

// ModuleLoader handles the loading and instantiation of WebAssembly modules.
// It manages module dependencies and ensures proper initialization order.
//
// The loader supports:
// - Loading modules from the mounted filesystem
// - Dynamic resolution of imported modules at runtime
// - Automatic handling of WASI modules
// - Thread-safe module instantiation
//
// Dynamic Loading:
// When a WASM module imports another module by name, the loader will:
// 1. Check if the module is already instantiated
// 2. If not, check if it's the WASI module
// 3. If not, look for a matching .wasm file in the mounted filesystem
// 4. If found, load and instantiate the module recursively
// 5. If not found, return an error
//
// Module Resolution:
// - Direct paths: Load the WASM file at the given path
// - Directories: Look for a single .wasm file in the directory
// - Module names: Resolve against the mounted filesystem
type ModuleLoader struct {
	// runtime is the WASM runtime used for module compilation and instantiation
	runtime wazero.Runtime
	// config contains the configuration for module instantiation
	config wazero.ModuleConfig
	// fs is the filesystem where modules are mounted
	fs fs.FS

	// mtx ensures thread-safe module instantiation
	// The runtime will throw an error if the same module is instantiated more than once
	mtx sync.Mutex
}

// NewModuleLoader creates a new module loader with the given runtime, configuration,
// and mounted filesystem.
func NewModuleLoader(runtime wazero.Runtime, config wazero.ModuleConfig, fs fs.FS) *ModuleLoader {
	return &ModuleLoader{
		runtime: runtime,
		config:  config,
		fs:      fs,
	}
}

// InstantiateModule loads and instantiates the module at the given path and all of
// its dependencies. It looks in the provided filesystem for modules.
//
// This function calls itself recursively for any discovered dependencies on the
// loaded modules, so that the returned module has all of its dependencies fully
// instantiated and is ready to use.
func (loader *ModuleLoader) InstantiateModule(ctx context.Context, modulePath string) (api.Module, error) {
	// Check if module is already instantiated
	if module := loader.runtime.Module(modulePath); module != nil {
		return module, nil
	}

	// Load the module from the mounted filesystem
	compiledModule, err := loader.loadModuleByPath(ctx, modulePath)
	if err != nil {
		return nil, err
	}

	// Load all dependencies in parallel
	var wg multierrgroup.Group
	for _, importedFunc := range compiledModule.ImportedFunctions() {
		moduleName, _, _ := importedFunc.Import()
		wg.Go(func() error {
			_, err := loader.loadModuleByName(ctx, moduleName)
			return err
		})
	}
	if err = wg.Wait(); err != nil {
		return nil, NewModuleLoadError(modulePath, err)
	}

	// We now have all dependencies loaded, so load this module.
	loader.mtx.Lock()
	defer loader.mtx.Unlock()

	// Double-check module hasn't been instantiated while we were loading dependencies
	if module := loader.runtime.Module(modulePath); module != nil {
		return module, nil
	}

	module, err := loader.runtime.InstantiateModule(ctx, compiledModule, loader.config.WithName(modulePath))
	if err != nil {
		return nil, NewModuleLoadError(modulePath, err)
	}
	return module, nil
}

// loadModuleByPath loads and compiles a module from the filesystem.
// If the path is a directory containing a single file, that file is used.
func (loader *ModuleLoader) loadModuleByPath(ctx context.Context, path string) (wazero.CompiledModule, error) {
	log.Ctx(ctx).Debug().Str("Path", path).Msg("Loading WASM module")
	resolvedPath, err := loader.resolveModulePath(path)
	if err != nil {
		return nil, err
	}

	return loader.loadFile(ctx, resolvedPath)
}

// loadModuleByName handles dynamic module imports by name.
// It first checks if the module is already instantiated or is the WASI module.
// If not, it attempts to resolve the module name against the mounted filesystem
// and load it recursively.
//
// The function is thread-safe and handles concurrent loading of the same module.
func (loader *ModuleLoader) loadModuleByName(ctx context.Context, moduleName string) (api.Module, error) {
	// First check if already instantiated or is WASI
	if module, err := func() (api.Module, error) {
		loader.mtx.Lock()
		defer loader.mtx.Unlock()
		if module := loader.runtime.Module(moduleName); module != nil {
			// Module already instantiated.
			return module, nil
		}

		if moduleName == wasi_snapshot_preview1.ModuleName {
			_, err := wasi_snapshot_preview1.NewBuilder(loader.runtime).Instantiate(ctx)
			if err != nil {
				return nil, NewWASIError(err)
			}
			return loader.runtime.Module(moduleName), nil
		}

		return nil, nil
	}(); module != nil || err != nil {
		return module, err
	}

	// Try to find and load from filesystem
	modulePath, err := loader.resolveModulePath(moduleName)
	if err == nil {
		return loader.InstantiateModule(ctx, modulePath)
	}

	return nil, NewUnknownModuleError(moduleName)
}

// resolveModulePath resolves a module path or name to an actual file path.
// If the path is a directory, it looks for a single .wasm file in that directory.
// Returns an error if the path cannot be resolved or if a directory contains
// multiple .wasm files.
func (loader *ModuleLoader) resolveModulePath(path string) (string, error) {
	info, err := fs.Stat(loader.fs, path)
	if err != nil {
		return "", NewModuleNotFoundError(path)
	}

	// If path is a directory, look for a single WASM file
	if info.IsDir() {
		files, err := fs.ReadDir(loader.fs, path)
		if err != nil {
			return "", NewModuleLoadError(path, err)
		}

		var wasmFiles []fs.DirEntry
		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".wasm" {
				wasmFiles = append(wasmFiles, file)
			}
		}

		if len(wasmFiles) != 1 {
			return "", NewModuleNotFoundError(path)
		}
		path = filepath.Join(path, wasmFiles[0].Name())
	}
	return path, nil
}

// loadFile compiles a WASM module from the given path in the mounted filesystem.
// It reads the module bytes and compiles them for execution.
func (loader *ModuleLoader) loadFile(ctx context.Context, path string) (wazero.CompiledModule, error) {
	bytes, err := fs.ReadFile(loader.fs, path)
	if err != nil {
		return nil, NewModuleLoadError(path, err)
	}

	module, err := loader.runtime.CompileModule(ctx, bytes)
	if err != nil {
		return nil, NewModuleCompileError(path, err)
	}

	return module, nil
}
