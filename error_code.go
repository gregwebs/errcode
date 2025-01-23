// Copyright Greg Weber and PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0

// Package errcode facilitates standardized API error codes.
// The goal is that clients can reliably understand errors by checking against error codes
//
// This godoc documents usage. For broader context, see the [README].
//
// Error codes are represented as strings by [CodeStr], not by numbers.
//
// This package is a flexible starting point for you needs-
// The only requirement is to satisfy the [ErrorCode] interface by attaching a [Code] to an error.
//
// A Code can have metadata such as an HTTP code associated with it.
// A code can point to a parent code and will inherit its metadata.
//
// Generic top-level error codes are provided (see the variables section of the doc).
// You are encouraged to create your own error codes customized to your application but generic errors may suffice to get you started.
//
// Stack traces can be automatically added to error codes. This is done by [NewInternalErr].
//
// To extract any ErrorCodes from an error, use [GetCode] or [CodeChain].
//
// [NewJSONFormat] is an opinionated way to send error data to a client: you can define a similar function to meet your needs.
// [JSONFormat] includes a body of response data (the "data field") that is by default the data from the Error
// serialized to JSON.
//
// [README]: https://github.com/gregwebs/errcode/tree/master/README.md
package errcode

import (
	"fmt"
	"strings"
)

// CodeStr is the name of the error code.
// The underlying type is string rather than int.
// This enhances both extensibility (avoids merge conflicts) and user-friendliness.
// CodeStr uses dot separators to indicate hierarchy.
type CodeStr string

func (str CodeStr) String() string { return string(str) }

// A Code has a [CodeStr] representation.
// It is attached to a parent Code to find metadata from it.
type Code struct {
	// this field does not include parent paths
	// The full code (with parent paths) is accessed with the CodeStr() method.
	codeStr CodeStr
	Parent  *Code
}

// CodeStr gives the full dot-seperated path.
// This is what should be used for equality comparison.
func (code Code) CodeStr() CodeStr {
	if code.Parent == nil {
		return code.codeStr
	}
	return (*code.Parent).CodeStr() + "." + code.codeStr
}

// NewCode creates a new top-level code.
// Codes can be created from an existing hierachry with the [Code.Child] method.
// A top-level code must not contain any dot separators: that will panic.
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

// IsAncestor looks for the given code in its ancestors.
func (code Code) IsAncestor(ancestorCode Code) bool {
	if code == ancestorCode {
		return true
	}
	if code.Parent == nil {
		return false
	}
	return (*code.Parent).IsAncestor(ancestorCode)
}

// ErrorCode is the interface that ties an error and Code together.
//
// The ErrorCode interface allows error codes to be flexibly defined.
// There are many pre-existing generic error code implementations available such as [NotFoundErr].
type ErrorCode interface {
	error
	Code() Code
}

// Return the [Code] associated to the error.
// This will unwrap the error until it encounters an [ErrorCode].
// To get the ErrorCode in addition to the code, use [CodeChain].
func GetCode(err error) *Code {
	if errCode := CodeChain(err); errCode != nil {
		code := errCode.Code()
		return &code
	}
	return nil
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
// Normally this function is used rather than GetClientData.
func ClientData(errCode ErrorCode) interface{} {
	if hasData, ok := errCode.(HasClientData); ok {
		return hasData.GetClientData()
	}
	var err error = errCode
	for {
		if un, ok := err.(unwrapError); ok {
			err = un.Unwrap()
			if hasData, ok := err.(HasClientData); ok {
				return hasData.GetClientData()
			}
		} else {
			break
		}
	}
	return nil
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

// operationClientData gives the results of both the ClientData and Operation functions.
// The Operation function is applied to the original ErrorCode.
// If that does not return an operation, it is applied to the result of ClientData.
// This function is used by NewJSONFormat to fill JSONFormat.
func operationClientData(errCode ErrorCode) (string, interface{}) {
	op := Operation(errCode)
	data := ClientData(errCode)
	if op == "" && data != nil {
		op = Operation(data)
	}
	return op, data
}

// NewJSONFormat turns an ErrorCode into a JSONFormat.
// You can create your own json struct and write your own version of this function.
func NewJSONFormat(errCode ErrorCode) JSONFormat {
	return newJSONFormat(errCode, false)
}

func newJSONFormat(errCode ErrorCode, recur bool) JSONFormat {
	// Gather up multiple errors.
	// We discard any that are not ErrorCode.
	var others []JSONFormat
	if !recur {
		errorCodes := ErrorCodes(errCode)[1:]
		others = make([]JSONFormat, len(errorCodes))
		for i, err := range errorCodes {
			others[i] = newJSONFormat(err, true)
		}
	}

	op, data := operationClientData(errCode)

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
