// Code generated by "stringer -type=JobEventType --trimprefix=JobEvent"; DO NOT EDIT.

package model

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[jobEventUnknown-0]
	_ = x[JobEventCreated-1]
	_ = x[JobEventDealUpdated-2]
	_ = x[JobEventBid-3]
	_ = x[JobEventBidAccepted-4]
	_ = x[JobEventBidRejected-5]
	_ = x[JobEventBidCancelled-6]
	_ = x[JobEventRunning-7]
	_ = x[JobEventComputeError-8]
	_ = x[JobEventResultsProposed-9]
	_ = x[JobEventResultsAccepted-10]
	_ = x[JobEventResultsRejected-11]
	_ = x[JobEventResultsPublished-12]
	_ = x[JobEventError-13]
	_ = x[jobEventDone-14]
}

const _JobEventType_name = "jobEventUnknownCreatedDealUpdatedBidBidAcceptedBidRejectedBidCancelledRunningComputeErrorResultsProposedResultsAcceptedResultsRejectedResultsPublishedErrorjobEventDone"

var _JobEventType_index = [...]uint8{0, 15, 22, 33, 36, 47, 58, 70, 77, 89, 104, 119, 134, 150, 155, 167}

func (i JobEventType) String() string {
	if i < 0 || i >= JobEventType(len(_JobEventType_index)-1) {
		return "JobEventType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _JobEventType_name[_JobEventType_index[i]:_JobEventType_index[i+1]]
}
