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

// ErrorCodes return all errors (including those grouped) that are of interface ErrorCode.
// It first calls the Errors function.
func ErrorCodes(err error) []ErrorCode {
	errorCodes := make([]ErrorCode, 0)
	//nolint:staticcheck
	errors.WalkDeep(err, func(err error) bool {
		if errcode, ok := err.(ErrorCode); ok {
			// avoid duplicating codes
			if len(errorCodes) == 0 || errorCodes[len(errorCodes)-1].Code().codeStr != errcode.Code().codeStr {
				errorCodes = append(errorCodes, errcode)
			}
		}
		return false
	})
	return errorCodes
}

type multiCode[Err ErrorCode] struct {
	ErrCode Err
	rest    []error
}

// Combine constructs a group that has at least one ErrorCode
// This is "horizontal" composition.
// If you want normal "vertical" composition use the Wrap* functions.
func combineGeneric[Err ErrorCode](initial Err, others ...error) *multiCode[Err] {
	var rest []error
	for _, other := range others {
		if ErrorCode(initial) == nil {
			if errCode, ok := other.(Err); ok {
				initial = errCode
				continue
			}
		}
		rest = append(rest, other)
	}
	if len(rest) == 0 && ErrorCode(initial) == nil {
		return nil
	}
	return &multiCode[Err]{
		ErrCode: initial,
		rest:    rest,
	}
}

var _ ErrorCode = (*multiCode[ErrorCode])(nil)     // assert implements interface
var _ unwrapsError = (*multiCode[ErrorCode])(nil)  // assert implements interface
var _ errorGroup = (*multiCode[ErrorCode])(nil)    // assert implements interface
var _ fmt.Formatter = (*multiCode[ErrorCode])(nil) // assert implements interface

// A MultiErrorCode contains at least one ErrorCode and uses that to satisfy the ErrorCode and related interfaces
// The Error method will produce a string of all the errors with a semi-colon separation.
type MultiErrorCode struct{ multiCode[ErrorCode] }

// A MultiUserCode is similar to a MultiErrorCode but satisfies UserCode
type MultiUserCode struct{ multiCode[UserCode] }

var _ UserCode = (*MultiUserCode)(nil) // assert implements interface

func (e MultiUserCode) GetUserMsg() string {
	return e.ErrCode.GetUserMsg()
}

func Combine(initial ErrorCode, others ...error) *MultiErrorCode {
	combined := combineGeneric(initial, others...)
	if combined == nil {
		return nil
	}
	multiErrCode := multiCode[ErrorCode]{
		ErrCode: combined.ErrCode,
		rest:    combined.rest,
	}
	return &MultiErrorCode{multiErrCode}
}

// CombineUser constructs a group that has at least one UserCode
// It is the same as Combine but the result will satisfy UserCode
func CombineUser(initial UserCode, others ...error) *MultiUserCode {
	combined := combineGeneric(initial, others...)
	if combined == nil {
		return nil
	}
	multiErrCode := multiCode[UserCode]{
		ErrCode: combined.ErrCode,
		rest:    combined.rest,
	}
	return &MultiUserCode{multiErrCode}
}

func (e multiCode[Err]) Error() string {
	output := e.ErrCode.Error()
	for _, item := range e.rest {
		output += "; " + item.Error()
	}
	return output
}

// Errors fullfills the errorGroup inteface
func (e multiCode[Err]) Errors() []error {
	return append([]error{error(e.ErrCode)}, e.rest...)
}

// Code fullfills the ErrorCode inteface
func (e multiCode[Err]) Code() Code {
	return e.ErrCode.Code()
}

// Unwrap fullfills the errors package Unwrap function
func (e multiCode[Err]) Unwrap() []error {
	return e.Errors()
}

func (e multiCode[Err]) First() Err {
	return e.ErrCode
}

type unwrapsError interface {
	Unwrap() []error
}

type errorGroup interface {
	Errors() []error
}

// This interface is checked by errors.As
type asAny interface {
	As(any) bool
}

// CodeChain resolves wrapped errors down to the first ErrorCode.
// An error that is a grouping with multiple codes will have its error codes combined to a MultiErrorCode.
// If the given error is not an ErrorCode, a ContextChain will be returned with Top set to the given error.
// This allows the return object to maintain a full Error() message.
func CodeChain(errInput error) ErrorCode {
	checkError := func(err error) ErrorCode {
		if errCode, ok := err.(ErrorCode); ok {
			return errCode
		}

		as, asOK := err.(asAny)
		{
			var ecAs ErrorCode
			if asOK && as.As(ecAs) {
				return ecAs
			}
		}

		eg, egOK := err.(errorGroup)
		if !egOK && asOK && as.As(eg) {
			egOK = true
		}
		if egOK {
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
					errs := make([]error, len(group[1:]))
					for i, errCode := range group[1:] {
						errs[i] = error(errCode)
					}
					return Combine(group[0], errs...)
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
			return ChainContext{errCode, errInput}
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
	ErrorCode
	Top error
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
	return err.ErrorCode
}

var _ ErrorCode = (*ChainContext)(nil)
var _ unwrapError = (*ChainContext)(nil)

// Format implements the Formatter interface
func (err ChainContext) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v\n", err.ErrorCode)
			if errors.HasStack(err.ErrorCode) {
				fmt.Fprintf(s, "%v", err.Top)
			} else {
				fmt.Fprintf(s, "%+v", err.Top)
			}
			return
		}
		if s.Flag('#') {
			fmt.Fprintf(s, "ChainContext{Code: %#v, Top: %#v}", err.ErrorCode, err.Top)
			return
		}
		fallthrough
	case 's':
		fmt.Fprintf(s, "Code: %s. Top Error: %s", err.ErrorCode.Code().CodeStr(), err.Top)
	case 'q':
		fmt.Fprintf(s, "Code: %q. Top Error: %q", err.ErrorCode.Code().CodeStr(), err.Top)
	}
}

// Format implements the Formatter interface
func (e multiCode[Err]) Format(s fmt.State, verb rune) {
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
