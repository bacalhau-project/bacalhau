// Code generated by "stringer -type=Network --trimprefix=Network"; DO NOT EDIT.

package models

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[NetworkNone-0]
	_ = x[NetworkFull-1]
	_ = x[NetworkHTTP-2]
}

const _Network_name = "NoneFullHTTP"

var _Network_index = [...]uint8{0, 4, 8, 12}

func (i Network) String() string {
	if i < 0 || i >= Network(len(_Network_index)-1) {
		return "Network(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Network_name[_Network_index[i]:_Network_index[i+1]]
}
