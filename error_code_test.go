// Copyright Greg Weber and PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0

package errcode_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/gregwebs/errcode"
	"github.com/gregwebs/errors"
	"github.com/gregwebs/errors/errwrap"
	"github.com/gregwebs/stackfmt"
)

const httpCodeStr = "input.http" // Test setting the HTTP code

var codeHTTP900 = errcode.InvalidInputCode.Child(httpCodeStr).SetHTTP(900)

type HTTPError struct{ *errcode.CodedError }

func NewHTTPError(err error) HTTPError {
	coded := errcode.NewCodedError(err, codeHTTP900)
	return HTTPError{&coded}
}

func TestHttpErrorCode(t *testing.T) {
	http := NewHTTPError(errors.New("error"))
	AssertHTTPCode(t, http, 900)
	ErrorEquals(t, http, "error")
	ClientDataEquals(t, http, nil, httpCodeStr)
}

type MinimalError struct{} // Test a very simple error

func (e MinimalError) Error() string      { return "error" }
func (e MinimalError) Code() errcode.Code { return registeredCode }

var _ errcode.ErrorCode = (*MinimalError)(nil) // assert implements interface

const codeString errcode.CodeStr = "input.testcode"

var registeredCode errcode.Code = errcode.InvalidInputCode.Child(codeString)

func TestMinimalErrorCode(t *testing.T) {
	minimal := MinimalError{}
	AssertCodes(t, minimal)
	ErrorEquals(t, minimal, "error")
	ClientDataEqualsDef(t, minimal, nil)
	OpEquals(t, minimal, "")
	UserMsgEquals(t, minimal, "")
}

// We don't prevent duplicate codes
var childPathOnlyCode errcode.Code = errcode.InvalidInputCode.Child("testcode")

type ChildOnlyError struct{ *errcode.CodedError }

var _ errcode.ErrorCode = (*ChildOnlyError)(nil) // assert implements interface

func NewChildOnlyError(err error) ChildOnlyError {
	coded := errcode.NewCodedError(err, childPathOnlyCode)
	return ChildOnlyError{&coded}
}
func (e ChildOnlyError) Code() errcode.Code { return childPathOnlyCode }

func TestChildOnlyErrorCode(t *testing.T) {
	coe := NewChildOnlyError(errors.New("error"))
	AssertCodes(t, coe)
	ErrorEquals(t, coe, "error")
	ClientDataEqualsDef(t, coe, nil)
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
	ClientDataEquals(t, top, nil, topCodeStr)
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
	ClientDataEquals(t, deep, nil, deepCodeStr)
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
func (e *ErrorWrapper) WrapError(apply func(err error) error) {
	e.Err = apply(e.Err)
}

var _ errcode.ErrorCode = (*ErrorWrapper)(nil)    // assert implements interface
var _ errwrap.ErrorWrapper = (*ErrorWrapper)(nil) // assert implements interface

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
	err := errors.New("error")
	wrapped := &ErrorWrapper{Err: err}
	AssertCodes(t, wrapped)
	ErrorEquals(t, wrapped, "error")
	ClientDataEqualsDef(t, wrapped, err)
	s2 := Struct2{A: "A", B: "B"}
	wrappedS2 := &ErrorWrapper{Err: s2}
	AssertCodes(t, wrappedS2)
	ErrorEquals(t, wrappedS2, "error A & B A & B")
	ClientDataEqualsDef(t, wrappedS2, s2)
	s1 := Struct1{A: "A"}
	ClientDataEqualsDef(t, &ErrorWrapper{Err: s1}, s1)
	sconst := StructConstError1{A: "A"}
	ClientDataEqualsDef(t, &ErrorWrapper{Err: sconst}, sconst)
}

var internalChildCodeStr errcode.CodeStr = "internal.child.granchild"
var internalChild = errcode.InternalCode.Child("internal.child").SetHTTP(503).Child(internalChildCodeStr)

type InternalChild struct{}

func (ic InternalChild) Error() string      { return "internal child error" }
func (ic InternalChild) Code() errcode.Code { return internalChild }

