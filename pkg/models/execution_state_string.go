// Code generated by "stringer -type=ExecutionStateType --trimprefix=ExecutionState --output execution_state_string.go"; DO NOT EDIT.

package models

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ExecutionStateUndefined-0]
	_ = x[ExecutionStateNew-1]
	_ = x[ExecutionStateAskForBid-2]
	_ = x[ExecutionStateAskForBidAccepted-3]
	_ = x[ExecutionStateAskForBidRejected-4]
	_ = x[ExecutionStateBidAccepted-5]
	_ = x[ExecutionStateBidRejected-6]
	_ = x[ExecutionStateCompleted-7]
	_ = x[ExecutionStateFailed-8]
	_ = x[ExecutionStateCancelled-9]
}

const _ExecutionStateType_name = "UndefinedNewAskForBidAskForBidAcceptedAskForBidRejectedBidAcceptedBidRejectedCompletedFailedCancelled"

var _ExecutionStateType_index = [...]uint8{0, 9, 12, 21, 38, 55, 66, 77, 86, 92, 101}

func (i ExecutionStateType) String() string {
	if i < 0 || i >= ExecutionStateType(len(_ExecutionStateType_index)-1) {
		return "ExecutionStateType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ExecutionStateType_name[_ExecutionStateType_index[i]:_ExecutionStateType_index[i+1]]
}
