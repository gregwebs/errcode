// Copyright Greg Weber and PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0

package errcode

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

// userMsgErrCode is an ErrorCode with a Msg field attached.
// This can be conveniently constructed with NewUserMsg and AddTo or WithUserMsg
// see the HasUserMsg documentation for alternatives.
type userMsgErrCode struct {
	ErrorCode
	Msg string
}

// Unwrap satisfies the errors package Unwrap function
func (e userMsgErrCode) Unwrap() error {
	return e.ErrorCode
}

// Error prefixes the user message to the underlying Err Error.
func (e userMsgErrCode) Error() string {
	return e.Msg + ": " + e.ErrorCode.Error()
}

// GetUserMsg satisfies the [HasUserMsg] interface.
func (e userMsgErrCode) GetUserMsg() string {
	return e.Msg
}

var _ ErrorCode = (*userMsgErrCode)(nil)   // assert implements interface
var _ HasUserMsg = (*userMsgErrCode)(nil)  // assert implements interface
var _ unwrapError = (*userMsgErrCode)(nil) // assert implements interface

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

// WithUserMsg creates a UserCode
// Panics if msg is empty or err is nil.
func WithUserMsg(msg string, err ErrorCode) UserCode {
	if err == nil {
		return nil
	}
	return userMsgErrCode{Msg: msg, ErrorCode: err}
}
