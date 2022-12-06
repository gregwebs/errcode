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

// ErrorCodes uses CodeChain to return all ErrorCodes that can be found by unwrapping or looking at the ErrorGroup.
func ErrorCodes(err error) []ErrorCode {
	ec := CodeChain(err)
	if ec == nil {
		return nil
	}
	fmt.Printf("%#v\n", ec)
	if mc, ok := ec.(MultiErrCode); ok {
		return mc.ErrorCodes()
	} else if cc, ok := ec.(ChainContext); ok {
		return append([]ErrorCode{cc.ErrCode}, ErrorCodes(cc.Next)...)
	} else {
		return []ErrorCode{ec}
	}
}

// A MultiErrCode contains at least one ErrorCode and uses that to satisfy the ErrorCode and related interfaces
// The Error method will produce a string of all the errors with a semi-colon separation.
// Later code (such as a JSON response) needs to look for the ErrorGroup interface.
type MultiErrCode struct {
	ErrCode ErrorCode
	rest    []ErrorCode
}

// Combine constructs a MultiErrCode.
// It will combine any other MultiErrCode into just one MultiErrCode.
// This is "horizontal" composition.
// If you want normal "vertical" composition use BuildChain.
func Combine(initial ErrorCode, others ...ErrorCode) MultiErrCode {
	return MultiErrCode{
		ErrCode: initial,
		rest:    others,
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
	errs := make([]error, 1+len(e.rest))
	errs[0] = e.ErrCode.(error)
	for i, er := range e.rest {
		errs[i+1] = er.(error)
	}
	return errs
}

func (e MultiErrCode) ErrorCodes() []ErrorCode {
	return append([]ErrorCode{e.ErrCode}, e.rest...)
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

// CodeChain resolves an error chain down to a chain of just error codes
// Any ErrorGroups found are converted to a MultiErrCode.
// Passed over error information is retained using ChainContext.
// If a code was overidden in the chain, it will show up as a MultiErrCode.
func CodeChain(err error) ErrorCode {
	return codeChainTop(err, err, true)
}

// keep track of the top error
func codeChainTop(err error, top error, errsAreEqual bool) ErrorCode {
	if err == nil {
		return nil
	}
	if top == nil {
		panic("nil top")
	}
	makeChainErrCode := func(errcode ErrorCode) ErrorCode {
		next := CodeChain(errors.Unwrap(err))
		if next == nil && errsAreEqual {
			return errcode
		}
		return ChainContext{Top: top, ErrCode: errcode, Next: next}
	}
	if errcode, ok := err.(ErrorCode); ok {
		return makeChainErrCode(errcode)
	} else if eg, ok := err.(errors.ErrorGroup); ok {
		group := []ErrorCode{}
		for _, errItem := range eg.Errors() {
			if itemCode := codeChainTop(errItem, top, false); itemCode != nil {
				group = append(group, itemCode)
			}
		}
		if len(group) == 0 {
			return codeChainTop(errors.Unwrap(err), top, false)
		}

		var codeGroup ErrorCode
		if len(group) == 1 {
			codeGroup = group[0]
		} else {
			codeGroup = Combine(group[0], group[1:]...)
		}
		return makeChainErrCode(codeGroup)
	} else {
		return codeChainTop(errors.Unwrap(err), top, false)
	}
}

// ChainContext is returned by CodeChain
// to retain the full wrapped error message of the error chain.
// If you annotated an ErrorCode with additional information, it is retained in the Top field.
// The Top field is used for the Error() and Unwrap() methods.
// The Next field is used to point to the Next known ErrorCode in the chain
type ChainContext struct {
	Top     error
	ErrCode ErrorCode
	Next    ErrorCode
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
