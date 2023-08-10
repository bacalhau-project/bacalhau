package wasm

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"golang.org/x/exp/maps"
)

// ValidateModuleAgainstJob will return an error if the passed job does not
// represent a valid WASM executor job or the passed module is not able to be
// run to fulfill the job.
func ValidateModuleAgainstJob(
	module wazero.CompiledModule,
	job models.Spec,
	importModules ...wazero.CompiledModule,
) error {
	err := ValidateModuleImports(module, importModules...)
	if err != nil {
		return err
	}

	return ValidateModuleAsEntryPoint(module, job.Wasm.EntryPoint)
}

// ValidateModuleImports will return an error if the passed module requires
// imports that are not found in any of the passed importModules. Imports have
// to match exactly, i.e. function names and signatures must be an exact match.
func ValidateModuleImports(
	module wazero.CompiledModule,
	importModules ...wazero.CompiledModule,
) error {
reqImport:
	for _, requiredImport := range module.ImportedFunctions() {
		importNamespace, funcName, _ := requiredImport.Import()
		exists := false
		for _, importModule := range importModules {
			log.Debug().Str("Func", funcName).Str("Module", importModule.Name()).Msg("Looking for import")
			for _, funct := range maps.Keys(importModule.ExportedFunctions()) {
				log.Debug().Str("Func", funct).Str("Module", importModule.Name()).Bool("Eq", funct == funcName).Msg("Has function")
				if funct == funcName {
					err := ValidateModuleHasFunction(
						importModule,
						funcName,
						requiredImport.ParamTypes(),
						requiredImport.ResultTypes(),
					)

					// If the module has the import but the signature doesn't match,
					// as we enforce that imports are unique, this will break even
					// if there is another import with correct name and signature.
					if err != nil {
						return err
					} else {
						log.Debug().Msg("kthnx")
						continue reqImport
					}
				}
			}
		}

		if !exists {
			// We didn't find an export from any module.
			return fmt.Errorf("no export found for '%s::%s' required by module", importNamespace, funcName)
		}
	}

	return nil
}

// ValidateModuleAsEntryPoint returns an error if the passed module is not
// capable of being an entry point to a job, i.e. that it contains a function of
// the passed name that meets the specification of:
//
// - the named function exists and is exported
// - the function takes no parameters
// - the function returns one i32 value (exit code)
func ValidateModuleAsEntryPoint(
	module wazero.CompiledModule,
	name string,
) error {
	return ValidateModuleHasFunction(
		module,
		name,
		[]api.ValueType{},
		[]api.ValueType{},
	)
}

// ValidateModuleHasFunction returns an error if the passed module does not
// contain an exported function with the passed name, parameters and return
// values.
func ValidateModuleHasFunction(
	module wazero.CompiledModule,
	name string,
	parameters []api.ValueType,
	results []api.ValueType,
) error {
	function, ok := module.ExportedFunctions()[name]
	if !ok {
		return fmt.Errorf("function '%s' required but no WASM export with that name was found", name)
	}

	if len(function.ParamTypes()) != len(parameters) {
		return fmt.Errorf("function '%s' should take %d parameters", name, len(parameters))
	}
	for i := range parameters {
		expectedType := parameters[i]
		actualType := function.ParamTypes()[i]
		if expectedType != actualType {
			return fmt.Errorf("function '%s': expected param %d to have type %v", name, i, expectedType)
		}
	}

	if len(function.ResultTypes()) != len(results) {
		return fmt.Errorf("function '%s' should return %d results", name, len(results))
	}
	for i := range results {
		expectedType := results[i]
		actualType := function.ResultTypes()[i]
		if expectedType != actualType {
			return fmt.Errorf("function '%s': expected result %d to have type %v", name, i, expectedType)
		}
	}

	return nil
}
