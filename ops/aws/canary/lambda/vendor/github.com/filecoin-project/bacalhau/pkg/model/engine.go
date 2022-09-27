package model

import (
	"fmt"
)

//go:generate stringer -type=EngineType --trimprefix=Engine
type EngineType int

const (
	engineUnknown EngineType = iota // must be first
	EngineNoop
	EngineDocker
	EngineWasm       // raw wasm executor not implemented yet
	EngineLanguage   // wraps python_wasm
	EnginePythonWasm // wraps docker
	engineDone       // must be last
)

func IsValidEngineType(engineType EngineType) bool {
	return engineType > engineUnknown && engineType < engineDone
}

func ParseEngineType(str string) (EngineType, error) {
	for typ := engineUnknown + 1; typ < engineDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return engineUnknown, fmt.Errorf(
		"executor: unknown engine type '%s'", str)
}

func EnsureEngineType(typ EngineType, str string) (EngineType, error) {
	if IsValidEngineType(typ) {
		return typ, nil
	}
	return ParseEngineType(str)
}

func EngineTypes() []EngineType {
	var res []EngineType
	for typ := engineUnknown + 1; typ < engineDone; typ++ {
		res = append(res, typ)
	}

	return res
}
