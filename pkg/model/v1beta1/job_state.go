package v1beta1

import (
	"fmt"
)

// JobStateType is the state of a job on a particular node. Note that the job
// will typically have different states on different nodes.
//
//go:generate stringer -type=JobStateType --trimprefix=JobState
type JobStateType int

// these are the states a job can be in against a single node
const (
	jobStateUnknown JobStateType = iota // must be first

	// a compute node has selected a job and has bid on it
	// we are currently waiting to hear back from the requester
	// node whether our bid was accepted or not
	JobStateBidding

	// the bid has been accepted but we have not yet started the job
	JobStateWaiting

	// the job is in the process of running
	JobStateRunning

	// the compute node has finished execution and has communicated the ResultsProposal
	JobStateVerifying

	// a requester node has either rejected the bid or the compute node has canceled the bid
	// either way - this node will not progress with this job any more
	JobStateCancelled

	// the job had an error - this is an end state
	JobStateError

	// our results have been processed and published
	JobStateCompleted

	jobStateDone // must be last
)

// IsTerminal returns true if the given job type signals the end of the
// lifecycle of that job on a particular node. After this, the job can be
// safely ignored by the node.
func (s JobStateType) IsTerminal() bool {
	return s == JobStateCompleted || s == JobStateError || s == JobStateCancelled
}

// IsComplete returns true if the given job has succeeded at the bid stage
// and has finished running the job - this is used to calculate if a job
// has completed across all nodes because a cancelation does not count
// towards actually "running" the job whereas an error does (even though it failed
// it still "ran")
func (s JobStateType) IsComplete() bool {
	return s == JobStateCompleted || s == JobStateError
}

func (s JobStateType) HasPassedBidAcceptedStage() bool {
	return s == JobStateWaiting || s == JobStateRunning || s == JobStateVerifying || s == JobStateError || s == JobStateCompleted
}

func (s JobStateType) IsError() bool {
	return s == JobStateError
}

// tells you if this event is a valid one
func IsValidJobState(s JobStateType) bool {
	return s > jobStateUnknown && s < jobStateDone
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

func JobStateTypeNames() []string {
	var names []string
	for _, typ := range JobStateTypes() {
		names = append(names, typ.String())
	}
	return names
}

func (s JobStateType) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *JobStateType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*s, err = ParseJobStateType(name)
	return
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

	// yikes
	case JobEventError, JobEventComputeError, JobEventInvalidRequest:
		return JobStateError

	// we are complete
	case JobEventResultsProposed:
		return JobStateVerifying

	// both of these are "finalized"
	case JobEventResultsAccepted:
		return JobStateVerifying

	case JobEventResultsRejected:
		return JobStateError

	case JobEventResultsPublished:
		return JobStateCompleted

	default:
		return jobStateUnknown
	}
}
