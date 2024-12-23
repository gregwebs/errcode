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
	"net/http"

	"github.com/gregwebs/errors"
)

var (
	// InternalCode is equivalent to HTTP 500 Internal Server Error.
	InternalCode = NewCode("internal").SetHTTP(http.StatusInternalServerError)

	// NotFoundCode is equivalent to HTTP 404 Not Found.
	NotFoundCode = NewCode("missing").SetHTTP(http.StatusNotFound)

	// UnimplementedCode is mapped to HTTP 501.
	UnimplementedCode = InternalCode.Child("internal.unimplemented").SetHTTP(http.StatusNotImplemented)

	// Unavailable is mapped to HTTP 503.
	UnavailableCode = InternalCode.Child("internal.unavailable").SetHTTP(http.StatusServiceUnavailable)

	// StateCode is an error that is invalid due to the current system state.
	// This operatiom could become valid if the system state changes
	// This is mapped to HTTP 400.
	StateCode = NewCode("state").SetHTTP(http.StatusBadRequest)

	// AlreadyExistsCode indicates an attempt to create an entity failed because it already exists.
	// This is mapped to HTTP 422. 409 is sometimes used in these cases, but 409 is supposed to be for re-submittable errors.
	// It would also be possible to use a 400
	AlreadyExistsCode = StateCode.Child("state.exists").SetHTTP(http.StatusUnprocessableEntity)

	// OutOfRangeCode indicates an operation was attempted past a valid range.
	// This is mapped to HTTP 400.
	OutOfRangeCode = StateCode.Child("state.range")

	// InvalidInputCode is equivalent to HTTP 400 Bad Request.
	InvalidInputCode = NewCode("input").SetHTTP(http.StatusBadRequest)

	NotAcceptableCode = InvalidInputCode.Child("input.notacceptable").SetHTTP(http.StatusNotAcceptable)

	// AuthCode represents an authentication or authorization issue.
	AuthCode = NewCode("auth")

	// NotAuthenticatedCode indicates the user is not authenticated.
	// This is mapped to HTTP 401.
	// Note that HTTP 401 is poorly named "Unauthorized".
	NotAuthenticatedCode = AuthCode.Child("auth.unauthenticated").SetHTTP(http.StatusUnauthorized)

	// ForbiddenCode indicates the user is not authorized.
	// This is mapped to HTTP 403.
	ForbiddenCode = AuthCode.Child("auth.forbidden").SetHTTP(http.StatusForbidden)

	UnprocessableEntityCode = StateCode.Child("state.unprocessable").SetHTTP(http.StatusUnprocessableEntity)

	// TimeoutCode represents a timed out connection
	TimeoutCode        = NewCode("timeout")
	TimeoutGatewayCode = TimeoutCode.Child("timeout.gateway").SetHTTP(http.StatusGatewayTimeout)
	TimeoutRequestCode = TimeoutCode.Child("timeout.request").SetHTTP(http.StatusRequestTimeout)
)

// CodedError is a convenience to attach a code to an error and already satisfy the ErrorCode interface.
// If the error is a struct, that struct will get preseneted as data to the client.
//
// To override the http code or the data representation or just for clearer documentation,
// you are encouraged to wrap CodeError with your own struct that inherits it.
// Look at the implementation of invalidInput, InternalErr, and notFound.
type CodedError struct {
	GetCode Code
	*errors.ErrorWrap
}

// NewCodedError is for constructing broad error kinds (e.g. those representing HTTP codes)
// Which could have many different underlying go errors.
// Eventually you may want to give your go errors more specific codes.
// The second argument is the broad code.
//
// If the error given is already an ErrorCode,
// that will be used as the code instead of the second argument.
func NewCodedError(err error, code Code) CodedError {
	if err == nil {
		panic("NewCodedError error is nil")
	}
	if errcode, ok := err.(ErrorCode); ok {
		code = errcode.Code()
	}
	return CodedError{
		GetCode:   code,
		ErrorWrap: errors.NewErrorWrap(err),
	}
}

