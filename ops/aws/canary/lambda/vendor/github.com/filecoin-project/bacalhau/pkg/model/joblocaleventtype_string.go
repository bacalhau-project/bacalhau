// Code generated by "stringer -type=JobLocalEventType --trimprefix=JobLocalEvent"; DO NOT EDIT.

package model

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[jobLocalEventUnknown-0]
	_ = x[JobLocalEventSelected-1]
	_ = x[JobLocalEventBid-2]
	_ = x[JobLocalEventBidAccepted-3]
	_ = x[JobLocalEventBidRejected-4]
	_ = x[JobLocalEventVerified-5]
	_ = x[jobLocalEventDone-6]
}

const _JobLocalEventType_name = "jobLocalEventUnknownSelectedBidBidAcceptedBidRejectedVerifiedjobLocalEventDone"

var _JobLocalEventType_index = [...]uint8{0, 20, 28, 31, 42, 53, 61, 78}

func (i JobLocalEventType) String() string {
	if i < 0 || i >= JobLocalEventType(len(_JobLocalEventType_index)-1) {
		return "JobLocalEventType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _JobLocalEventType_name[_JobLocalEventType_index[i]:_JobLocalEventType_index[i+1]]
}
