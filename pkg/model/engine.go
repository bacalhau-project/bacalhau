package model

import (
	"fmt"
)

//go:generate stringer -type=Engine --trimprefix=Engine
type Engine int

const (
	engineUnknown Engine = iota // must be first
	EngineNoop
	EngineDocker
	EngineWasm       // raw wasm executor not implemented yet
	EngineLanguage   // wraps python_wasm
	EnginePythonWasm // wraps docker
	engineDone       // must be last
)

func IsValidEngineType(e Engine) bool {
	return e > engineUnknown && e < engineDone
}

func ParseEngineType(str string) (Engine, error) {
	for typ := engineUnknown + 1; typ < engineDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return engineUnknown, fmt.Errorf(
		"executor: unknown engine type '%s'", str)
}

func EnsureEngineType(typ Engine, str string) (Engine, error) {
	if IsValidEngineType(typ) {
		return typ, nil
	}
	return ParseEngineType(str)
}

func EngineTypes() []Engine {
	var res []Engine
	for typ := engineUnknown + 1; typ < engineDone; typ++ {
		res = append(res, typ)
	}

	return res
}
