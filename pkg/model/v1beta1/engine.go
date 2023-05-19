package v1beta1

import (
	"fmt"
	"strings"
)

//go:generate stringer -type=Engine --trimprefix=Engine
type Engine int

const (
	engineUnknown Engine = iota // must be first
	EngineNoop
	EngineDocker
	EngineWasm
	EngineLanguage   // wraps python_wasm
	EnginePythonWasm // wraps docker
	engineDone       // must be last
)

func IsValidEngine(e Engine) bool {
	return e > engineUnknown && e < engineDone
}

func ParseEngine(str string) (Engine, error) {
	for typ := engineUnknown + 1; typ < engineDone; typ++ {
		if strings.EqualFold(typ.String(), str) {
			return typ, nil
		}
	}

	return engineUnknown, fmt.Errorf(
		"executor: unknown engine type '%s'", str)
}

func EngineTypes() []Engine {
	var res []Engine
	for typ := engineUnknown + 1; typ < engineDone; typ++ {
		res = append(res, typ)
	}

	return res
}

func EngineNames() []string {
	var names []string
	for _, typ := range EngineTypes() {
		names = append(names, typ.String())
	}
	return names
}

func (e Engine) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

func (e *Engine) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*e, err = ParseEngine(name)
	return
}
