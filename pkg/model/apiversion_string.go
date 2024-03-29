// Code generated by "stringer -type=APIVersion"; DO NOT EDIT.

package model

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[apiVersionUnknown-0]
	_ = x[V1alpha1-1]
	_ = x[V1beta1-2]
	_ = x[V1beta2-3]
	_ = x[apiVersionDone-4]
}

const _APIVersion_name = "apiVersionUnknownV1alpha1V1beta1V1beta2apiVersionDone"

var _APIVersion_index = [...]uint8{0, 17, 25, 32, 39, 53}

func (i APIVersion) String() string {
	if i < 0 || i >= APIVersion(len(_APIVersion_index)-1) {
		return "APIVersion(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _APIVersion_name[_APIVersion_index[i]:_APIVersion_index[i+1]]
}
