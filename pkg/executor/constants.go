package executor

import (
	"fmt"
	"strings"
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

func ParseEngineType(str string) (EngineType, error) {
	for typ := engineUnknown + 1; typ < engineDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return engineUnknown, fmt.Errorf(
		"executor: unknown engine type '%s'", str)
}

func EngineTypes() []EngineType {
	var res []EngineType
	for typ := engineUnknown + 1; typ < engineDone; typ++ {
		res = append(res, typ)
	}

	return res
}

//go:generate stringer -type=JobEventType --trimprefix=JobEvent
type JobEventType int

const (
	jobEventUnknown JobEventType = iota // must be first
	JobEventCreated
	JobEventDealUpdated
	JobEventBid
	JobEventBidAccepted
	JobEventBidRejected
	JobEventResults
	JobEventResultsAccepted
	JobEventResultsRejected
	JobEventError
	jobEventDone // must be last
)

func (event JobEventType) IsTerminal() bool {
	return event == JobEventError || event == JobEventResults
}

func ParseJobEventType(str string) (JobEventType, error) {
	for typ := jobEventUnknown + 1; typ < jobEventDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return jobEventUnknown, fmt.Errorf(
		"executor: unknown job event type '%s'", str)
}

func JobEventTypes() []JobEventType {
	var res []JobEventType
	for typ := jobEventUnknown + 1; typ < jobEventDone; typ++ {
		res = append(res, typ)
	}

	return res
}

//go:generate stringer -type=JobStateType --trimprefix=JobState
type JobStateType int

const (
	jobStateUnknown JobStateType = iota // must be first
	JobStateBidding
	JobStateBidRejected
	JobStateRunning
	JobStateError
	JobStateComplete
	jobStateDone // must be last
)

func ParseJobStateType(str string) (JobStateType, error) {
	for typ := jobStateUnknown + 1; typ < jobStateDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return jobStateUnknown, fmt.Errorf(
		"executor: unknown job event type '%s'", str)
}

func JobStateTypes() []JobStateType {
	var res []JobStateType
	for typ := jobStateUnknown + 1; typ < jobStateDone; typ++ {
		res = append(res, typ)
	}

	return res
}

func equal(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	return strings.EqualFold(a, b)
}
