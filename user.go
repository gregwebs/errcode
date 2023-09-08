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

// HasUserMsg is an interface to retrieve a user message.
// The goal is to be able to show an error message that is either tailored for end users or to hide extended error messages from the client.
//
// GetUserMsg is defined, but generally the operation should be retrieved with UserMsg().
// UserMsg() will check if a HasUserMsg interface exists.
// As an alternative to defining this interface
// you can use an existing wrapper (UsserMsgErrCode via NewUserMsg) or embedding (EmbedUserMsg) that has already defined it.
type HasUserMsg interface {
	UserMsg() string
}

// UserMsg will return an user message string if it exists.
// It checks recursively for the HasUserMsg interface.
// Otherwise it will return the zero value (empty) string.
func UserMsg(v interface{}) string {
	var msg string
	if hasMsg, ok := v.(HasUserMsg); ok {
		msg = hasMsg.UserMsg()
	} else if un, ok := v.(unwrapper); ok {
		return UserMsg(un.Unwrap())
	}
	return msg
}

// EmbedUserMsg is designed to be embedded into your existing error structs.
// It provides the HasOperation interface already, which can reduce your boilerplate.
type EmbedUserMsg struct{ Msg string }

// UserMsg satisfies the HasUserMsg interface
func (e EmbedUserMsg) UserMsg() string {
	return e.Msg
}

// UserMsgErrCode is an ErrorCode with a Msg field attached.
// This can be conveniently constructed with Op() and AddTo() to record the operation information for the error.
// However, it isn't required to be used, see the HasUserMsg documentation for alternatives.
type UserMsgErrCode struct {
	Msg string
	Err ErrorCode
}

// Unwrap satisfies the errors package Unwrap function
func (e UserMsgErrCode) Unwrap() error {
	return e.Err
}

// Error prefixes the operation to the underlying Err Error.
func (e UserMsgErrCode) Error() string {
	return e.Msg + ": " + e.Err.Error()
}

// GetOperation satisfies the HasOperation interface.
func (e UserMsgErrCode) UserMsg() string {
	return e.Msg
}

// Code returns the underlying Code of Err.
func (e UserMsgErrCode) Code() Code {
	return e.Err.Code()
}

// GetClientData returns the ClientData of the underlying Err.
func (e UserMsgErrCode) GetClientData() interface{} {
	return ClientData(e.Err)
}

var _ ErrorCode = (*UserMsgErrCode)(nil)     // assert implements interface
var _ HasClientData = (*UserMsgErrCode)(nil) // assert implements interface
var _ HasUserMsg = (*UserMsgErrCode)(nil)    // assert implements interface
var _ unwrapper = (*UserMsgErrCode)(nil)     // assert implements interface

// AddUserMsg is constructed by AddUserMsg. It allows method chaining with AddTo.
type AddUserMsg func(ErrorCode) UserMsgErrCode

// AddTo adds the operation from Op to the ErrorCode
func (add AddUserMsg) AddTo(err ErrorCode) UserMsgErrCode {
	return add(err)
}

// Op adds an operation to an ErrorCode with AddTo.
// This converts the error to the type OpErrCode.
//
//	userMsg := errcode.NewUserMsg("dont do that")
//	if start < obstable && obstacle < end  {
//		return userMsg.AddTo(PathBlocked{start, end, obstacle})
//	}
func NewUserMsg(msg string) AddUserMsg {
	return func(err ErrorCode) UserMsgErrCode {
		if err == nil {
			panic("UserMsg error is nil")
		}
		return UserMsgErrCode{Msg: msg, Err: err}
	}
}