func TestNewInvalidInputErr(t *testing.T) {
	var err errcode.ErrorCode
	err = errcode.NewInvalidInputErr(errors.New("new error"))
	AssertCodes(t, err, "input")
	ErrorEquals(t, err, "new error")
	ClientDataEquals(t, err, nil, "input")

	err = errcode.NewInvalidInputErr(MinimalError{})
	AssertCodes(t, err, "input.testcode")
	ErrorEquals(t, err, "error")
	ClientDataEquals(t, err, nil, errcode.CodeStr("input.testcode"))

	invalidCodeStr := errcode.InvalidInputCode.CodeStr()
	internalCodeStr := errcode.InternalCode.CodeStr()

	internalErr := errcode.NewInternalErr(MinimalError{})
	err = errcode.NewInvalidInputErr(internalErr)
	AssertCode(t, err, invalidCodeStr)
	AssertHTTPCode(t, err, 400)
	ErrorEquals(t, err, "error")
	ClientDataEquals(t, err, nil, invalidCodeStr, internalErr)

	wrappedInternalErr := errcode.NewInternalErr(internalErr)
	AssertCode(t, wrappedInternalErr, internalCodeStr)
	AssertHTTPCode(t, wrappedInternalErr, 500)
	ErrorEquals(t, err, "error")
	ClientDataEquals(t, wrappedInternalErr, nil, internalCodeStr, MinimalError{})
	// It should use the original stack trace, not the wrapped
	AssertStackEquals(t, wrappedInternalErr, StackTrace(internalErr))

	err = errcode.NewInternalErr(InternalChild{})
	AssertCode(t, err, internalChildCodeStr)
	AssertHTTPCode(t, err, 503)
	ErrorEquals(t, err, "internal child error")
	ClientDataEquals(t, err, nil, internalChildCodeStr)

	err = errcode.NewInvalidInputErr(InternalChild{})
	AssertCode(t, err, invalidCodeStr)
	AssertHTTPCode(t, err, 400)
	ErrorEquals(t, err, "internal child error")
	ClientDataEquals(t, err, nil, invalidCodeStr, InternalChild{})
}

func TestStackTrace(t *testing.T) {
	internalCodeStr := errcode.CodeStr("internal")
	err := errors.New("errors stack")
	wrappedInternalErr := errcode.NewInternalErr(err)
	AssertCode(t, wrappedInternalErr, internalCodeStr)
	AssertHTTPCode(t, wrappedInternalErr, 500)
	ErrorEquals(t, err, "errors stack")
	ClientDataEquals(t, wrappedInternalErr, nil, internalCodeStr)
	// It should use the original stack trace, not the wrapped
	AssertStackEquals(t, wrappedInternalErr, StackTrace(err))
}

func TestNewInternalErr(t *testing.T) {
	internalCodeStr := errcode.CodeStr("internal")
	err := errcode.NewInternalErr(errors.New("new error"))
	AssertCode(t, err, internalCodeStr)
	AssertHTTPCode(t, err, 500)
	ErrorEquals(t, err, "new error")
	ClientDataEquals(t, err, nil, "internal")

	err = errcode.NewInternalErr(MinimalError{})
	AssertCode(t, err, internalCodeStr)
	AssertHTTPCode(t, err, 500)
	ErrorEquals(t, err, "error")
	ClientDataEquals(t, err, nil, internalCodeStr, MinimalError{})

	invalidErr := errcode.NewInvalidInputErr(MinimalError{})
	err = errcode.NewInternalErr(invalidErr)
	AssertCode(t, err, internalCodeStr)
	AssertHTTPCode(t, err, 500)
	ErrorEquals(t, err, "error")
	ClientDataEquals(t, err, nil, internalCodeStr, MinimalError{})
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

func (e UserMsgError) GetUserMsg() string { return "user" }

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
	ClientDataResult(t, has, clientDataResult{
		data:      nil,
		operation: has.GetOperation(),
		codeStr:   has.Code().CodeStr(),
	})
	OpEquals(t, has, "has")

	OpEquals(t, OpErrorEmbed{}, "")
	OpEquals(t, OpErrorEmbed{EmbedOp: errcode.EmbedOp{Op: "field"}}, "field")

	opEmpty := errcode.Op("")
	op := errcode.Op("modify")
	OpEquals(t, opEmpty.AddTo(MinimalError{}), "")
	OpEquals(t, op.AddTo(MinimalError{}), "modify")

	OpEquals(t, &ErrorWrapper{Err: has}, "has")
	OpEquals(t, &ErrorWrapper{Err: OpErrorEmbed{EmbedOp: errcode.EmbedOp{Op: "field"}}}, "field")

	opErrCode := errcode.Op("opcode").AddTo(MinimalError{})
	AssertOperation(t, opErrCode, "opcode")
	OpEquals(t, opErrCode, "opcode")

	OpEquals(t, &ErrorWrapper{Err: opErrCode}, "opcode")
	opErrCode = errcode.Op("opcode").AddTo(has)
	wrappedHas := &ErrorWrapper{Err: opErrCode}
	AssertOperation(t, wrappedHas, "opcode")
	OpEquals(t, wrappedHas, "opcode")
	OpEquals(t, opErrCode, "opcode")
}

/*
func assertPanics[T any](t *testing.T, f func() T) {
	t.Helper()
	var res T
	defer func() {
		if r := recover(); r == nil {
			t.Helper()
			t.Errorf("testPanic: did not panic, got: %v", res)
		}
	}()

	res = f()
}
*/

