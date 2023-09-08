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

package errcode_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/gregwebs/errcode"
	"github.com/gregwebs/errors"
)

// Test setting the HTTP code
type HTTPError struct{}

func (e HTTPError) Error() string { return "error" }

const httpCodeStr = "input.http"

var codeHTTP900 = errcode.InvalidInputCode.Child(httpCodeStr).SetHTTP(900)

func (e HTTPError) Code() errcode.Code {
	return codeHTTP900
}

func TestHttpErrorCode(t *testing.T) {
	http := HTTPError{}
	AssertHTTPCode(t, http, 900)
	ErrorEquals(t, http, "error")
	ClientDataEquals(t, http, http, httpCodeStr)
}

// Test a very simple error
type MinimalError struct{}

func (e MinimalError) Error() string { return "error" }

var _ errcode.ErrorCode = (*MinimalError)(nil) // assert implements interface

const codeString errcode.CodeStr = "input.testcode"

var registeredCode errcode.Code = errcode.InvalidInputCode.Child(codeString)

func (e MinimalError) Code() errcode.Code { return registeredCode }

func TestMinimalErrorCode(t *testing.T) {
	minimal := MinimalError{}
	AssertCodes(t, minimal)
	ErrorEquals(t, minimal, "error")
	ClientDataEquals(t, minimal, minimal)
	OpEquals(t, minimal, "")
	UserMsgEquals(t, minimal, "")
}

// We don't prevent duplicate codes
var childPathOnlyCode errcode.Code = errcode.InvalidInputCode.Child("testcode")

type ChildOnlyError struct{}

func (e ChildOnlyError) Error() string { return "error" }

var _ errcode.ErrorCode = (*ChildOnlyError)(nil) // assert implements interface

func (e ChildOnlyError) Code() errcode.Code { return childPathOnlyCode }

func TestChildOnlyErrorCode(t *testing.T) {
	coe := ChildOnlyError{}
	AssertCodes(t, coe)
	ErrorEquals(t, coe, "error")
	ClientDataEquals(t, coe, coe)
}

// Test a top-level error
type TopError struct{}

func (e TopError) Error() string { return "error" }

var _ errcode.ErrorCode = (*TopError)(nil) // assert implements interface

const topCodeStr errcode.CodeStr = "top"

var topCode errcode.Code = errcode.NewCode(topCodeStr)

func (e TopError) Code() errcode.Code { return topCode }

func TestTopErrorCode(t *testing.T) {
	top := TopError{}
	AssertCodes(t, top, topCodeStr)
	ErrorEquals(t, top, "error")
	ClientDataEquals(t, top, top, topCodeStr)
}

// Test a deep hierarchy
type DeepError struct{}

func (e DeepError) Error() string { return "error" }

var _ errcode.ErrorCode = (*DeepError)(nil) // assert implements interface

const deepCodeStr errcode.CodeStr = "input.testcode.very.very.deep"

var intermediateCode = registeredCode.Child("input.testcode.very").SetHTTP(800)
var deepCode errcode.Code = intermediateCode.Child("input.testcode.very.very").Child(deepCodeStr)

func (e DeepError) Code() errcode.Code { return deepCode }

func TestDeepErrorCode(t *testing.T) {
	deep := DeepError{}
	AssertHTTPCode(t, deep, 800)
	AssertCode(t, deep, deepCodeStr)
	ErrorEquals(t, deep, "error")
	ClientDataEquals(t, deep, deep, deepCodeStr)
}

// Test an ErrorWrapper that has different error types placed into it
type ErrorWrapper struct{ Err error }

var _ errcode.ErrorCode = (*ErrorWrapper)(nil)     // assert implements interface
var _ errcode.HasClientData = (*ErrorWrapper)(nil) // assert implements interface

func (e ErrorWrapper) Code() errcode.Code {
	return registeredCode
}
func (e ErrorWrapper) Error() string {
	return e.Err.Error()
}
func (e ErrorWrapper) GetClientData() interface{} {
	return e.Err
}
func (e ErrorWrapper) Unwrap() error {
	return e.Err
}

type Struct1 struct{ A string }
type StructConstError1 struct{ A string }

func (e Struct1) Error() string {
	return e.A
}

func (e StructConstError1) Error() string {
	return "error"
}

type Struct2 struct {
	A string
	B string
}

func (e Struct2) Error() string {
	return fmt.Sprintf("error A & B %s & %s", e.A, e.B)
}

