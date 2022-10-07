package wasm

import (
	"fmt"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"golang.org/x/exp/slices"
)

// ValidateModuleAgainstJob will return an error if the passed job does not
// represent a valid WASM executor job or the passed module is not able to be
// run to fulfill the job.
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

	return ValidateModuleAsEntryPoint(module, job.Language.Command)
}

// ValidateModuleAsEntryPoint returns an error if the passed module is not
// capable of being an entry point to a job, i.e. that it contains a function of
// the passed name that meets the specification of:
//
// - the named function exists and is exported
// - the function takes no parameters
// - the function returns one i32 value (exit code)
func ValidateModuleAsEntryPoint(
	module *wasmtime.Module,
	name string,
) error {
	return ValidateModuleHasFunction(
		module,
		name,
		[]wasmtime.ValKind{},
		[]wasmtime.ValKind{wasmtime.KindI32},
	)
}

// ValidateModuleHasFunction returns an error if the passed module does not
// contain an exported function with the passed name, parameters and return
// values.
func ValidateModuleHasFunction(
	module *wasmtime.Module,
	name string,
	parameters []wasmtime.ValKind,
	results []wasmtime.ValKind,
) error {
	funcIndex := slices.IndexFunc(module.Exports(), func(export *wasmtime.ExportType) bool {
		return export.Name() == name
	})
	if funcIndex < 0 {
		return fmt.Errorf("function '%s' required but no WASM export with that name was found", name)
	}

	function := module.Exports()[funcIndex]
	funcType := function.Type().FuncType()
	if funcType == nil {
		return fmt.Errorf("'%s' is not exported as a function", name)
	}

	if len(funcType.Params()) != len(parameters) {
		return fmt.Errorf("function '%s' should take %d parameters", name, len(parameters))
	}
	for i := range parameters {
		expectedType := parameters[i]
		actualType := funcType.Params()[i].Kind()
		if expectedType != actualType {
			return fmt.Errorf("function '%s': expected param %d to have type %v", name, i, expectedType)
		}
	}

	if len(funcType.Results()) != len(results) {
		return fmt.Errorf("function '%s' should return %d results", name, len(results))
	}
	for i := range results {
		expectedType := results[i]
		actualType := funcType.Results()[i].Kind()
		if expectedType != actualType {
			return fmt.Errorf("function '%s': expected result %d to have type %v", name, i, expectedType)
		}
	}

	return nil
}
