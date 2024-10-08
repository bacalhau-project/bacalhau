package bacerrors

import (
	"fmt"
	"runtime"
	"strings"
)

// stack represents a stack of program counters.
type stack []uintptr

func (s *stack) String() string {
	var builder strings.Builder
	frames := runtime.CallersFrames(*s)
	for {
		frame, more := frames.Next()
		if frame.Function != "" {
			fmt.Fprintf(&builder, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		}
		if !more {
			break
		}
	}
	return builder.String()
}

func callers() *stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(4, pcs[:])
	var st stack = pcs[0:n]
	return &st
}
