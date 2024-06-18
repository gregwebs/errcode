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

// Package errcode facilitates standardized API error codes.
// The goal is that clients can reliably understand errors by checking against immutable error codes
//
// This godoc documents usage. For broader context, see https://github.com/gregwebs/errcode/tree/master/README.md
//
// Error codes are represented as strings by CodeStr (see CodeStr documentation).
//
// This package is designed to have few opinions and be a starting point for how you want to do errors in your project.
// The main requirement is to satisfy the ErrorCode interface by attaching a Code to an Error.
// See the documentation of ErrorCode.
// Additional optional interfaces HasClientData, HasOperation, StackTracer, and the Unwrap method are used for extensibility
// in creating structured error data representations.
//
// Hierarchies are supported: a Code can point to a parent.
// This is used in the HTTPCode implementation to inherit HTTP codes found with MetaDataFromAncestors.
// The hierarchy is present in the Code's string representation with a dot separation.
//
// A few generic top-level error codes are provided (see the variables section of the doc).
// You are encouraged to create your own error codes customized to your application rather than solely using generic errors.
//
// See NewJSONFormat for an opinion on how to send back meta data about errors with the error data to a client.
// JSONFormat includes a body of response data (the "data field") that is by default the data from the Error
// serialized to JSON.
//
// Stack traces are automatically added by NewInternalErr and show up as the Stack field in JSONFormat.
// Errors can be grouped with Combine() and ungrouped via Errors() which show up as the Others field in JSONFormat.
//
// To extract any ErrorCodes from an error, use CodeChain().
// This extracts error codes without information loss (using ChainContext).
package errcode

import (
	"fmt"
	"strings"

	"github.com/gregwebs/errors"
)

// CodeStr is the name of the error code.
// It is a representation of the type of a particular error.
// The underlying type is string rather than int.
// This enhances both extensibility (avoids merge conflicts) and user-friendliness.
// A CodeStr can have dot separators indicating a hierarchy.
//
// Generally a CodeStr should never be modified once used by clients.
// Instead a new CodeStr should be created.
type CodeStr string

func (str CodeStr) String() string { return string(str) }

// A Code has a CodeStr representation.
// It is attached to a Parent to find metadata from it.
type Code struct {
	// codeStr does not include parent paths
	// The full code (with parent paths) is accessed with CodeStr
	codeStr CodeStr
	Parent  *Code
}

// CodeStr gives the full dot-separted path.
// This is what should be used for equality comparison.
func (code Code) CodeStr() CodeStr {
	if code.Parent == nil {
		return code.codeStr
	}
	return (*code.Parent).CodeStr() + "." + code.codeStr
}

// NewCode creates a new top-level code.
// A top-level code must not contain any dot separators: that will panic
// Most codes should be created from hierachry with the Child method.
func NewCode(codeRep CodeStr) Code {
	code := Code{codeStr: codeRep}
	if err := code.checkCodePath(); err != nil {
		panic(err)
	}
	return code
}

// Child creates a new code from a parent.
// For documentation purposes, a childStr may include the parent codes with dot-separation.
// An incorrect parent reference in the string panics.
func (code Code) Child(childStr CodeStr) Code {
	child := Code{codeStr: childStr, Parent: &code}
	if err := child.checkCodePath(); err != nil {
		panic(err)
	}
	// Don't store parent paths, those are re-constructed in CodeStr()
	paths := strings.Split(child.codeStr.String(), ".")
	child.codeStr = CodeStr(paths[len(paths)-1])
	return child
}

// FindAncestor looks for an ancestor satisfying the given test function.
func (code Code) findAncestor(test func(Code) bool) *Code {
	if test(code) {
		return &code
	}
	if code.Parent == nil {
		return nil
	}
	return (*code.Parent).findAncestor(test)
}

// IsAncestor looks for the given code in its ancestors.
func (code Code) IsAncestor(ancestorCode Code) bool {
	return nil != code.findAncestor(func(an Code) bool { return an == ancestorCode })
}

// ErrorCode is the interface that ties an error and RegisteredCode together.
//
// Note that there are additional interfaces (HasClientData, HasOperation, please see the docs)
// that can be defined by an ErrorCode to customize finding structured data for the client.
//
// ErrorCode allows error codes to be defined
// without being forced to use a particular struct such as CodedError.
// CodedError is convenient for generic errors that wrap many different errors with similar codes.
// Please see the docs for CodedError.
// For an application specific error with a 1:1 mapping between a go error structure and a RegisteredCode,
// You probably want to use this interface directly. Example:
//
//	// First define a normal error type
//	type PathBlocked struct {
//		start     uint64 `json:"start"`
//		end       uint64 `json:"end"`
//		obstacle  uint64 `json:"end"`
//	}
//
//	func (e PathBlocked) Error() string {
//		return fmt.Sprintf("The path %d -> %d has obstacle %d", e.start, e.end, e.obstacle)
//	}
//
//	// Now define the code
//	var PathBlockedCode = errcode.StateCode.Child("state.blocked")
//
//	// Now attach the code to the error type
//	func (e PathBlocked) Code() Code {
//		return PathBlockedCode
//	}
type ErrorCode interface {
	Error() string // The Error interface
	Code() Code
}

// unwrapper allows the abstract retrieval of the underlying error.
// Formalize the Unwrap interface, but don't export it.
// The standard library errors package should export it.
// Types that wrap errors should implement this to allow viewing of the underlying error.
type unwrapper interface {
	Unwrap() error
}

