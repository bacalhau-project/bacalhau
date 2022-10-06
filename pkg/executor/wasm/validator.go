package wasm

import (
	"fmt"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"golang.org/x/exp/slices"
)

func ValidateModuleAgainstJob(
	module *wasmtime.Module,
	job model.Spec,
) error {
	if !job.Language.Deterministic {
		return fmt.Errorf("WASM jobs are all deterministic but Deterministic is not set to true")
	}

	if len(module.Imports()) > 0 {
		return fmt.Errorf("imports are specified for the WASM module but there should be none")
	}

	entryPoint := job.Language.Command
	entryFuncIndex := slices.IndexFunc(module.Exports(), func(export *wasmtime.ExportType) bool {
		return export.Name() == entryPoint
	})
	if entryFuncIndex < 0 {
		return fmt.Errorf("job specifies '%s' as the entry point but no WASM export with that name was found", entryPoint)
	}

	entryFunc := module.Exports()[entryFuncIndex]
	entryFuncType := entryFunc.Type().FuncType()
	if entryFuncType == nil {
		return fmt.Errorf("job specifies '%s' as the entry point but it is not a function", entryPoint)
	}

	if len(entryFuncType.Params()) != 0 {
		return fmt.Errorf("entry point '%s' should take 0 parameters", entryPoint)
	}
	if len(entryFuncType.Results()) != 1 {
		return fmt.Errorf("entry point '%s' should return 1 result", entryPoint)
	}
	returnType := entryFuncType.Results()[0]
	if returnType.Kind() != wasmtime.KindI32 {
		return fmt.Errorf("entry point '%s' should return an i32", entryPoint)
	}

	return nil
}
