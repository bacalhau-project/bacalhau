package model

import (
	"fmt"
	"time"
)

//go:generate stringer -type=JobEventType --trimprefix=JobEvent --output job_event_string.go
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

	// a user canceled a job
	JobEventCanceled

	// the requester node gives a compute node permission
	// to forget about the job and free any resources it might
	// currently be reserving - this can happen if a compute node
	// bids when a job has completed - if the compute node does
	// not hear back it will be stuck in reserving the resources for the job
	JobEventInvalidRequest

	// a job has been completed
	JobEventCompleted

	jobEventDone // must be last
)

// IsTerminal returns true if the given event type signals the end of the
// lifecycle of a job. After this, all nodes can safely ignore the job.
func (je JobEventType) IsTerminal() bool {
	return je == JobEventError || je == JobEventCompleted || je == JobEventCanceled
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

type JobEvent struct {
	JobID string `json:"JobID,omitempty" example:"9304c616-291f-41ad-b862-54e133c0149e"`
	// compute execution identifier
	ExecutionID string `json:"ExecutionID,omitempty" example:"9304c616-291f-41ad-b862-54e133c0149e"`
	// the node that emitted this event
	SourceNodeID string `json:"SourceNodeID,omitempty" example:"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF"`
	// the node that this event is for
	// e.g. "AcceptJobBid" was emitted by Requester but it targeting compute node
	TargetNodeID string `json:"TargetNodeID,omitempty" example:"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL"`

	EventName JobEventType `json:"EventName,omitempty"`
	Status    string       `json:"Status,omitempty" example:"Got results proposal of length: 0"`

	EventTime time.Time `json:"EventTime,omitempty" example:"2022-11-17T13:32:55.756658941Z"`
}
