package errcode

import (
	"github.com/gregwebs/errors"
)

// Wrap calls errors.Wrap or errors.Wrapf on the inner error.
// This will wrap in place via errors.ErrorWrapper if available
// If a nil is given it is a noop
func Wrap(errCode ErrorCode, msg string, args ...interface{}) ErrorCode {
	return wrapG(wrapWith, errCode, msg, args...)
}

// Wraps calls errors.Wraps on the inner error.
// This will wrap in place via errors.ErrorWrapper if available
// If a nil is given it is a noop
func Wraps(errCode ErrorCode, msg string, args ...interface{}) ErrorCode {
	return wrapWith(errCode, errors.WrapsFn(msg, args...))
}

// WrapUser calls errors.Wrap or errors.Wrapf on the inner error.
// This will wrap in place via errors.ErrorWrapper if available
// If a nil is given it is a noop
func WrapUser(errCode UserCode, msg string, args ...interface{}) UserCode {
	return wrapG(wrapUserWith, errCode, msg, args...)
}

// WrapsUser calls errors.Wraps on the inner error.
// This uses the WrapError method of ErrorWrap
// If a nil is given it is a noop
func WrapsUser(errCode UserCode, msg string, args ...interface{}) UserCode {
	return wrapUserWith(errCode, errors.WrapsFn(msg, args...))
}

// WrapOp calls errors.Wrap or errors.Wrapf on the inner error.
// This will wrap in place via errors.ErrorWrapper if available
// If a nil is given it is a noop
func WrapOp(errCode OpCode, msg string, args ...interface{}) OpCode {
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
	return wrappedErrorCode{newWithError(errCode, wrap)}
}

func wrapUserWith(errCode UserCode, wrap func(error) error) UserCode {
	if errCode == nil {
		return errCode
	}
	ok := errors.WrapInPlace(errCode, wrap)
	if ok {
		return errCode
	}
	return wrappedUserCode{newWithError(errCode, wrap)}
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
		ErrorWrap: &errors.ErrorWrap{Err: wrapErr(errCode)},
	}
}

type wrappedErrorCode struct{ withError[ErrorCode] }

func (wec wrappedErrorCode) Code() Code {
	return wec.With.Code()
}

type wrappedUserCode struct{ withError[UserCode] }

func (wec wrappedUserCode) Code() Code {
	return wec.With.Code()
}

func (wec wrappedUserCode) GetUserMsg() string {
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