var _ ErrorCode = (*CodedError)(nil)           // assert implements interface
var _ unwrapError = (*CodedError)(nil)         // assert implements interface
var _ errors.ErrorWrapper = (*CodedError)(nil) // assert implements interface

// Code returns the GetCode field
func (e CodedError) Code() Code {
	return e.GetCode
}

// invalidInputErr gives the code InvalidInputCode.
type invalidInputErr struct{ CodedError }

// NewInvalidInputErr creates an invalidInputErr from an err.
// If the error is already an ErrorCode it will use that code.
// Otherwise it will use InvalidInputCode which gives HTTP 400.
func NewInvalidInputErr(err error) ErrorCode {
	return invalidInputErr{NewCodedError(err, InvalidInputCode)}
}

var _ ErrorCode = (*invalidInputErr)(nil)   // assert implements interface
var _ unwrapError = (*invalidInputErr)(nil) // assert implements interface

// badReqeustErr gives the code BadRequestErr.
type BadRequestErr struct{ CodedError }

// NewBadRequestErr creates a BadReqeustErr from an err.
// If the error is already an ErrorCode it will use that code.
// Otherwise it will use BadRequestCode which gives HTTP 400.
func NewBadRequestErr(err error) BadRequestErr {
	return BadRequestErr{NewCodedError(err, InvalidInputCode)}
}

// InternalErr gives the code InternalCode
type InternalErr struct{ StackCode }

var internalStackCode = makeInternalStackCode(InternalCode)

// NewInternalErr creates an InternalErr from an err.
// If the given err is an ErrorCode that is a descendant of InternalCode,
// its code will be used.
// This ensures the intention of sending an HTTP 50x.
// This function also records a stack trace.
func NewInternalErr(err error) InternalErr {
	return InternalErr{internalStackCode(err)}
}

var _ ErrorCode = (*InternalErr)(nil)   // assert implements interface
var _ unwrapError = (*InternalErr)(nil) // assert implements interface

// makeInternalStackCode builds a function for making an an internal error with a stack trace.
func makeInternalStackCode(defaultCode Code) func(error) StackCode {
	if !defaultCode.IsAncestor(InternalCode) {
		panic(fmt.Errorf("code is not an internal code: %v", defaultCode))
	}
	return func(err error) StackCode {
		if err == nil {
			panic(fmt.Sprintf("makeInternalStackCode %v error is nil", defaultCode))
		}
		code := defaultCode
		if errcode, ok := err.(ErrorCode); ok {
			errCode := errcode.Code()
			if errCode.IsAncestor(InternalCode) {
				code = errCode
			}
		}
		return NewStackCode(CodedError{
			GetCode:   code,
			ErrorWrap: errors.NewErrorWrap(err),
		}, 3)
	}
}

type UnimplementedErr struct{ StackCode }

var unimplementedStackCode = makeInternalStackCode(UnimplementedCode)

// NewUnimplementedErr creates an InternalErr from an err.
// If the given err is an ErrorCode that is a descendant of InternalCode,
// its code will be used.
// This ensures the intention of sending an HTTP 50x.
// This function also records a stack trace.
func NewUnimplementedErr(err error) UnimplementedErr {
	return UnimplementedErr{unimplementedStackCode(err)}
}

type UnavailableErr struct{ StackCode }

var unavailableStackCode = makeInternalStackCode(UnavailableCode)

// NewUnavailableErr creates an InternalErr from an err.
// If the given err is an ErrorCode that is a descendant of InternalCode,
// its code will be used.
// This ensures the intention of sending an HTTP 50x.
// This function also records a stack trace.
func NewUnavailableErr(err error) UnavailableErr {
	return UnavailableErr{unavailableStackCode(err)}
}

// notFound gives the code NotFoundCode.
type NotFoundErr struct{ CodedError }

