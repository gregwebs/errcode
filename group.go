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

import (
	"fmt"

	"github.com/gregwebs/errors"
)

// ErrorCodes return all errors (from an ErrorGroup) that are of interface ErrorCode.
// It first calls the Errors function.
func ErrorCodes(err error) []ErrorCode {
	errors := errors.Errors(err)
	errorCodes := make([]ErrorCode, 0, len(errors))
	for _, errItem := range errors {
		if errcode, ok := errItem.(ErrorCode); ok {
			errorCodes = append(errorCodes, errcode)
		}
	}
	return errorCodes
}

// A MultiErrCode contains at least one ErrorCode and uses that to satisfy the ErrorCode and related interfaces
// The Error method will produce a string of all the errors with a semi-colon separation.
// Later code (such as a JSON response) needs to look for the ErrorGroup interface.
type MultiErrCode struct {
	ErrCode ErrorCode
	rest    []error
}

// Combine constructs a MultiErrCode.
// It will combine any other MultiErrCode into just one MultiErrCode.
// This is "horizontal" composition.
// If you want normal "vertical" composition use BuildChain.
func Combine(initial ErrorCode, others ...ErrorCode) MultiErrCode {
	var rest []error
	if group, ok := initial.(errors.ErrorGroup); ok {
		rest = group.Errors()
	}
	for _, other := range others {
		rest = append(rest, errors.Errors(other)...)
	}
	return MultiErrCode{
		ErrCode: initial,
		rest:    rest,
	}
}

var _ ErrorCode = (*MultiErrCode)(nil)         // assert implements interface
var _ HasClientData = (*MultiErrCode)(nil)     // assert implements interface
var _ unwrapper = (*MultiErrCode)(nil)         // assert implements interface
var _ errors.ErrorGroup = (*MultiErrCode)(nil) // assert implements interface
var _ fmt.Formatter = (*MultiErrCode)(nil)     // assert implements interface

func (e MultiErrCode) Error() string {
	output := e.ErrCode.Error()
	for _, item := range e.rest {
		output += "; " + item.Error()
	}
	return output
}

// Errors fullfills the ErrorGroup inteface
func (e MultiErrCode) Errors() []error {
	return append([]error{e.ErrCode.(error)}, e.rest...)
}

// Code fullfills the ErrorCode inteface
func (e MultiErrCode) Code() Code {
	return e.ErrCode.Code()
}

// Unwrap fullfills the errors package Unwrap function
func (e MultiErrCode) Unwrap() error {
	return e.ErrCode
}

// GetClientData fullfills the HasClientData inteface
func (e MultiErrCode) GetClientData() interface{} {
	return ClientData(e.ErrCode)
}

// CodeChain resolves wrapped errors down to the first ErrorCode.
// An error that is an ErrorGroup with multiple codes will have its error codes combined to a MultiErrCode.
// If the given error is not an ErrorCode, a ContextChain will be returned with Top set to the given error.
// This allows the return object to maintain a full Error() message.
func CodeChain(errInput error) ErrorCode {
	checkError := func(err error) ErrorCode {
		if errCode, ok := err.(ErrorCode); ok {
			return errCode
		} else if eg, ok := err.(errors.ErrorGroup); ok {
			group := []ErrorCode{}
			for _, errItem := range eg.Errors() {
				if itemCode := CodeChain(errItem); itemCode != nil {
					group = append(group, itemCode)
				}
			}
			if len(group) > 0 {
				if len(group) == 1 {
					return group[0]
				} else {
					return Combine(group[0], group[1:]...)
				}
			}
		}
		return nil
	}

	// In this case there is no need for ChainContext
	if errCode, ok := errInput.(ErrorCode); ok {
		return errCode
	}

	err := errInput
	for err != nil {
		if errCode := checkError(err); errCode != nil {
			return ChainContext{errInput, errCode}
		}
		err = errors.Unwrap(err)
	}

	return nil
}

// ChainContext is returned by ErrorCodeChain
// to retain the full wrapped error message of the error chain.
// If you annotated an ErrorCode with additional information, it is retained in the Top field.
// The Top field is used for the Error() and Unwrap() methods.
type ChainContext struct {
	Top     error
	ErrCode ErrorCode
}

// Code satisfies the ErrorCode interface
func (err ChainContext) Code() Code {
	return err.ErrCode.Code()
}

// Error satisfies the Error interface
func (err ChainContext) Error() string {
	return err.Top.Error()
}

// Unwrap satisfies the errors package Unwrap function
func (err ChainContext) Unwrap() error {
	if wrapped := errors.Unwrap(err.Top); wrapped != nil {
		return wrapped
	}
	return err.ErrCode
}

// GetClientData satisfies the HasClientData interface
func (err ChainContext) GetClientData() interface{} {
	return ClientData(err.ErrCode)
}

var _ ErrorCode = (*ChainContext)(nil)
var _ HasClientData = (*ChainContext)(nil)
var _ unwrapper = (*ChainContext)(nil)

// Format implements the Formatter interface
func (err ChainContext) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v\n", err.ErrCode)
			if errors.HasStack(err.ErrCode) {
				fmt.Fprintf(s, "%v", err.Top)
			} else {
				fmt.Fprintf(s, "%+v", err.Top)
			}
			return
		}
		if s.Flag('#') {
			fmt.Fprintf(s, "ChainContext{Code: %#v, Top: %#v}", err.ErrCode, err.Top)
			return
		}
		fallthrough
	case 's':
		fmt.Fprintf(s, "Code: %s. Top Error: %s", err.ErrCode.Code().CodeStr(), err.Top)
	case 'q':
		fmt.Fprintf(s, "Code: %q. Top Error: %q", err.ErrCode.Code().CodeStr(), err.Top)
	}
}

// Format implements the Formatter interface
func (e MultiErrCode) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v\n", e.ErrCode)
			if errors.HasStack(e.ErrCode) {
				for _, nextErr := range e.rest {
					fmt.Fprintf(s, "%v", nextErr)
				}
			} else {
				for _, nextErr := range e.rest {
					fmt.Fprintf(s, "%+v", nextErr)
				}
			}
			return
		}
		fallthrough
	case 's':
		fmt.Fprintf(s, "%s\n", e.ErrCode)
		fmt.Fprintf(s, "%s", e.rest)
	case 'q':
		fmt.Fprintf(s, "%q\n", e.ErrCode)
		fmt.Fprintf(s, "%q\n", e.rest)
	}
}
