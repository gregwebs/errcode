package goa_test

import (
	"encoding/json"
	"testing"

	"github.com/gregwebs/errcode"
	"github.com/gregwebs/errcode/goa"
	"github.com/gregwebs/errors"
	goalib "goa.design/goa/v3/pkg"
)

func TestErrorResponse(t *testing.T) {
	ecgPtr := goa.AsErrorCodeGoa(errcode.NewInternalErr(errors.New("goa test")))
	if ecgPtr == nil {
		t.Fatal("Expectd non-nil goa error")
	}
	ecg := *ecgPtr

	resSame := goa.ErrorResponse(ecg)
	if ecg != resSame {
		t.Errorf("Expectd %T '%v' as goa error, got %T '%v'", ecg, ecg, resSame, resSame)
	}

	wrapped := errors.Wrap(ecg, "wrapped")
	resWrap := goa.ErrorResponse(wrapped)
	if wrapped.Error() != resWrap.Error() {
		t.Errorf("Expected %T '%v' as goa error, got %T '%v'", wrapped, wrapped, resWrap, resWrap)
	}
	jsonBytes, err := json.Marshal(resWrap)
	if err != nil {
		t.Fatalf("expected json marshal success, got %v", err)
	}
	expectedJSON := `{"code":"internal","msg":"wrapped: goa test","data":{}}`
	if string(jsonBytes) != expectedJSON {
		t.Fatalf("expected %s, got %s", expectedJSON, string(jsonBytes))
	}
}

func TestServiceErrorToErrorCode(t *testing.T) {
	err := errors.New("test err")
	svcErr := goalib.NewServiceError(err, "Name", false, false, false)
	gotCode := goa.ServiceErrorToErrorCode(svcErr)
	got := gotCode.Code().CodeStr()
	expected := "input.Name"
	if expected != string(got) {
		t.Errorf("expected %s but got %s", expected, got)
	}
	gotMsg := errcode.GetUserMsg(goa.AsErrorCodeGoa(errcode.WithUserMsg("user", gotCode)))
	if gotMsg != "user" {
		t.Errorf("expected user, got %s", gotMsg)
	}
	/*
		jsonBytes, err := json.Marshal(gotCode)
		if err != nil {
			t.Fatalf("expected json marshal success, got %v", err)
		}
		expectedJSON := `{"code":"input.Name","msg":"test err","data":{"Name":"Name","ID":"cFMCxpn6","Field":null,"Message":"test err","Timeout":false,"Temporary":false,"Fault":false}}`
		if string(jsonBytes) != expectedJSON {
			t.Fatalf("expected {}, got %s", string(jsonBytes))
		}
	*/

	svcErr = goalib.NewServiceError(err, "Name", true, false, false)
	got = goa.ServiceErrorToErrorCode(svcErr).Code().CodeStr()
	expected = "timeout.request"
	if expected != string(got) {
		t.Errorf("expected %s but got %s", expected, got)
	}
}