func TestUserMsg(t *testing.T) {
	AssertUserMsg(t, "foo", "")
	ue := UserMsgError{}
	AssertUserMsg(t, ue, "user")
	AssertCodes(t, ue)
	ErrorEquals(t, ue, "error")
	ClientDataEqualsDef(t, ue, nil)
	UserMsgEquals(t, ue, "user")

	UserMsgEquals(t, UserMsgErrorEmbed{}, "")
	UserMsgEquals(t, UserMsgErrorEmbed{EmbedUserMsg: errcode.EmbedUserMsg{Msg: "field"}}, "field")

	um := errcode.UserMsg("modify")
	UserMsgEquals(t, um.AddTo(MinimalError{}), "modify")

	umEmpty := errcode.UserMsg("")
	if errcode.GetUserMsg(umEmpty.AddTo(MinimalError{})) != "" {
		t.Errorf("expected empty string")
	}
	if umEmpty.AddTo(nil) != nil {
		t.Errorf("expected nil")
	}

	UserMsgEquals(t, &ErrorWrapper{Err: ue}, "user")
	UserMsgEquals(t, &ErrorWrapper{Err: UserMsgErrorEmbed{EmbedUserMsg: errcode.EmbedUserMsg{Msg: "field"}}}, "field")

	msgErrCode := errcode.UserMsg("msg").AddTo(MinimalError{})
	AssertUserMsg(t, msgErrCode, "msg")
	UserMsgEquals(t, msgErrCode, "msg")

	UserMsgEquals(t, &ErrorWrapper{Err: msgErrCode}, "msg")
	wrappedUser := &ErrorWrapper{Err: errcode.UserMsg("msg").AddTo(ue)}
	AssertUserMsg(t, wrappedUser, "msg")
	UserMsgEquals(t, wrappedUser, "msg")
	UserMsgEquals(t, errcode.UserMsg("msg").AddTo(ue), "msg")
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
		t.Errorf("expected HTTP Code %v but got %v", httpCode, expected)
	}
}

func ErrorEquals(t *testing.T, err error, msg string) {
	if err.Error() != msg {
		t.Errorf("Expected error %v. Got error %v", msg, err.Error())
	}
}

func ClientDataEqualsDef(t *testing.T, code errcode.ErrorCode, data interface{}) {
	t.Helper()
	ClientDataEquals(t, code, data, codeString)
}

type clientDataResult struct {
	data       interface{}
	operation  string
	codeStr    errcode.CodeStr
	otherCodes []errcode.ErrorCode
}

func ClientDataResult(t *testing.T, code errcode.ErrorCode, result clientDataResult) {
	t.Helper()

	jsonEquals(t, "ClientData", result.data, errcode.ClientData(code))
	msg := errcode.GetUserMsg(code)
	if msg == "" {
		msg = code.Error()
	}

	others := make([]errcode.JSONFormat, len(result.otherCodes))
	more := make([]errcode.JSONFormat, 0)
	for i, err := range result.otherCodes {
		others[i] = errcode.NewJSONFormat(err)
		if len(others[i].Others) > 0 {
			more = append(more, others[i].Others...)
			others[i].Others = nil
		}
	}
	others = append(others, more...)
	op := result.operation
	if op == "" {
		op = errcode.Operation(result.data)
	}
	jsonExpected := errcode.JSONFormat{
		Data:      result.data,
		Msg:       msg,
		Code:      result.codeStr,
		Operation: op,
		Others:    others,
	}
	newJSON := errcode.NewJSONFormat(code)
	jsonEquals(t, "JSONFormat", jsonExpected, newJSON)
}

func ClientDataEquals(t *testing.T, code errcode.ErrorCode, data interface{}, codeStr errcode.CodeStr, otherCodes ...errcode.ErrorCode) {
	t.Helper()
	ClientDataResult(t, code, clientDataResult{
		data:       data,
		codeStr:    codeStr,
		otherCodes: otherCodes,
	})
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
	opGot := errcode.Operation(code)
	if opGot != op {
		t.Errorf("\nOp expected: %#v\n but got: %#v", op, opGot)
	}
}

func UserMsgEquals(t *testing.T, code errcode.ErrorCode, msg string) {
	t.Helper()
	msgGot := errcode.GetUserMsg(code)
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
	msgGot := errcode.GetUserMsg(v)
	if msgGot != msg {
		t.Errorf("\nUser msg expected: %#v\n but got: %#v", msg, msgGot)
	}
}

func AssertStackEquals(t *testing.T, given errcode.ErrorCode, stExpected stackfmt.StackTrace) {
	t.Helper()
	stGiven := StackTrace(given)
	if stGiven == nil || stExpected == nil || stGiven[0] != stExpected[0] {
		t.Errorf("\nStack expected: %#v\n Stack but got: %#v", stExpected[0], stGiven[0])
	}
}

// StackTrace retrieves the errors.StackTrace from the error if it is present.
// If there is not StackTrace it will return nil
//
// StackTrace looks to see if the error is a StackTracer or if an Unwrap of the error is a StackTracer.
// It will return the stack trace from the deepest error it can find.
func StackTrace(err error) stackfmt.StackTrace {
	if tracer := errors.GetStackTracer(err); tracer != nil {
		return tracer.StackTrace()
	}
	return nil
}
