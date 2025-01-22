// Copyright Greg Weber and PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0

package errcode

import (
	"fmt"
	"net/http"

	"github.com/gregwebs/errors/errwrap"
)

var (
	// InternalCode is associated to [http.StatusInternalServerError] HTTP 500
	InternalCode = NewCode("internal").SetHTTP(http.StatusInternalServerError)

	// NotFoundCode is associated to [http.StatusNotFound] HTTP 404
	NotFoundCode = NewCode("missing").SetHTTP(http.StatusNotFound)

	// UnimplementedCode is associated to [http.StatusNotImplemented] HTTP 501
	UnimplementedCode = InternalCode.Child("internal.unimplemented").SetHTTP(http.StatusNotImplemented)

	// Unavailable is mapped to [http.StatusServiceUnavailable] HTTP 503
	UnavailableCode = InternalCode.Child("internal.unavailable").SetHTTP(http.StatusServiceUnavailable)

	// StateCode is an error that is invalid due to the current system state.
	// This operatiom could become valid if the system state changes
	// This is associated to HTTP 400.
	StateCode = NewCode("state").SetHTTP(http.StatusBadRequest)

	// AlreadyExistsCode indicates an attempt to create an entity failed because it already exists.
	// This is mapped to HTTP 422. 409 is sometimes used in these cases, but 409 is supposed to be for re-submittable errors.
	// It would also be possible to use a 400
	AlreadyExistsCode = StateCode.Child("state.exists").SetHTTP(http.StatusUnprocessableEntity)

	// OutOfRangeCode indicates an operation was attempted past a valid range.
	// This is associated to HTTP 400.
	OutOfRangeCode = StateCode.Child("state.range")

	// InvalidInputCode is equivalent to HTTP 400 Bad Request.
	InvalidInputCode = NewCode("input").SetHTTP(http.StatusBadRequest)

	// NotAcceptableCode is associated to [http.StatusNotAcceptable] HTTP 406
	NotAcceptableCode = InvalidInputCode.Child("input.notacceptable").SetHTTP(http.StatusNotAcceptable)

	// AuthCode represents an authentication or authorization issue.
	// This is the parent code of [NotAuthenticatedCode] and [ForbiddenCode]
	AuthCode = NewCode("auth")

	// NotAuthenticatedCode indicates the user is not authenticated.
	// This is associated to [http.StatusUnauthorized] HTTP 401 which is poorly named "Unauthorized".
	NotAuthenticatedCode = AuthCode.Child("auth.unauthenticated").SetHTTP(http.StatusUnauthorized)

	// ForbiddenCode indicates the user is not authorized.
	// This is associated to [http.StatusForbidden] HTTP 403.
	ForbiddenCode = AuthCode.Child("auth.forbidden").SetHTTP(http.StatusForbidden)

	// UnprocessableEntityCode is associated to [http.StatusUnprocessableEntity] HTTP 422
	UnprocessableEntityCode = StateCode.Child("state.unprocessable").SetHTTP(http.StatusUnprocessableEntity)

	// TimeoutCode represents a timed out connection. It is the parent code of [TimeoutGatewayCode] and [TimeoutRequestCode].
	TimeoutCode = NewCode("timeout")

	// TimeoutGatewayCode is associated to [http.StatusGatewayTimeout] HTTP 504
	TimeoutGatewayCode = TimeoutCode.Child("timeout.gateway").SetHTTP(http.StatusGatewayTimeout)
	// TimeoutRequestCode is associated to [http.StatusRequestTimeout] HTTP 408
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
	*errwrap.ErrorWrap
}

// NewCodedError is a helper for constructing error codes.
// If the error given is already an ErrorCode descending from the given Code,
// that will be used as the code.
func NewCodedError(err error, code Code) CodedError {
	ce, _ := newCodedError(err, code)
	return ce
}

func newCodedError(err error, code Code) (CodedError, ErrorCode) {
	if err == nil {
		panic("NewCodedError error is nil")
	}
	var alternative ErrorCode
	if errcode, ok := err.(ErrorCode); ok {
		if errcode.Code().IsAncestor(code) {
			code = errcode.Code()
		} else {
			alternative = errcode
		}
	}
	return CodedError{
		GetCode:   code,
		ErrorWrap: errwrap.NewErrorWrap(err),
	}, alternative
}

