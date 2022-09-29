package model

import "fmt"

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

	// a requester node has either rejected the bid or the compute node has canceled the bid
	// either way - this node will not progress with this job any more
	JobStateCancelled

	// the bid has been accepted but we have not yet started the job
	JobStateWaiting

	// the job is in the process of running
	JobStateRunning

	// the job had an error - this is an end state
	JobStateError

	// the compute node has finished execution and has communicated the ResultsProposal
	JobStateVerifying

	// our results have been processed and published
	JobStateCompleted

	jobStateDone // must be last
)

// IsTerminal returns true if the given job type signals the end of the
// lifecycle of that job on a particular node. After this, the job can be
// safely ignored by the node.
func (state JobStateType) IsTerminal() bool {
	return state == JobStateCompleted || state == JobStateError || state == JobStateCancelled
}

// IsComplete returns true if the given job has succeeded at the bid stage
// and has finished running the job - this is used to calculate if a job
// has completed across all nodes because a cancelation does not count
// towards actually "running" the job whereas an error does (even though it failed
// it still "ran")
func (state JobStateType) IsComplete() bool {
	return state == JobStateCompleted || state == JobStateError
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
	case JobEventError:
		return JobStateError

	// we are complete
	case JobEventResultsProposed:
		return JobStateVerifying

	// both of these are "finalized"
	case JobEventResultsAccepted:
		return JobStateVerifying

	case JobEventResultsRejected:
		return JobStateVerifying

	case JobEventResultsPublished:
		return JobStateCompleted

	default:
		return jobStateUnknown
	}
}