// NewNotFoundErr creates a notFound from an err.
// If the error is already an ErrorCode it will use that code.
// Otherwise it will use NotFoundCode which gives HTTP 404.
func NewNotFoundErr(err error) NotFoundErr {
	return NotFoundErr{NewCodedError(err, NotFoundCode)}
}

var _ ErrorCode = (*NotFoundErr)(nil)   // assert implements interface
var _ unwrapError = (*NotFoundErr)(nil) // assert implements interface

// NotAuthenticatedErr gives the code NotAuthenticatedCode.
type NotAuthenticatedErr struct{ CodedError }

// NewNotAuthenticatedErr creates a NotAuthenticatedErr from an err.
// If the error is already an ErrorCode it will use that code.
// Otherwise it will use NotAuthenticatedCode which gives HTTP 401.
func NewNotAuthenticatedErr(err error) NotAuthenticatedErr {
	return NotAuthenticatedErr{NewCodedError(err, NotAuthenticatedCode)}
}

var _ ErrorCode = (*NotAuthenticatedErr)(nil)   // assert implements interface
var _ unwrapError = (*NotAuthenticatedErr)(nil) // assert implements interface

// ForbiddenErr gives the code ForbiddenCode.
type ForbiddenErr struct{ CodedError }

// NewForbiddenErr creates a ForbiddenErr from an err.
// If the error is already an ErrorCode it will use that code.
// Otherwise it will use ForbiddenCode which gives HTTP 401.
func NewForbiddenErr(err error) ForbiddenErr {
	return ForbiddenErr{NewCodedError(err, ForbiddenCode)}
}

var _ ErrorCode = (*ForbiddenErr)(nil)   // assert implements interface
var _ unwrapError = (*ForbiddenErr)(nil) // assert implements interface

// UnprocessableErr gives the code UnprocessibleCode.
type UnprocessableErr struct{ CodedError }

// NewUnprocessableErr creates an UnprocessableErr from an err.
// If the error is already an ErrorCode it will use that code.
// Otherwise it will use UnprocessableEntityCode which gives HTTP 422.
func NewUnprocessableErr(err error) UnprocessableErr {
	return UnprocessableErr{NewCodedError(err, UnprocessableEntityCode)}
}

// NotAcceptableErr gives the code NotAcceptableCode.
type NotAcceptableErr struct{ CodedError }

// NewUnprocessableErr creates an UnprocessableErr from an err.
// If the error is already an ErrorCode it will use that code.
// Otherwise it will use NotAcceptableCode which gives HTTP 406.
func NewNotAcceptableErr(err error) NotAcceptableErr {
	return NotAcceptableErr{NewCodedError(err, NotAcceptableCode)}
}

type AlreadyExistsErr struct{ CodedError }

// NewAlreadyExistsErr creates an AlreadyExistsErr from an err.
// If the error is already an ErrorCode it will use that code.
// Otherwise it will use AlreadyExistsCode which gives HTTP 409.
func NewAlreadyExistsErr(err error) AlreadyExistsErr {
	return AlreadyExistsErr{NewCodedError(err, AlreadyExistsCode)}
}

// TimeoutGatewayErr gives the code TimeoutGatewayCode
type TimeoutGatewayErr struct{ CodedError }

// NewTimeoutGatewayErr creates a TimeoutGatewayErr from an err.
// If the error is already an ErrorCode it will use that code.
// Otherwise it will use TimeoutGatewayErr which gives HTTP 504.
func NewTimeoutGatewayErr(err error) TimeoutGatewayErr {
	return TimeoutGatewayErr{NewCodedError(err, TimeoutGatewayCode)}
}

// TimeoutRequestErr gives the code TimeoutRequestCode
type TimeoutRequestErr struct{ CodedError }

// NewTimeoutRequestErr creates a TimeoutRequestErr from an err.
// If the error is already an ErrorCode it will use that code.
// Otherwise it will use TimeoutRequestErr which gives HTTP 408.
func NewTimeoutRequestErr(err error) TimeoutRequestErr {
	return TimeoutRequestErr{NewCodedError(err, TimeoutRequestCode)}
}