func TestErrorWrapperCode(t *testing.T) {
	wrapped := ErrorWrapper{Err: errors.New("error")}
	AssertCodes(t, wrapped)
	ErrorEquals(t, wrapped, "error")
	ClientDataEquals(t, wrapped, errors.New("error"))
	s2 := Struct2{A: "A", B: "B"}
	wrappedS2 := ErrorWrapper{Err: s2}
	AssertCodes(t, wrappedS2)
	ErrorEquals(t, wrappedS2, "error A & B A & B")
	ClientDataEquals(t, wrappedS2, s2)
	s1 := Struct1{A: "A"}
	ClientDataEquals(t, ErrorWrapper{Err: s1}, s1)
	sconst := StructConstError1{A: "A"}
	ClientDataEquals(t, ErrorWrapper{Err: sconst}, sconst)
}

var internalChildCodeStr errcode.CodeStr = "internal.child.granchild"
var internalChild = errcode.InternalCode.Child("internal.child").SetHTTP(503).Child(internalChildCodeStr)

type InternalChild struct{}

func (ic InternalChild) Error() string      { return "internal child error" }
func (ic InternalChild) Code() errcode.Code { return internalChild }

func TestNewInvalidInputErr(t *testing.T) {
	err := errcode.NewInvalidInputErr(errors.New("new error"))
	AssertCodes(t, err, "input")
	ErrorEquals(t, err, "new error")
	ClientDataEquals(t, err, errors.New("new error"), "input")

	err = errcode.NewInvalidInputErr(MinimalError{})
	AssertCodes(t, err, "input.testcode")
	ErrorEquals(t, err, "error")
	ClientDataEquals(t, err, MinimalError{}, errcode.CodeStr("input.testcode"))

	internalErr := errcode.NewInternalErr(MinimalError{})
	err = errcode.NewInvalidInputErr(internalErr)
	internalCodeStr := errcode.CodeStr("internal")
	AssertCode(t, err, internalCodeStr)
	AssertHTTPCode(t, err, 500)
	ErrorEquals(t, err, "error")
	ClientDataEquals(t, err, MinimalError{}, internalCodeStr)

	wrappedInternalErr := errcode.NewInternalErr(internalErr)
	AssertCode(t, err, internalCodeStr)
	AssertHTTPCode(t, err, 500)
	ErrorEquals(t, err, "error")
	ClientDataEquals(t, wrappedInternalErr, MinimalError{}, internalCodeStr)
	// It should use the original stack trace, not the wrapped
	AssertStackEquals(t, wrappedInternalErr, errcode.StackTrace(internalErr))

	err = errcode.NewInvalidInputErr(InternalChild{})
	AssertCode(t, err, internalChildCodeStr)
	AssertHTTPCode(t, err, 503)
	ErrorEquals(t, err, "internal child error")
	ClientDataEquals(t, err, InternalChild{}, internalChildCodeStr)
}

func TestStackTrace(t *testing.T) {
	internalCodeStr := errcode.CodeStr("internal")
	err := errors.New("errors stack")
	wrappedInternalErr := errcode.NewInternalErr(err)
	AssertCode(t, wrappedInternalErr, internalCodeStr)
	AssertHTTPCode(t, wrappedInternalErr, 500)
	ErrorEquals(t, err, "errors stack")
	ClientDataEquals(t, wrappedInternalErr, err, internalCodeStr)
	// It should use the original stack trace, not the wrapped
	AssertStackEquals(t, wrappedInternalErr, errcode.StackTrace(err))
}

func TestNewInternalErr(t *testing.T) {
	internalCodeStr := errcode.CodeStr("internal")
	err := errcode.NewInternalErr(errors.New("new error"))
	AssertCode(t, err, internalCodeStr)
	AssertHTTPCode(t, err, 500)
	ErrorEquals(t, err, "new error")
	ClientDataEquals(t, err, errors.New("new error"), "internal")

	err = errcode.NewInternalErr(MinimalError{})
	AssertCode(t, err, internalCodeStr)
	AssertHTTPCode(t, err, 500)
	ErrorEquals(t, err, "error")
	ClientDataEquals(t, err, MinimalError{}, internalCodeStr)

	invalidErr := errcode.NewInvalidInputErr(MinimalError{})
	err = errcode.NewInternalErr(invalidErr)
	AssertCode(t, err, internalCodeStr)
	AssertHTTPCode(t, err, 500)
	ErrorEquals(t, err, "error")
	ClientDataEquals(t, err, MinimalError{}, internalCodeStr)
}

