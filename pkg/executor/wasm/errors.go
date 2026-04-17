package wasm

import (
	"fmt"
	"net/http"

	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
)

const Component = "WASM"

// WASM-specific error codes
const (
	ModuleNotFound     = "ModuleNotFound"
	ModuleCompileError = "ModuleCompileError"
	ModuleLoadError    = "ModuleLoadError"
	WASIError          = "WASIError"
	UnknownModuleError = "UnknownModuleError"
	// Handler and Executor specific error codes
	SpecError        = "SpecError"
	LogError         = "LogError"
	EntrypointError  = "EntrypointError"
	MemoryLimitError = "MemoryLimitError"
	FilesystemError  = "FilesystemError"
	// Configuration error codes
	InputConfigError  = "InputConfigError"
	OutputConfigError = "OutputConfigError"
)

// NewModuleNotFoundError creates an error when a WASM module cannot be found in the filesystem.
func NewModuleNotFoundError(path string) bacerrors.Error {
	return bacerrors.Newf("module not found: %q", path).
		WithCode(ModuleNotFound).
		WithHTTPStatusCode(http.StatusNotFound).
		WithComponent(Component).
		WithHint(`To resolve this:
1. Ensure the module file exists at the specified path
2. If using a directory, make sure it contains exactly one .wasm file
3. Check that the module has been properly added to the job's input sources
4. Verify the target name matches the module's location in the filesystem`)
}

// NewModuleCompileError creates an error when a WASM module fails to compile.
func NewModuleCompileError(path string, err error) bacerrors.Error {
	return bacerrors.Wrapf(err, "failed to compile module at %q", path).
		WithCode(ModuleCompileError).
		WithHTTPStatusCode(http.StatusBadRequest).
		WithComponent(Component).
		WithHint("Check that the module is compatible with the WASM runtime and follows the correct format")
}

// NewModuleLoadError creates an error when a WASM module fails to load.
func NewModuleLoadError(path string, err error) bacerrors.Error {
	return bacerrors.Wrapf(err, "failed to load module at %q", path).
		WithCode(ModuleLoadError).
		WithHTTPStatusCode(http.StatusInternalServerError).
		WithComponent(Component).
		WithHint("This could be due to memory constraints or runtime issues. Check system resources and try again")
}

// NewWASIError creates an error when there's an issue with the WASI module.
func NewWASIError(err error) bacerrors.Error {
	return bacerrors.Wrap(err, "WASI module error").
		WithCode(WASIError).
		WithHTTPStatusCode(http.StatusInternalServerError).
		WithComponent(Component).
		WithHint("This is an internal error with the WASI module. Please report this issue")
}

// NewUnknownModuleError creates an error when a module name cannot be resolved.
func NewUnknownModuleError(moduleName string) bacerrors.Error {
	return bacerrors.Newf("unknown module %q", moduleName).
		WithCode(UnknownModuleError).
		WithHTTPStatusCode(http.StatusBadRequest).
		WithComponent(Component).
		WithHint(fmt.Sprintf(`To resolve this:
1. Check if the module name is correct
2. Ensure the module is available in the job's input sources
3. Verify the module's target name matches the import name
4. If using WASI, make sure the module name matches %q`, wasi_snapshot_preview1.ModuleName))
}

// NewSpecError creates an error when there's an issue with the WASM spec
func NewSpecError(err error) bacerrors.Error {
	return bacerrors.Wrap(err, "invalid WASM spec").
		WithCode(SpecError).
		WithHTTPStatusCode(http.StatusBadRequest).
		WithComponent(Component).
		WithHint("Check that the WASM spec is properly formatted and contains all required fields")
}

// NewLogError creates an error when there's an issue with logging
func NewLogError(err error) bacerrors.Error {
	return bacerrors.Wrap(err, "logging error").
		WithCode(LogError).
		WithHTTPStatusCode(http.StatusInternalServerError).
		WithComponent(Component).
		WithHint("This is an internal error with the logging system. Please report this issue")
}

// NewEntrypointError creates an error when there's an issue with the entrypoint
func NewEntrypointError(entrypoint string) bacerrors.Error {
	return bacerrors.Newf("entrypoint '%s' not found", entrypoint).
		WithCode(EntrypointError).
		WithHTTPStatusCode(http.StatusBadRequest).
		WithComponent(Component).
		WithHint("Check that the specified entrypoint exists in the WASM module")
}

// NewMemoryLimitError creates an error when memory limits are exceeded
func NewMemoryLimitError(requested, max uint64) bacerrors.Error {
	return bacerrors.Newf("requested memory exceeds the wasm limit - %.2f GB > %.2f GB",
		float64(requested)/float64(BytesInGB),
		float64(max)/float64(BytesInGB)).
		WithCode(MemoryLimitError).
		WithHTTPStatusCode(http.StatusBadRequest).
		WithComponent(Component).
		WithHint("Reduce the requested memory to be within the WASM limit")
}

// NewFilesystemError creates an error when there's an issue with the filesystem
func NewFilesystemError(path string, err error) bacerrors.Error {
	return bacerrors.Wrapf(err, "filesystem error at %q", path).
		WithCode(FilesystemError).
		WithHTTPStatusCode(http.StatusInternalServerError).
		WithComponent(Component).
		WithHint("This is an internal error with the filesystem. Please report this issue")
}

// NewOutputError creates an error when there's an issue with output configuration
func NewOutputError(msg string) bacerrors.Error {
	return bacerrors.Newf("output configuration error: %s", msg).
		WithCode(OutputConfigError).
		WithHTTPStatusCode(http.StatusBadRequest).
		WithComponent(Component).
		WithHint(`To resolve this:
1. Check that all output volumes have both a name and path specified
2. Ensure output paths are valid and accessible
3. Verify that output names are unique
4. Make sure output paths don't conflict with input paths`)
}

// NewInputConfigError creates an error when there's an issue with input configuration
func NewInputConfigError(msg string) bacerrors.Error {
	return bacerrors.Newf("input configuration error: %s", msg).
		WithCode(InputConfigError).
		WithHTTPStatusCode(http.StatusBadRequest).
		WithComponent(Component).
		WithHint(`To resolve this:
1. Check that all input sources have valid paths
2. Ensure input targets are unique and don't conflict with output paths
3. Verify that input sources exist and are accessible
4. Make sure input paths are properly formatted`)
}
