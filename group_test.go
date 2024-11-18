package errcode_test

import (
	"reflect"
	"testing"

	"github.com/gregwebs/errcode"
	"github.com/gregwebs/errors"
)

type MultiErrors struct{ Multi []error }

func (e MultiErrors) Error() string {
	return "MultiErrors"
}

// Errors fullfills the ErrorGroup inteface
func (e MultiErrors) Errors() []error {
	return e.Multi
}

func (e MultiErrors) Unwrap() []error {
	return e.Multi
}

var _ error = MultiErrors{}
var _ errors.ErrorGroup = MultiErrors{}

func AssertLength[Any any](t *testing.T, slice []Any, expected int) {
	if len(slice) != expected {
		t.Helper()
		t.Errorf("expected length %d, got: %d. %v", expected, len(slice), slice)
	}

}

func TestErrorCodes(t *testing.T) {
	codes := errcode.ErrorCodes(nil)
	AssertLength(t, codes, 0)
	codes = errcode.ErrorCodes(errors.New("no codes"))
	AssertLength(t, codes, 0)
	codes = errcode.ErrorCodes(MinimalError{})
	AssertLength(t, codes, 1)
	code := errcode.NewInvalidInputErr(errors.New("inner invalid input"))
	codes = errcode.ErrorCodes(code)
	AssertLength(t, codes, 1)
	code = errcode.NewInvalidInputErr(MinimalError{})
	codes = errcode.ErrorCodes(code)
	AssertLength(t, codes, 1)
}

func TestErrorCodeChain(t *testing.T) {
	AssertCodeChain(t, errors.New("nil"), nil)

	code := MinimalError{}
	// AssertCodeChain(t, code, code)
	ann := errors.Wrap(code, "added annotation")
	AssertCodeChain(t, ann, errcode.ChainContext{Top: ann, ErrorCode: code})
	ann2 := errors.Wrap(ann, "another annotation")
	AssertCodeChain(t, ann2, errcode.ChainContext{Top: ann2, ErrorCode: code})

	code2 := MinimalError{}
	// horizontal composition
	multiCode := errcode.Combine(code, code2)
	annMultiCode := errors.Wrap(multiCode, "multi ann")
	AssertCodeChain(t, annMultiCode, errcode.ChainContext{Top: annMultiCode, ErrorCode: multiCode})
	multiErr := MultiErrors{Multi: []error{errors.New("ignore"), annMultiCode}}
	AssertCodeChain(t, multiErr, errcode.ChainContext{Top: multiErr, ErrorCode: errcode.ChainContext{Top: annMultiCode, ErrorCode: multiCode}})
	// TODO: vertical composition
}

func AssertCodeChain(t *testing.T, input error, expected errcode.ErrorCode) {
	t.Helper()
	output := errcode.CodeChain(input)
	if !reflect.DeepEqual(output, expected) {
		t.Errorf("ErrorCodeChain expected type %T value %#v%v\n				  got type %T value %#v%v", expected, expected, expected, output, output, output)
	}
}
