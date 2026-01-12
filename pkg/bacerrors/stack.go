package bacerrors

import (
	"fmt"
	"runtime"
	"strings"
)

const (
	// maxStackDepth defines the maximum number of stack frames to capture
	// in the error stack trace.
	maxStackDepth = 32
)

// stack represents a stack of program counters.
type stack []uintptr

func (s *stack) String() string {
	var builder strings.Builder
	frames := runtime.CallersFrames(*s)
	for {
		frame, more := frames.Next()
		if frame.Function != "" {
			_, _ = fmt.Fprintf(&builder, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		}
		if !more {
			break
		}
	}
	return builder.String()
}

func callers() *stack {
	const skipCallers = 4
	var pcs [maxStackDepth]uintptr
	n := runtime.Callers(skipCallers, pcs[:])
	var st stack = pcs[0:n]
	return &st
}
