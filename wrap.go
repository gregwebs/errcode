package errcode

import (
	"github.com/gregwebs/errors"
)

// Wrap calls errors.Wrap on the inner error.
// This will wrap in place via errors.ErrorWrapper if available
// If a nil is given it is a noop
func Wrap(errCode ErrorCode, msg string) ErrorCode {
	//nolint:govet
	return wrapG(wrapWith, errCode, msg)
}

// Wrapf calls errors.Wrapf on the inner error.
// This will wrap in place via errors.ErrorWrapper if available
// If a nil is given it is a noop
func Wrapf(errCode ErrorCode, msg string, args ...interface{}) ErrorCode {
	return wrapG(wrapWith, errCode, msg, args...)
}

// Wraps calls errors.Wraps on the inner error.
// This will wrap in place via errors.ErrorWrapper if available
// If a nil is given it is a noop
func Wraps(errCode ErrorCode, msg string, args ...interface{}) ErrorCode {
	return wrapWith(errCode, errors.WrapsFn(msg, args...))
}

// WrapUser calls errors.Wrap on the inner error.
// This will wrap in place via errors.ErrorWrapper if available
// If a nil is given it is a noop
func WrapUser[Err UserCode](errCode Err, msg string) *wrappedUserCode[Err] {
	return wrapUserWith(errCode, errors.WrapFn(msg))
}

// WrapUserf calls errors.Wrapf on the inner error.
// This will wrap in place via errors.ErrorWrapper if available
// If a nil is given it is a noop
func WrapfUser[Err UserCode](errCode Err, msg string, args ...interface{}) *wrappedUserCode[Err] {
	if len(args) == 0 {
		return wrapUserWith(errCode, errors.WrapFn(msg))
	}
	return wrapUserWith(errCode, errors.WrapfFn(msg, args...))
}

// WrapsUser calls errors.Wraps on the inner error.
// This uses the WrapError method of ErrorWrap
// If a nil is given it is a noop
func WrapsUser(errCode UserCode, msg string, args ...interface{}) UserCode {
	return wrapUserWith(errCode, errors.WrapsFn(msg, args...))
}

// WrapOp calls errors.Wrap on the inner error.
// This will wrap in place via errors.ErrorWrapper if available
// If a nil is given it is a noop
func WrapOp(errCode OpCode, msg string) OpCode {
	//nolint:govet
	return wrapG(wrapOpWith, errCode, msg)
}

// WrapOpf calls errors.Wrapf on the inner error.
// This will wrap in place via errors.ErrorWrapper if available
// If a nil is given it is a noop
func WrapfOp(errCode OpCode, msg string, args ...interface{}) OpCode {
	return wrapG(wrapOpWith, errCode, msg, args...)
}

// WrapsOp calls errors.Wraps on the inner error.
// This uses the WrapError method of ErrorWrap
// If a nil is given it is a noop
func WrapsOp(errCode OpCode, msg string, args ...interface{}) OpCode {
	return wrapOpWith(errCode, errors.WrapsFn(msg, args...))
}

func wrapWith(errCode ErrorCode, wrap func(error) error) ErrorCode {
	if errCode == nil {
		return errCode
	}
	ok := errors.WrapInPlace(errCode, wrap)
	if ok {
		return errCode
	}
	return wrappedErrorCode[ErrorCode]{newWithError(errCode, wrap)}
}

func id[T any](x T) T {
	return x
}

func wrapUserWith[Err UserCode](errCode Err, wrap func(error) error) *wrappedUserCode[Err] {
	if UserCode(errCode) == nil {
		return nil
	}
	ok := errors.WrapInPlace(errCode, wrap)
	if ok {
		return &wrappedUserCode[Err]{newWithError(errCode, id)}
	}
	return &wrappedUserCode[Err]{newWithError(errCode, wrap)}
}

func wrapOpWith(errCode OpCode, wrap func(error) error) OpCode {
	if errCode == nil {
		return errCode
	}
	ok := errors.WrapInPlace(errCode, wrap)
	if ok {
		return errCode
	}
	return wrappedOpCode{newWithError(errCode, wrap)}
}

// unwrapError allows the abstract retrieval of the underlying error.
// Formalize the Unwrap interface, but don't export it.
// The standard library errors package should export it.
// Types that wrap errors should implement this to allow viewing of the underlying error.
type unwrapError interface {
	Unwrap() error
}

type withError[T any] struct {
	With T
	*errors.ErrorWrap
}

// do a nil check before calling this
func newWithError[Err error](errCode Err, wrapErr func(error) error) withError[Err] {
	return withError[Err]{
		With:      errCode,
		ErrorWrap: errors.NewErrorWrap(wrapErr(errCode)),
	}
}

type wrappedErrorCode[Err ErrorCode] struct{ withError[ErrorCode] }

func (wec wrappedErrorCode[Err]) Code() Code {
	return wec.With.Code()
}

type wrappedUserCode[Err UserCode] struct{ withError[Err] }

func (wec wrappedUserCode[Err]) Code() Code {
	return wec.With.Code()
}

func (wec wrappedUserCode[Err]) Unwrap() error {
	return wec.With
}

func (wec wrappedUserCode[Err]) GetUserMsg() string {
	return wec.With.GetUserMsg()
}

type wrappedOpCode struct{ withError[OpCode] }

func (wec wrappedOpCode) Code() Code {
	return wec.With.Code()
}

func (wec wrappedOpCode) GetOperation() string {
	return wec.With.GetOperation()
}

func wrapG[Err ErrorCode](errWrap func(Err, func(error) error) Err, errCode Err, msg string, args ...interface{}) Err {
	if len(args) == 0 {
		return errWrap(errCode, errors.WrapFn(msg))
	}
	return errWrap(errCode, errors.WrapfFn(msg, args...))
}
