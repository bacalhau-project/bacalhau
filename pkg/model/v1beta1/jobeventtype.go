package v1beta1

import (
	"fmt"
)

//go:generate stringer -type=JobEventType --trimprefix=JobEvent
type JobEventType int

const (
	jobEventUnknown JobEventType = iota // must be first

	// Job has been created by client and is communicating with requestor node
	JobEventInitialSubmission

	// Job has been created on the requestor node
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

	// a compute node had an error running a job
	JobEventComputeError

	// a compute node completed running a job
	JobEventResultsProposed

	// a Requester node accepted the results from a node for a job
	JobEventResultsAccepted

	// a Requester node rejected the results from a node for a job
	JobEventResultsRejected

	// once the results have been accepted or rejected
	// the compute node will publish them and issue this event
	JobEventResultsPublished

	// a requester node declared an error running a job
	JobEventError

	// the requester node gives a compute node permission
	// to forget about the job and free any resources it might
	// currently be reserving - this can happen if a compute node
	// bids when a job has completed - if the compute node does
	// not hear back it will be stuck in reserving the resources for the job
	JobEventInvalidRequest

	jobEventDone // must be last
)

// IsTerminal returns true if the given event type signals the end of the
// lifecycle of a job. After this, all nodes can safely ignore the job.
func (je JobEventType) IsTerminal() bool {
	return je == JobEventError || je == JobEventResultsPublished
}

// IsIgnorable returns true if given event type signals that a node can safely
// ignore the rest of the job's lifecycle. This is the case for events caused
// by a node's bid being rejected.
func (je JobEventType) IsIgnorable() bool {
	return je.IsTerminal() || je == JobEventComputeError || je == JobEventBidRejected || je == JobEventInvalidRequest
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

func (je JobEventType) MarshalText() ([]byte, error) {
	return []byte(je.String()), nil
}

func (je *JobEventType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*je, err = ParseJobEventType(name)
	return
}