var _ ErrorCode = (*CodedError)(nil)            // assert implements interface
var _ unwrapError = (*CodedError)(nil)          // assert implements interface
var _ errwrap.ErrorWrapper = (*CodedError)(nil) // assert implements interface

// Code returns the GetCode field
func (e CodedError) Code() Code {
	return e.GetCode
}

// invalidInputErr gives the code InvalidInputCode.
type invalidInputErr struct{ CodedError }

// NewInvalidInputErr creates an invalidInputErr from an err.
// If the error is already a descendant of InvalidInputCode it will use that code.
// Otherwise it will use InvalidInputCode which gives HTTP 400.
func NewInvalidInputErr(err error) ErrorCode {
	return invalidInputErr{NewCodedError(err, InvalidInputCode)}
}

var _ ErrorCode = (*invalidInputErr)(nil)   // assert implements interface
var _ unwrapError = (*invalidInputErr)(nil) // assert implements interface

// BadReqeustErr is coded to InvalidInputCode
type BadRequestErr struct{ CodedError }

// NewBadRequestErr creates a BadReqeustErr from an error.
// If the error is already a descendant of InvalidInputCode it will use that code.
// Otherwise it will use InvalidInputCode which gives HTTP 400.
func NewBadRequestErr(err error) BadRequestErr {
	return BadRequestErr{NewCodedError(err, InvalidInputCode)}
}

// InternalErr is a coded to [InternalCode] and will have a stack trace attached.
type InternalErr struct{ StackCode }

var internalStackCode = makeInternalStackCode(InternalCode)

// NewInternalErr creates an [InternalErr] from an error.
// If the given error is an [ErrorCode] that is a descendant of [InternalCode],
// its code will be used.
// This ensures the intention of sending an HTTP 50x.
func NewInternalErr(err error) InternalErr {
	return InternalErr{internalStackCode(err)}
}

var _ ErrorCode = (*InternalErr)(nil)   // assert implements interface
var _ unwrapError = (*InternalErr)(nil) // assert implements interface

// makeInternalStackCode builds a function for making an an internal error with a stack trace.
func makeInternalStackCode(defaultCode Code) func(error) StackCode {
	if !(defaultCode.IsAncestor(InternalCode) || defaultCode.HTTPCode() >= 500) {
		panic(fmt.Errorf("code is not an internal code: %v", defaultCode))
	}
	return func(err error) StackCode {
		if err == nil {
			panic(fmt.Sprintf("makeInternalStackCode %v error is nil", defaultCode))
		}
		code := defaultCode
		if errcode, ok := err.(ErrorCode); ok {
			errCode := errcode.Code()
			if errCode.IsAncestor(defaultCode) {
				code = errCode
			}
		}
		return NewStackCode(CodedError{
			GetCode:   code,
			ErrorWrap: errwrap.NewErrorWrap(err),
		}, 3)
	}
}

// UnimplementedErr is coded to [UnimplementedCode] and ensures a stack trace
type UnimplementedErr struct{ StackCode }

var unimplementedStackCode = makeInternalStackCode(UnimplementedCode)

// NewUnimplementedErr creates an [UnimplementedErr] from an error.
// If the given error is an [ErrorCode] that is a descendant of UnimplementedCode,
// its code will be used.
func NewUnimplementedErr(err error) UnimplementedErr {
	return UnimplementedErr{unimplementedStackCode(err)}
}

// UnavailableErr is coded to UnavailableCode and ensures a stack trace
type UnavailableErr struct{ StackCode }

var unavailableStackCode = makeInternalStackCode(UnavailableCode)

// NewUnavailableErr creates an [UnavailableErr] from an error.
// If the given error is an [ErrorCode] that is a descendant of UnavailableCode,
// its code will be used.
func NewUnavailableErr(err error) UnavailableErr {
	return UnavailableErr{unavailableStackCode(err)}
}

// NotFoundErr is coded to [NotFoundCode].
type NotFoundErr struct{ CodedError }

// NewNotFoundErr creates a [NotFoundErr] from an error.
// If the error is already a descendant of [NotFoundCode] it will use that code.
// Otherwise it will use [NotFoundCode] which gives HTTP 404.
func NewNotFoundErr(err error) NotFoundErr {
	return NotFoundErr{NewCodedError(err, NotFoundCode)}
}

