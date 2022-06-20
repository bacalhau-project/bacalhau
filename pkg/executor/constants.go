package executor

import (
	"fmt"
	"strings"
)

//go:generate stringer -type=Type --trimprefix=Type
type Type int

const (
	typeUnknown Type = iota // must be first
	TypeNoop
	TypeDocker
	TypeWasm
	typeDone // must be last
)

func ParseType(str string) (Type, error) {
	for typ := typeUnknown + 1; typ < typeDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return typeUnknown, fmt.Errorf("executor: unknown type '%s'", str)
}

func Types() []Type {
	var res []Type
	for typ := typeUnknown + 1; typ < typeDone; typ++ {
		res = append(res, typ)
	}

	return res
}

func equal(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	return strings.EqualFold(a, b)
}

type JobEventType string

// event names - i.e. "this just happened"
const (
	JOB_EVENT_CREATED          JobEventType = "job_created"
	JOB_EVENT_DEAL_UPDATED     JobEventType = "deal_updated"
	JOB_EVENT_BID              JobEventType = "bid"
	JOB_EVENT_BID_ACCEPTED     JobEventType = "bid_accepted"
	JOB_EVENT_BID_REJECTED     JobEventType = "bid_rejected"
	JOB_EVENT_RESULTS          JobEventType = "results"
	JOB_EVENT_RESULTS_ACCEPTED JobEventType = "results_accepted"
	JOB_EVENT_RESULTS_REJECTED JobEventType = "results_rejected"
	JOB_EVENT_ERROR            JobEventType = "error"
)

type JobStateType string

// job states - these will be collected per host against a job
const (
	JOB_STATE_BIDDING      JobStateType = "bidding"
	JOB_STATE_BID_REJECTED JobStateType = "bid_rejected"
	JOB_STATE_RUNNING      JobStateType = "running"
	JOB_STATE_ERROR        JobStateType = "error"
	JOB_STATE_COMPLETE     JobStateType = "complete"
)
