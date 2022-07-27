package executor

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
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

	// the job was created by a client
	JobEventCreated

	// the concurrency or other mutable properties of the job were
	// changed by the client
	JobEventDealUpdated

	// a compute node bid on a job
	JobEventBid

	// a requester node accepted for rejected a job bid
	JobEventBidAccepted
	JobEventBidRejected

	// a compute node cancled a job bid
	JobEventBidCancelled

	// a compute node is preparing to run a job
	// (e.g. preparing storage volumes and downloading docker images)
	JobEventPreparing

	// a compute node progressed with running a job
	// this is called periodically for running jobs
	// to give the client confidence the job is still running
	// this is like a heartbeat for running jobs
	JobEventRunning

	// a compute node completed running a job
	JobEventCompleted

	// a compute node had an error running a job
	JobEventError

	// a requestor node accepted the results from a node for a job
	JobEventResultsAccepted

	// a requestor node rejected the results from a node for a job
	JobEventResultsRejected

	jobEventDone // must be last
)

// IsTerminal returns true if the given event type signals the end of the
// lifecycle of a job. After this, all nodes can safely ignore the job.
func (event JobEventType) IsTerminal() bool {
	return event == JobEventError || event == JobEventCompleted
}

// IsIgnorable returns true if given event type signals that a node can safely
// ignore the rest of the job's lifecycle. This is the case for events caused
// by a node's bid being rejected.
func (event JobEventType) IsIgnorable() bool {
	return event.IsTerminal() || event == JobEventBidRejected
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
// JobStateType is the state of a job on a particular node. Note that the job
// will typically have different states on different nodes.
type JobStateType int

// these are the states a job can be in against a single node
const (
	jobStateUnknown JobStateType = iota // must be first

	// a compute node has selected a job and has bid on it
	JobStateBidding

	// a requester node has either rejected the bid or the compute node has cancelled the bid
	JobStateCancelled

	// the job is in the process of running
	JobStateRunning

	// the job had an error - this is an end state
	JobStateError

	// the requestor node is verifying the results
	// we got back from the compute node
	JobStateComplete

	// these are 2 end states of a job
	// the requestor node has made a decision about the results
	// submitted by this node
	JobStateResultsAccepted
	JobStateResultsRejected

	jobStateDone // must be last
)

// IsTerminal returns true if the given job type signals the end of the
// lifecycle of that job on a particular node. After this, the job can be
// safely ignored by the node.
<<<<<<< HEAD
func (typ JobStateType) IsTerminal() bool {
	return typ == JobStateComplete || typ == JobStateError || typ == JobStateBidRejected
}

// MarshalYAML encodes a JobStateType as a string for readability.
func (typ JobStateType) MarshalYAML() (interface{}, error) {
	return typ.String(), nil
}

// UnmarshalYAML decodes a JobStateType from a string or an int.
func (typ *JobStateType) UnmarshalYAML(value *yaml.Node) error {
	// First try and parse value.Value as an int:
	i, err := strconv.ParseInt(value.Value, 10, 32) // nolint:gomnd
	if err == nil {
		*typ = JobStateType(i)
		return nil
	}

	// If that fails, try to parse value.Value as a string:
	t, err := ParseJobStateType(value.Value)
	if err != nil {
		return err
	}

	*typ = t
	return nil
||||||| parent of c1290fd7 (move resourceusage package into capacity manager)
func (event JobStateType) IsTerminal() bool {
	return event == JobStateComplete || event == JobStateError || event == JobStateBidRejected
=======
func (state JobStateType) IsTerminal() bool {
	return state == JobStateComplete || state == JobStateError || state == JobStateCancelled
>>>>>>> c1290fd7 (move resourceusage package into capacity manager)
}

func ParseJobStateType(str string) (JobStateType, error) {
	for typ := jobStateUnknown + 1; typ < jobStateDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return jobStateUnknown, fmt.Errorf(
		"executor: unknown job typ type '%s'", str)
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