var _ ErrorCode = (*NotFoundErr)(nil)   // assert implements interface
var _ unwrapError = (*NotFoundErr)(nil) // assert implements interface

// NotAuthenticatedErr gives the code [NotAuthenticatedCode].
type NotAuthenticatedErr struct{ CodedError }

// NewNotAuthenticatedErr creates a [NotAuthenticatedErr] from an error.
// If the error is already a descendant of [NotAuthenticatedCode] it will use that code.
// Otherwise it will use [NotAuthenticatedCode] which gives HTTP 401.
func NewNotAuthenticatedErr(err error) NotAuthenticatedErr {
	return NotAuthenticatedErr{NewCodedError(err, NotAuthenticatedCode)}
}

var _ ErrorCode = (*NotAuthenticatedErr)(nil)   // assert implements interface
var _ unwrapError = (*NotAuthenticatedErr)(nil) // assert implements interface

// ForbiddenErr is coded to [ForbiddenCode].
type ForbiddenErr struct{ CodedError }

// NewForbiddenErr creates a [ForbiddenErr] from an error.
// If the error is already a descendant of [ForbiddenCode] it will use that code.
// Otherwise it will use [ForbiddenCode] which gives HTTP 401.
func NewForbiddenErr(err error) ForbiddenErr {
	return ForbiddenErr{NewCodedError(err, ForbiddenCode)}
}

var _ ErrorCode = (*ForbiddenErr)(nil)   // assert implements interface
var _ unwrapError = (*ForbiddenErr)(nil) // assert implements interface

// UnprocessableErr gives the code [UnprocessableEntityCode].
type UnprocessableErr struct{ CodedError }

// NewUnprocessableErr creates an [UnprocessableErr] from an error.
// If the error is already a descedant of [UnprocessableEntityCode] it will use that code.
// Otherwise it will use [UnprocessableEntityCode] which gives HTTP 422.
func NewUnprocessableErr(err error) UnprocessableErr {
	return UnprocessableErr{NewCodedError(err, UnprocessableEntityCode)}
}

// NotAcceptableErr is coded to [NotAcceptableCode].
type NotAcceptableErr struct{ CodedError }

// NewUnprocessableErr creates an [NotAcceptableCode] from an error.
// If the error is already a descendant of [NotAcceptableCode] it will use that code.
// Otherwise it will use [NotAcceptableCode] which gives HTTP 406.
func NewNotAcceptableErr(err error) NotAcceptableErr {
	return NotAcceptableErr{NewCodedError(err, NotAcceptableCode)}
}

type AlreadyExistsErr struct{ CodedError }

// NewAlreadyExistsErr creates an [AlreadyExistsErr] from an error.
// If the error is already a descendant of [AlreadyExistsCode] it will use that code.
// Otherwise it will use [AlreadyExistsCode] which gives HTTP 409.
func NewAlreadyExistsErr(err error) AlreadyExistsErr {
	return AlreadyExistsErr{NewCodedError(err, AlreadyExistsCode)}
}

// TimeoutGatewayErr is coded to [TimeoutGatewayCode].
type TimeoutGatewayErr struct{ CodedError }

// NewTimeoutGatewayErr creates a TimeoutGatewayErr from an error.
// If the error is already a descendant of [TimeoutGatewayCode] it will use that code.
// Otherwise it will use [TimeoutGatewayErr] which gives HTTP 504.
func NewTimeoutGatewayErr(err error) TimeoutGatewayErr {
	return TimeoutGatewayErr{NewCodedError(err, TimeoutGatewayCode)}
}

// TimeoutRequestErr gives the code [TimeoutRequestCode]
type TimeoutRequestErr struct{ CodedError }

// NewTimeoutRequestErr creates a [TimeoutRequestErr] from an error.
// If the error is already a descendant of [TimeoutRequestCode] it will use that code.
// Otherwise it will use [TimeoutRequestErr] which gives HTTP 408.
func NewTimeoutRequestErr(err error) TimeoutRequestErr {
	return TimeoutRequestErr{NewCodedError(err, TimeoutRequestCode)}
}
