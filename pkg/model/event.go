package model

import (
	"fmt"
)

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

	// a compute node had an error running a job
	JobEventError

	// a compute node completed running a job
	JobEventResultsProposed

	// a requestor node accepted the results from a node for a job
	JobEventResultsAccepted

	// a requestor node rejected the results from a node for a job
	JobEventResultsRejected

	// once the results have been accepted or rejected
	// the compute node will publish them and issue this event
	JobEventResultsPublished

	jobEventDone // must be last
)

// IsTerminal returns true if the given event type signals the end of the
// lifecycle of a job. After this, all nodes can safely ignore the job.
func (event JobEventType) IsTerminal() bool {
	return event == JobEventError || event == JobEventResultsPublished
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

	// compute node
	// this means "we have selected this job"
	// used to avoid calling external selection hooks
	// where capacity manager says we can't quite run
	// the job yet but we will want to bid when there
	// is space
	JobLocalEventSelected
	// compute node
	// this means "we have bid" on a job where "we"
	// is the compute node
	JobLocalEventBid
	// requester node
	// used to avoid race conditions with the requester
	// node knowing which bids it's already responded to
	JobLocalEventBidAccepted
	JobLocalEventBidRejected

	// requester node
	// flag a job as having already had it's verification done
	JobLocalEventVerified

	jobLocalEventDone // must be last
)
