// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

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
// you can use an existing wrapper (OpErrCode via AddOp) or embedding (EmbedOp) that has already defined it.
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

// OpErrCode is an ErrorCode with an Operation field attached.
// This can be conveniently constructed with Op() and AddTo() to record the operation information for the error.
// However, it isn't required to be used, see the HasOperation documentation for alternatives.
type OpErrCode struct {
	ErrorCode
	EmbedOp
}

// Unwrap satisfies the errors package Unwrap function
func (e OpErrCode) Unwrap() error {
	return e.ErrorCode
}

// Error prefixes the operation to the underlying Err Error.
func (e OpErrCode) Error() string {
	return e.Op + ": " + e.ErrorCode.Error()
}

var _ ErrorCode = (*OpErrCode)(nil)    // assert implements interface
var _ HasOperation = (*OpErrCode)(nil) // assert implements interface
var _ OpCode = (*OpErrCode)(nil)       // assert implements interface
var _ unwrapError = (*OpErrCode)(nil)  // assert implements interface

// AddOp is constructed by Op. It allows method chaining with AddTo.
type AddOp func(ErrorCode) OpErrCode

// AddTo adds the operation from Op to the ErrorCode
func (addOp AddOp) AddTo(err ErrorCode) OpErrCode {
	return addOp(err)
}

// Op adds an operation to an ErrorCode with AddTo.
// This converts the error to the type OpErrCode.
//
//	op := errcode.Op("path.move.x")
//	if start < obstable && obstacle < end  {
//		return op.AddTo(PathBlocked{start, end, obstacle})
//	}
func Op(operation string) AddOp {
	return func(err ErrorCode) OpErrCode {
		if err == nil {
			panic("Op errorcode is nil")
		}
		return OpErrCode{
			ErrorCode: err,
			EmbedOp:   EmbedOp{Op: operation},
		}
	}
}