// wrappedErrorCode is a convenience to maintain the ErrorCode type when wrapping errors
type wrappedErrorCode struct {
	Err       error
	ErrorCode ErrorCode
}

// Code fulfills the ErrorCode interface
func (wrapped wrappedErrorCode) Code() Code {
	return wrapped.ErrorCode.Code()
}

// Error fulfills the ErrorCode interface
func (wrapped wrappedErrorCode) Error() string {
	return wrapped.Err.Error()
}

// Allow unwrapping
func (wrapped wrappedErrorCode) Unwrap() error {
	return wrapped.Err
}

// Wrap is a convenience that calls errors.Wrap but still returns the ErrorCode interface
func Wrap(errCode ErrorCode, msg string) ErrorCode {
	return wrappedErrorCode{
		Err:       errors.Wrap(errCode, msg),
		ErrorCode: errCode,
	}
}

// Wrapf is a convenience that calls errors.Wrapf but still returns the ErrorCode interface
func Wrapf(errCode ErrorCode, msg string, args ...interface{}) ErrorCode {
	return wrappedErrorCode{
		Err:       errors.Wrapf(errCode, msg, args...),
		ErrorCode: errCode,
	}
}

// Wraps is a convenience that calls errors.Wraps but still returns the ErrorCode interface
func Wraps(errCode ErrorCode, msg string, args ...interface{}) ErrorCode {
	return wrappedErrorCode{
		Err:       errors.Wraps(errCode, msg, args...),
		ErrorCode: errCode,
	}
}

// HasClientData is used to defined how to retrieve the data portion of an ErrorCode to be returned to the client.
// Otherwise the struct itself will be assumed to be all the data by the ClientData method.
// This is provided for exensibility, but may be unnecessary for you.
// Data should be retrieved with the ClientData method.
type HasClientData interface {
	GetClientData() interface{}
}

// ClientData retrieves data from a structure that implements HasClientData
// It will unwrap errors to look for HasClientData
// If HasClientData is not defined it uses the given ErrorCode object.
// Normally this function is used rather than GetClientData.
func ClientData(errCode ErrorCode) interface{} {
	if hasData, ok := errCode.(HasClientData); ok {
		return hasData.GetClientData()
	}
	var err error = errCode
	for {
		if un, ok := err.(unwrapper); ok {
			err = un.Unwrap()
			if hasData, ok := err.(HasClientData); ok {
				return hasData.GetClientData()
			}
		} else {
			break
		}
	}
	return errCode
}

// JSONFormat serializes an ErrorCode to a particular JSON format.
// You can write your own version of this that matches your needs along with your own constructor function.
//
// * Code is the error code string (CodeStr)
// * Msg is the string from Error() and should be friendly to end users.
// * Data is the ad-hoc data filled in by GetClientData and should be consumable by clients.
// * Operation is the high-level operation that was happening at the time of the error.
// The Operation field may be missing, and the Data field may be empty.
//
// The rest of the fields may be populated sparsely depending on the application:
// * Stack is a stack trace. This is only given for internal errors.
// * Others gives other errors that occurred (perhaps due to parallel requests).
type JSONFormat struct {
	Code      CodeStr      `json:"code"`
	Msg       string       `json:"msg"`
	Data      interface{}  `json:"data"`
	Operation string       `json:"operation,omitempty"`
	Others    []JSONFormat `json:"others,omitempty"`
}

// OperationClientData gives the results of both the ClientData and Operation functions.
// The Operation function is applied to the original ErrorCode.
// If that does not return an operation, it is applied to the result of ClientData.
// This function is used by NewJSONFormat to fill JSONFormat.
func OperationClientData(errCode ErrorCode) (string, interface{}) {
	op := Operation(errCode)
	data := ClientData(errCode)
	if op == "" {
		op = Operation(data)
	}
	return op, data
}

// NewJSONFormat turns an ErrorCode into a JSONFormat.
// You can create your own json struct and write your own version of this function.
func NewJSONFormat(errCode ErrorCode) JSONFormat {
	// Gather up multiple errors.
	// We discard any that are not ErrorCode.
	errorCodes := ErrorCodes(errCode)[1:]
	others := make([]JSONFormat, len(errorCodes))
	for i, err := range errorCodes {
		others[i] = NewJSONFormat(err)
	}

	op, data := OperationClientData(errCode)

	msg := GetUserMsg(errCode)
	if msg == "" {
		msg = errCode.Error()
	}

	return JSONFormat{
		Data:      data,
		Msg:       msg,
		Code:      errCode.Code().CodeStr(),
		Operation: op,
		Others:    others,
	}
}

// checkCodePath checks that the given code string either
// contains no dots or extends the parent code string
func (code Code) checkCodePath() error {
	paths := strings.Split(code.codeStr.String(), ".")
	if len(paths) == 1 {
		return nil
	}
	if code.Parent == nil {
		if len(paths) > 1 {
			return fmt.Errorf("expected no parent paths: %#v", code.codeStr)
		}
	} else {
		parent := *code.Parent
		parentPath := paths[len(paths)-2]
		if parentPath != parent.codeStr.String() {
			return fmt.Errorf("got %#v but expected a path to parent %#v for %#v", parentPath, parent.codeStr, code.codeStr)
		}
	}
	return nil
}
