package errcode_test

import (
	"testing"

	"github.com/gregwebs/errcode"
	"github.com/gregwebs/errors"
)

func TestErrorWrapperNil(t *testing.T) {
	// Don't panic!
	if errcode.Wrap(errcode.ErrorCode(nil), "wrapped") != nil {
		t.Errorf("Wrap nil should be nil")
	}
	if errcode.Wrap(errcode.ErrorCode(nil), "wrapped") != nil {
		t.Errorf("Wrapf nil should be nil")
	}
	if errcode.Wraps(errcode.ErrorCode(nil), "wrapped") != nil {
		t.Errorf("Wraps nil should be nil")
	}
}

func TestErrorWrapperFunctions(t *testing.T) {
	underlying := errors.New("underlying")

	{
		bad := errcode.NewBadRequestErr(underlying)
		AssertCode(t, bad, errcode.InvalidInputCode.CodeStr())
		coded := errcode.Wrap(bad, "wrapped")
		AssertCode(t, coded, errcode.InvalidInputCode.CodeStr())
		if errMsg := coded.Error(); errMsg != "wrapped: underlying" {
			t.Errorf("Wrap unexpected: %s", errMsg)
		}
		doubleUnwrap := errors.Unwrap(errors.Unwrap(coded))
		if doubleUnwrap.Error() != underlying.Error() {
			t.Errorf("bad unwrap: %s", doubleUnwrap.Error())
		}
	}

	{
		bad := errcode.NewBadRequestErr(underlying)
		AssertCode(t, bad, errcode.InvalidInputCode.CodeStr())
		coded := errcode.Wrap(bad, "wrapped %s", "arg")
		AssertCode(t, coded, errcode.InvalidInputCode.CodeStr())
		if errMsg := coded.Error(); errMsg != "wrapped arg: underlying" {
			t.Errorf("Wrap unexpected: %s", errMsg)
		}
		doubleUnwrap := errors.Unwrap(errors.Unwrap(coded))
		if doubleUnwrap.Error() != underlying.Error() {
			t.Errorf("bad unwrap: %s", doubleUnwrap.Error())
		}
	}

	{
		bad := errcode.NewBadRequestErr(underlying)
		AssertCode(t, bad, errcode.InvalidInputCode.CodeStr())
		coded := errcode.Wraps(bad, "wrapped", "arg", 1)
		AssertCode(t, coded, errcode.InvalidInputCode.CodeStr())
		if errMsg := coded.Error(); errMsg != "wrapped arg=1: underlying" {
			t.Errorf("Wrap unexpected: %s", errMsg)
		}
		doubleUnwrap := errors.Unwrap(errors.Unwrap(coded))
		if doubleUnwrap.Error() != underlying.Error() {
			t.Errorf("bad unwrap: %s", doubleUnwrap.Error())
		}
	}
}

func TestUserWrapperFunctions(t *testing.T) {
	underlying := errors.New("underlying")

	{
		coded := errcode.WithUserMsg("user",
			errcode.NewBadRequestErr(underlying),
		)
		AssertCode(t, coded, errcode.InvalidInputCode.CodeStr())
		coded = errcode.WrapUser(coded, "wrapped")
		AssertCode(t, coded, errcode.InvalidInputCode.CodeStr())
		if errMsg := coded.Error(); errMsg != "user: wrapped: underlying" {
			t.Errorf("Wrap unexpected: %s", errMsg)
		}
		tripleUnwrap := errors.Unwrap(errors.Unwrap(errors.Unwrap(coded)))
		if tripleUnwrap.Error() != underlying.Error() {
			t.Errorf("bad unwrap: %s", tripleUnwrap.Error())
		}
	}

	{
		coded := errcode.WithUserMsg("user",
			errcode.NewBadRequestErr(underlying),
		)
		AssertCode(t, coded, errcode.InvalidInputCode.CodeStr())
		coded = errcode.WrapUser(coded, "wrapped %s", "arg")
		AssertCode(t, coded, errcode.InvalidInputCode.CodeStr())
		if errMsg := coded.Error(); errMsg != "user: wrapped arg: underlying" {
			t.Errorf("Wrap unexpected: %s", errMsg)
		}
		tripleUnwrap := errors.Unwrap(errors.Unwrap(errors.Unwrap(coded)))
		if tripleUnwrap.Error() != underlying.Error() {
			t.Errorf("bad unwrap: %s", tripleUnwrap.Error())
		}
	}

	{
		coded := errcode.WithUserMsg("user",
			errcode.NewBadRequestErr(underlying),
		)
		AssertCode(t, coded, errcode.InvalidInputCode.CodeStr())
		coded = errcode.WrapsUser(coded, "wrapped", "arg", 1)
		AssertCode(t, coded, errcode.InvalidInputCode.CodeStr())
		if errMsg := coded.Error(); errMsg != "user: wrapped arg=1: underlying" {
			t.Errorf("Wrap unexpected: %s", errMsg)
		}
		tripleUnwrap := errors.Unwrap(errors.Unwrap(errors.Unwrap(coded)))
		if tripleUnwrap.Error() != underlying.Error() {
			t.Errorf("bad unwrap: %s", tripleUnwrap.Error())
		}
	}
}

func TestManyWrappings(t *testing.T) {
	underlying := errors.New("underlying")

	{
		user := errcode.WithUserMsg("user",
			errcode.NewBadRequestErr(underlying),
		)
		opCode := errcode.Op("op").AddTo(user)
		AssertCode(t, opCode, errcode.InvalidInputCode.CodeStr())
		coded := errcode.WrapOp(opCode, "wrapped")
		AssertCode(t, coded, errcode.InvalidInputCode.CodeStr())
		if errMsg := coded.Error(); errMsg != "op: user: wrapped: underlying" {
			t.Errorf("Wrap unexpected: %s", errMsg)
		}
		tripleUnwrap := errors.Unwrap(errors.Unwrap(errors.Unwrap(errors.Unwrap(coded))))
		if tripleUnwrap.Error() != underlying.Error() {
			t.Errorf("bad unwrap: %s", tripleUnwrap.Error())
		}
	}
}

func TestWrapNotInPlace(t *testing.T) {
	user := errcode.WithUserMsg("user",
		MinimalError{},
	)
	op := errcode.Op("op").AddTo(user)
	AssertCode(t, op, codeString)
	coded := errcode.Wrap(op, "wrapped")
	AssertCode(t, coded, codeString)
	if errMsg := coded.Error(); errMsg != "wrapped: op: user: error" {
		t.Errorf("Wrap unexpected: %s", errMsg)
	}
	tripleUnwrap := errors.Unwrap(errors.Unwrap(errors.Unwrap(errors.Unwrap(coded))))
	if tripleUnwrap.Error() != (MinimalError{}).Error() {
		t.Errorf("bad unwrap: %s", tripleUnwrap.Error())
	}
}
