package model

import (
	"strings"

	"github.com/rs/zerolog/log"
)

//go:generate stringer -type=Engine --trimprefix=Engine
type Engine int

const (
	engineUnknown Engine = iota // must be first
	EngineNoop
	EngineDocker
	EngineWasm
	engineDone // must be last
)

func IsValidEngine(e Engine) bool {
	return e > engineUnknown && e < engineDone
}

// ParseEngine will either return a valid engine type or `engineUnknown`
func ParseEngine(str string) Engine {
	for typ := engineUnknown + 1; typ < engineDone; typ++ {
		if strings.EqualFold(typ.String(), str) {
			return typ
		}
	}

	// NB: change introduced in #2552 due to remove of language and pythonwasm engine types.
	log.Warn().Msgf("executor: unknown engine type: '%s'", str)
	return engineUnknown
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
	*e = ParseEngine(name)
	return
}
