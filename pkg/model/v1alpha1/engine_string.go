// Code generated by "stringer -type=Engine --trimprefix=Engine"; DO NOT EDIT.

package v1alpha1

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[engineUnknown-0]
	_ = x[EngineNoop-1]
	_ = x[EngineDocker-2]
	_ = x[EngineWasm-3]
	_ = x[EngineLanguage-4]
	_ = x[EnginePythonWasm-5]
	_ = x[engineDone-6]
}

const _Engine_name = "engineUnknownNoopDockerWasmLanguagePythonWasmengineDone"

var _Engine_index = [...]uint8{0, 13, 17, 23, 27, 35, 45, 55}

func (i Engine) String() string {
	if i < 0 || i >= Engine(len(_Engine_index)-1) {
		return "Engine(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Engine_name[_Engine_index[i]:_Engine_index[i+1]]
}