// Test Operation
type OpErrorHas struct{ MinimalError }

func (e OpErrorHas) GetOperation() string { return "has" }

type OpErrorEmbed struct {
	errcode.EmbedOp
	MinimalError
}

var _ errcode.ErrorCode = (*OpErrorHas)(nil)      // assert implements interface
var _ errcode.HasOperation = (*OpErrorHas)(nil)   // assert implements interface
var _ errcode.ErrorCode = (*OpErrorEmbed)(nil)    // assert implements interface
var _ errcode.HasOperation = (*OpErrorEmbed)(nil) // assert implements interface

type UserMsgError struct{ MinimalError }

func (e UserMsgError) UserMsg() string { return "user" }

type UserMsgErrorEmbed struct {
	errcode.EmbedUserMsg
	MinimalError
}

var _ errcode.ErrorCode = (*UserMsgError)(nil)       // assert implements interface
var _ errcode.HasUserMsg = (*UserMsgError)(nil)      // assert implements interface
var _ errcode.ErrorCode = (*UserMsgErrorEmbed)(nil)  // assert implements interface
var _ errcode.HasUserMsg = (*UserMsgErrorEmbed)(nil) // assert implements interface

func TestOpErrorCode(t *testing.T) {
	AssertOperation(t, "foo", "")
	has := OpErrorHas{}
	AssertOperation(t, has, "has")
	AssertCodes(t, has)
	ErrorEquals(t, has, "error")
	ClientDataEquals(t, has, has)
	OpEquals(t, has, "has")

	OpEquals(t, OpErrorEmbed{}, "")
	OpEquals(t, OpErrorEmbed{EmbedOp: errcode.EmbedOp{Op: "field"}}, "field")

	opEmpty := errcode.Op("")
	op := errcode.Op("modify")
	OpEquals(t, opEmpty.AddTo(MinimalError{}), "")
	OpEquals(t, op.AddTo(MinimalError{}), "modify")

	OpEquals(t, ErrorWrapper{Err: has}, "has")
	OpEquals(t, ErrorWrapper{Err: OpErrorEmbed{EmbedOp: errcode.EmbedOp{Op: "field"}}}, "field")

	opErrCode := errcode.OpErrCode{Operation: "opcode", Err: MinimalError{}}
	AssertOperation(t, opErrCode, "opcode")
	OpEquals(t, opErrCode, "opcode")

	OpEquals(t, ErrorWrapper{Err: opErrCode}, "opcode")
	wrappedHas := ErrorWrapper{Err: errcode.OpErrCode{Operation: "opcode", Err: has}}
	AssertOperation(t, wrappedHas, "opcode")
	OpEquals(t, wrappedHas, "opcode")
	OpEquals(t, errcode.OpErrCode{Operation: "opcode", Err: has}, "opcode")
}

func TestUserMsg(t *testing.T) {
	AssertUserMsg(t, "foo", "")
	ue := UserMsgError{}
	AssertUserMsg(t, ue, "user")
	AssertCodes(t, ue)
	ErrorEquals(t, ue, "error")
	ClientDataEquals(t, ue, ue)
	UserMsgEquals(t, ue, "user")

	UserMsgEquals(t, UserMsgErrorEmbed{}, "")
	UserMsgEquals(t, UserMsgErrorEmbed{EmbedUserMsg: errcode.EmbedUserMsg{Msg: "field"}}, "field")

	umEmpty := errcode.NewUserMsg("")
	um := errcode.NewUserMsg("modify")
	UserMsgEquals(t, umEmpty.AddTo(MinimalError{}), "")
	UserMsgEquals(t, um.AddTo(MinimalError{}), "modify")

	UserMsgEquals(t, ErrorWrapper{Err: ue}, "user")
	UserMsgEquals(t, ErrorWrapper{Err: UserMsgErrorEmbed{EmbedUserMsg: errcode.EmbedUserMsg{Msg: "field"}}}, "field")

	msgErrCode := errcode.UserMsgErrCode{Msg: "msg", Err: MinimalError{}}
	AssertUserMsg(t, msgErrCode, "msg")
	UserMsgEquals(t, msgErrCode, "msg")

	UserMsgEquals(t, ErrorWrapper{Err: msgErrCode}, "msg")
	wrappedUser := ErrorWrapper{Err: errcode.UserMsgErrCode{Msg: "msg", Err: ue}}
	AssertUserMsg(t, wrappedUser, "msg")
	UserMsgEquals(t, wrappedUser, "msg")
	UserMsgEquals(t, errcode.UserMsgErrCode{Msg: "msg", Err: ue}, "msg")
}

