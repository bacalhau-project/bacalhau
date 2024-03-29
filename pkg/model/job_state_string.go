// Code generated by "stringer -type=JobStateType --trimprefix=JobState --output job_state_string.go"; DO NOT EDIT.

package model

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[JobStateUndefined-0]
	_ = x[JobStateNew-1]
	_ = x[JobStateInProgress-2]
	_ = x[JobStateCancelled-3]
	_ = x[JobStateError-4]
	_ = x[JobStateCompleted-5]
	_ = x[JobStateQueued-6]
}

const _JobStateType_name = "UndefinedNewInProgressCancelledErrorCompletedQueued"

var _JobStateType_index = [...]uint8{0, 9, 12, 22, 31, 36, 45, 51}

func (i JobStateType) String() string {
	if i < 0 || i >= JobStateType(len(_JobStateType_index)-1) {
		return "JobStateType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _JobStateType_name[_JobStateType_index[i]:_JobStateType_index[i+1]]
}
