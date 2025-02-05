// Copyright Greg Weber and PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0

package errcode

import (
	"github.com/gregwebs/errors"
	"github.com/gregwebs/stackfmt"
)

// StackCode is an [ErrorCode] with stack trace information attached.
// This may be used as a convenience to record the strack trace information for the error.
// Stack traces are provided by [NewInternalErr].
// Its also possible to define your own structures that satisfy the [errors.StackTracer] interface.
type StackCode struct {
	ErrorCode
	GetStack stackfmt.StackTracer
}

// StackTrace fulfills the [errors.StackTracer] interface
func (e StackCode) StackTrace() stackfmt.StackTrace {
	return e.GetStack.StackTrace()
}

// NewStackCode constructs a [StackCode], which is an [ErrorCode] with stack trace information.
// The second variable is an optional stack position that gets rid of information about function calls to construct the stack trace.
// It is defaulted to 1 to remove this function call.
//
// NewStackCode first looks at the underlying error chain to see if it already has an [errors.StackTracer].
// If so, that StackTrace is used.
func NewStackCode(err ErrorCode, position ...int) StackCode {
	if err == nil {
		panic("NewStackCode: given error is nil")
	}

	// if there is an existing trace, take that: it should be deeper
	if tracer := errors.GetStackTracer(err); tracer != nil {
		return StackCode{ErrorCode: err, GetStack: tracer}
	}

	stackPosition := 1
	if len(position) > 0 {
		stackPosition = position[0]
	}
	return StackCode{ErrorCode: err, GetStack: stackfmt.NewStackSkip(stackPosition)}
}

// Unwrap satisfies the errors package Unwrap function
func (e StackCode) Unwrap() error {
	return e.ErrorCode
}

// Error ignores the stack and gives the underlying Err Error.
func (e StackCode) Error() string {
	return e.ErrorCode.Error()
}

var _ ErrorCode = (*StackCode)(nil)   // assert implements interface
var _ unwrapError = (*StackCode)(nil) // assert implements interface
