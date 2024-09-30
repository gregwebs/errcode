// Copyright Greg Weber
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

import "github.com/gregwebs/errors"

// HasUserMsg retrieves a user message.
// The goal is to be able to show an error message that is tailored for end users and to hide extended error messages from the user.
//
// The user message should be retrieved with [UserMsg].
// [UserMsg] will check if a HasUserMsg interface exists.
// As an alternative to defining this interface yourself,
// you can use an existing struct that has already defined it.
// There is a wrapper struct [UsserMsgErrCode] via [NewUserMsg] or [WithUserMsg]
// and an embedded struct [EmbedUserMsg].
type HasUserMsg interface {
	GetUserMsg() string
}

// UserCode is used to ensure that an ErrorCode has a user message
type UserCode interface {
	ErrorCode
	HasUserMsg
}

// GetUserMsg will return a user message string if it exists.
// It checks recursively for the [HasUserMsg] interface.
// This function stops when it finds a user message: it will not combine them.
// If a user message is not found, it will return the zero value (empty) string.
func GetUserMsg(v interface{}) string {
	var msg string
	if hasMsg, ok := v.(HasUserMsg); ok {
		msg = hasMsg.GetUserMsg()
	} else if un, ok := v.(unwrapError); ok {
		return GetUserMsg(un.Unwrap())
	}
	return msg
}

// EmbedUserMsg is designed to be embedded into your existing error structs.
// It provides the HasUserMsg interface already, which can reduce your boilerplate.
type EmbedUserMsg struct{ Msg string }

// GetUserMsg satisfies the HasUserMsg interface
func (e EmbedUserMsg) GetUserMsg() string {
	return e.Msg
}

// UserMsgErrCode is an ErrorCode with a Msg field attached.
// This can be conveniently constructed with NewUserMsg and AddTo or WithUserMsg
// see the HasUserMsg documentation for alternatives.
type UserMsgErrCode struct {
	Msg string
	Err ErrorCode
}

// Unwrap satisfies the errors package Unwrap function
func (e UserMsgErrCode) Unwrap() error {
	return e.Err
}

// Error prefixes the user message to the underlying Err Error.
func (e UserMsgErrCode) Error() string {
	return e.Msg + ": " + e.Err.Error()
}

// GetUserMsg satisfies the [HasUserMsg] interface.
func (e UserMsgErrCode) GetUserMsg() string {
	return e.Msg
}

// Code returns the underlying Code of Err.
func (e UserMsgErrCode) Code() Code {
	return e.Err.Code()
}

var _ ErrorCode = (*UserMsgErrCode)(nil)   // assert implements interface
var _ HasUserMsg = (*UserMsgErrCode)(nil)  // assert implements interface
var _ unwrapError = (*UserMsgErrCode)(nil) // assert implements interface

// AddUserMsg is constructed by UserMsg. It allows method chaining with AddTo.
type AddUserMsg func(ErrorCode) UserCode

// AddTo adds the user message from UserMsg to the given ErrorCode
func (add AddUserMsg) AddTo(err ErrorCode) UserCode {
	return add(err)
}

// UserMsg adds a user message to an ErrorCode with AddTo.
// This converts the error to the type AddUserMsg.
//
//	userMsg := errcode.UserMsg("dont do that")
//	if start < obstable && obstacle < end  {
//		return userMsg.AddTo(PathBlocked{start, end, obstacle})
//	}
func UserMsg(msg string) AddUserMsg {
	return func(err ErrorCode) UserCode {
		return WithUserMsg(msg, err)
	}
}

// WithUserMsg creates a UserMsgErrCode
// Panics if msg is empty or err is nil.
func WithUserMsg(msg string, err ErrorCode) UserCode {
	if err == nil {
		return nil
	}
	return UserMsgErrCode{Msg: msg, Err: err}
}

type UserCodeWrap[Wrap UserCode] interface {
	UserCode
	Unwrapper[Wrap]
}

// wrappedUserCode is a convenience to maintain the UserCode type when wrapping errors
type wrappedUserCode[Wrapped UserCode] struct {
	Err      error
	UserCode Wrapped
}

// Code fulfills the ErrorCode interface
func (wrapped wrappedUserCode[Wrapped]) Code() Code {
	return wrapped.UserCode.Code()
}

// GetUserMsg fulfills the UserCode interface
func (wrapped wrappedUserCode[Wrapped]) GetUserMsg() string {
	return wrapped.UserCode.GetUserMsg()
}

// Error fulfills the Error interface
func (wrapped wrappedUserCode[Wrapped]) Error() string {
	return wrapped.Err.Error()
}

// Allow unwrapping
func (wrapped wrappedUserCode[Wrapped]) Unwrap() error {
	return wrapped.UserCode
}

func (wrapped wrappedUserCode[Wrapped]) Unwrapped() Wrapped {
	return wrapped.UserCode
}

// UserWrap is a convenience that calls errors.Wrap but still returns the UserCode interface
// If a nil UserCode is given it will be returned as nil
func UserWrap[EC UserCode](errCode EC, msg string) UserCodeWrap[EC] {
	err := errors.Wrap(errCode, msg)
	if err == nil {
		return nil
	}
	return wrappedUserCode[EC]{
		Err:      err,
		UserCode: errCode,
	}
}

// UserWrapf is a convenience that calls errors.Wrapf but still returns the UserCode interface
// If a nil UserCode is given it will be returned as nil
func UserWrapf[EC UserCode](errCode EC, msg string, args ...interface{}) UserCodeWrap[EC] {
	err := errors.Wrapf(errCode, msg, args...)
	if err == nil {
		return nil
	}
	return wrappedUserCode[EC]{
		Err:      err,
		UserCode: errCode,
	}
}

// UserWraps is a convenience that calls errors.Wraps but still returns the UserCode interface
// If a nil UserCode is given it will be returned as nil
func UserWraps[EC UserCode](errCode EC, msg string, args ...interface{}) UserCodeWrap[EC] {
	err := errors.Wraps(errCode, msg, args...)
	if err == nil {
		return nil
	}
	return wrappedUserCode[EC]{
		Err:      err,
		UserCode: errCode,
	}
}
