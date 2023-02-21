package v1beta1

import "fmt"

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

func ParseJobLocalEventType(str string) (JobLocalEventType, error) {
	for typ := jobLocalEventUnknown + 1; typ < jobLocalEventDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return jobLocalEventDone, fmt.Errorf(
		"executor: unknown job event type '%s'", str)
}

func JobLocalEventTypes() []JobLocalEventType {
	var res []JobLocalEventType
	for typ := jobLocalEventUnknown + 1; typ < jobLocalEventDone; typ++ {
		res = append(res, typ)
	}

	return res
}

func (jle JobLocalEventType) MarshalText() ([]byte, error) {
	return []byte(jle.String()), nil
}

func (jle *JobLocalEventType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*jle, err = ParseJobLocalEventType(name)
	return
}
