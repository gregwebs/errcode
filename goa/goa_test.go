package goa_test

import (
	"reflect"
	"testing"

	"github.com/gregwebs/errcode"
	"github.com/gregwebs/errcode/goa"
	"github.com/gregwebs/errors"
)

func TestErrorResponse(t *testing.T) {
	ecgPtr := goa.AsErrorCodeGoa(errcode.NewInternalErr(errors.New("goa test")))
	if ecgPtr == nil {
		t.Fatal("Expectd non-nil goa error")
	}
	ecg := *ecgPtr

	resSame := goa.ErrorResponse(ecg)
	// lint:ignore deepequalerrors
	if !reflect.DeepEqual(ecg, resSame) {
		t.Errorf("Expectd %T '%v' as goa error, got %T '%v'", ecg, ecg, resSame, resSame)
	}

	wrapped := errors.Wrap(ecg, "wrapped")
	resWrap := goa.ErrorResponse(wrapped)
	if wrapped.Error() != resWrap.Error() {
		t.Errorf("Expected %T '%v' as goa error, got %T '%v'", wrapped, wrapped, resWrap, resWrap)
	}
}
