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

	// a compute node canceled a job bid
	JobEventBidCancelled

	// TODO: what if a requester node accepts a bid
	// and the compute node takes too long to start running it?
	// JobEventBidRevoked

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

//go:generate stringer -type=JobLocalEventType --trimprefix=JobLocalEvent
type JobLocalEventType int

const (
	jobLocalEventUnknown JobLocalEventType = iota // must be first

	JobLocalEventSelected
	JobLocalEventBidAccepted
	JobLocalEventBid

	jobLocalEventDone // must be last
)

//go:generate stringer -type=JobStateType --trimprefix=JobState
// JobStateType is the state of a job on a particular node. Note that the job
// will typically have different states on different nodes.
type JobStateType int

// these are the states a job can be in against a single node
const (
	jobStateUnknown JobStateType = iota // must be first

	// a compute node has selected a job and has bid on it
	// we are currently waiting to hear back from the requester
	// node whether our bid was accepted or not
	JobStateBidding

	// a requester node has either rejected the bid or the compute node has canceled the bid
	// either way - this node will not progress with this job any more
	JobStateCancelled

	// the bid has been accepted but we have not yet started the job
	JobStateWaiting

	// the job is in the process of running
	JobStateRunning

	// the job had an error - this is an end state
	JobStateError

	// the requestor node is verifying the results
	// we got back from the compute node
	JobStateComplete

	// our results have been processed
	JobStateFinalized

	jobStateDone // must be last
)

// IsTerminal returns true if the given job type signals the end of the
// lifecycle of that job on a particular node. After this, the job can be
// safely ignored by the node.
func (state JobStateType) IsTerminal() bool {
	return state == JobStateComplete || state == JobStateError || state == JobStateCancelled
}

// IsComplete returns true if the given job has succeeded at the bid stage
// and has finished running the job - this is used to calculate if a job
// has completed across all nodes because a cancelation does not count
// towards actually "running" the job whereas an error does (even though it failed
// it still "ran")
func (state JobStateType) IsComplete() bool {
	return state == JobStateComplete || state == JobStateError
}

func (state JobStateType) IsError() bool {
	return state == JobStateError
}

// tells you if this event is a valid one
func IsValidJobState(state JobStateType) bool {
	return state > jobStateUnknown && state < jobStateDone
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

// given an event name - return a job state
func GetStateFromEvent(eventType JobEventType) JobStateType {
	switch eventType {
	// we have bid and are waiting to hear if that has been accepted
	case JobEventBid:
		return JobStateBidding

	// our bid has been accepted but we've not yet started the job
	case JobEventBidAccepted:
		return JobStateWaiting

	// out bid got rejected so we are canceled
	case JobEventBidRejected:
		return JobStateCancelled

	// we canceled our bid so we are canceled
	case JobEventBidCancelled:
		return JobStateCancelled

	// we are running
	case JobEventRunning:
		return JobStateRunning

	// we are complete
	case JobEventCompleted:
		return JobStateComplete

	// we are complete
	case JobEventError:
		return JobStateError

	// both of these are "finalized"
	case JobEventResultsAccepted:
		return JobStateFinalized

	case JobEventResultsRejected:
		return JobStateFinalized

	default:
		return jobStateUnknown
	}
}