func AssertCodes(t *testing.T, code errcode.ErrorCode, codeStrs ...errcode.CodeStr) {
	t.Helper()
	AssertCode(t, code, codeStrs...)
	AssertHTTPCode(t, code, 400)
}

func AssertCode(t *testing.T, code errcode.ErrorCode, codeStrs ...errcode.CodeStr) {
	t.Helper()
	codeStr := codeString
	if len(codeStrs) > 0 {
		codeStr = codeStrs[0]
	}
	if code.Code().CodeStr() != codeStr {
		t.Errorf("code expected %v\ncode but got %v", codeStr, code.Code().CodeStr())
	}
}

func AssertHTTPCode(t *testing.T, code errcode.ErrorCode, httpCode int) {
	t.Helper()
	expected := code.Code().HTTPCode()
	if expected != httpCode {
		t.Errorf("excpected HTTP Code %v but got %v", httpCode, expected)
	}
}

func ErrorEquals(t *testing.T, err error, msg string) {
	if err.Error() != msg {
		t.Errorf("Expected error %v. Got error %v", msg, err.Error())
	}
}

func ClientDataEquals(t *testing.T, code errcode.ErrorCode, data interface{}, codeStrs ...errcode.CodeStr) {
	codeStr := codeString
	var stack errors.StackTrace
	if len(codeStrs) > 0 {
		codeStr = codeStrs[0]
		if code.Code().IsAncestor(errcode.InternalCode) {
			stack = errcode.StackTrace(code)
		}
	}
	t.Helper()

	jsonEquals(t, "ClientData", data, errcode.ClientData(code))
	msg := errcode.UserMsg(code)
	if msg == "" {
		msg = code.Error()
	}

	jsonExpected := errcode.JSONFormat{
		Data:      data,
		Msg:       msg,
		Code:      codeStr,
		Operation: errcode.Operation(data),
		Stack:     stack,
	}
	newJSON := errcode.NewJSONFormat(code)
	jsonEquals(t, "JSONFormat", jsonExpected, newJSON)
}

func jsonEquals(t *testing.T, errPrefix string, expectedIn interface{}, gotIn interface{}) {
	t.Helper()
	got, err1 := json.Marshal(gotIn)
	expected, err2 := json.Marshal(expectedIn)
	if err1 != nil || err2 != nil {
		t.Errorf("%v could not serialize to json", errPrefix)
	}
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("%v\nClientData expected: %#v\n ClientData but got: %#v", errPrefix, string(expected), string(got))
	}
}

func OpEquals(t *testing.T, code errcode.ErrorCode, op string) {
	t.Helper()
	opGot, _ := errcode.OperationClientData(code)
	if opGot != op {
		t.Errorf("\nOp expected: %#v\n but got: %#v", op, opGot)
	}
}

func UserMsgEquals(t *testing.T, code errcode.ErrorCode, msg string) {
	t.Helper()
	msgGot := errcode.UserMsg(code)
	if msgGot != msg {
		t.Errorf("\nUser msg expected: %#v\n but got: %#v", msg, msgGot)
	}
}

func AssertOperation(t *testing.T, v interface{}, op string) {
	t.Helper()
	opGot := errcode.Operation(v)
	if opGot != op {
		t.Errorf("\nOp expected: %#v\n but got: %#v", op, opGot)
	}
}

func AssertUserMsg(t *testing.T, v interface{}, msg string) {
	t.Helper()
	msgGot := errcode.UserMsg(v)
	if msgGot != msg {
		t.Errorf("\nUser msg expected: %#v\n but got: %#v", msg, msgGot)
	}
}

func AssertStackEquals(t *testing.T, given errcode.ErrorCode, stExpected errors.StackTrace) {
	t.Helper()
	stGiven := errcode.StackTrace(given)
	if stGiven == nil || stExpected == nil || stGiven[0] != stExpected[0] {
		t.Errorf("\nStack expected: %#v\n Stack but got: %#v", stExpected[0], stGiven[0])
	}
}
