package errors

import (
	"fmt"
	"runtime"

	"go.uber.org/zap/zapcore"
)

// ErrorStack is an ordered stacktrace for an error.
type ErrorStack []string

// MarshalLogArray will define the representation of an error stack in logs.
func (s ErrorStack) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	for _, value := range s {
		encoder.AppendString(value)
	}
	return nil
}

// createStack will create a stacktrace for an error.
func createStack() []string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:])
	var callers []uintptr = pcs[0:n]
	frames := runtime.CallersFrames(callers)
	stack := make([]string, 0)

	for {
		frame, more := frames.Next()
		trace := fmt.Sprintf("%s (%d)", frame.Function, frame.Line)
		stack = append(stack, trace)
		if !more {
			return stack
		}
	}
}
