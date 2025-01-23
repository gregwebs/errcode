// Copyright Greg Weber and PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0

package errcode

// HasOperation is an interface to retrieve the operation that occurred during an error.
// The end goal is to be able to see a trace of operations in a distributed system to quickly have a good understanding of what occurred.
// Inspiration is taken from upspin error handling: https://commandcenter.blogspot.com/2017/12/error-handling-in-upspin.html
// The relationship to error codes is not one-to-one.
// A given error code can be triggered by multiple different operations,
// just as a given operation could result in multiple different error codes.
//
// GetOperation is defined, but generally the operation should be retrieved with Operation().
// Operation() will check if a HasOperation interface exists.
// As an alternative to defining this interface
// you can use an existing wrapper (AddOp) or embedding (EmbedOp) that has already defined it.
type HasOperation interface {
	GetOperation() string
}

// Operation will return an operation string if it exists.
// It checks recursively for the HasOperation interface.
// Otherwise it will return the zero value (empty) string.
func Operation(v interface{}) string {
	if hasOp, ok := v.(HasOperation); ok {
		return hasOp.GetOperation()
	}
	if un, ok := v.(unwrapError); ok {
		return Operation(un.Unwrap())
	}
	return ""
}

// EmbedOp is designed to be embedded into your existing error structs.
// It provides the HasOperation interface already, which can reduce your boilerplate.
type EmbedOp struct{ Op string }

// GetOperation satisfies the HasOperation interface
func (e EmbedOp) GetOperation() string {
	return e.Op
}

type OpCode interface {
	ErrorCode
	HasOperation
}

// opErrCode is an ErrorCode with an Operation field attached.
// This can be conveniently constructed with Op() and AddTo() to record the operation information for the error.
// However, it isn't required to be used, see the HasOperation documentation for alternatives.
type opErrCode struct {
	ErrorCode
	EmbedOp
}

// Unwrap satisfies the errors package Unwrap function
func (e opErrCode) Unwrap() error {
	return e.ErrorCode
}

// Error prefixes the operation to the underlying Err Error.
func (e opErrCode) Error() string {
	return e.Op + ": " + e.ErrorCode.Error()
}

var _ ErrorCode = (*opErrCode)(nil)    // assert implements interface
var _ HasOperation = (*opErrCode)(nil) // assert implements interface
var _ OpCode = (*opErrCode)(nil)       // assert implements interface
var _ unwrapError = (*opErrCode)(nil)  // assert implements interface

// AddOp is constructed by Op. It allows method chaining with AddTo.
type AddOp func(ErrorCode) OpCode

// AddTo adds the operation from Op to the ErrorCode
func (addOp AddOp) AddTo(err ErrorCode) OpCode {
	return addOp(err)
}

// Op adds an operation to an ErrorCode with AddTo.
// This converts the error to the type OpCode.
//
//	op := errcode.Op("path.move.x")
//	if start < obstable && obstacle < end  {
//		return op.AddTo(PathBlocked{start, end, obstacle})
//	}
func Op(operation string) AddOp {
	return func(err ErrorCode) OpCode {
		if err == nil {
			panic("Op errorcode is nil")
		}
		return opErrCode{
			ErrorCode: err,
			EmbedOp:   EmbedOp{Op: operation},
		}
	}
}
