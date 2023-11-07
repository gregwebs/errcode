package goa

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gregwebs/errcode"
	goahttp "goa.design/goa/v3/http"
	goalib "goa.design/goa/v3/pkg"
)

type ErrorCodeGoa struct {
	errorCode errcode.ErrorCode
	err       error
}

// fulfill GOA expectation
func (ec ErrorCodeGoa) GoaErrorName() string {
	return string(ec.errorCode.Code().CodeStr())
}

// fulfill http.Statuser
func (ec ErrorCodeGoa) StatusCode() int {
	httpCode := errcode.HTTPCode(ec.Code())
	if httpCode == nil {
		slog.Error("no HTTP Status Code", "code", ec.Code(), "error", ec.Error())
		return http.StatusBadRequest
	}
	return *httpCode
}

func (ec ErrorCodeGoa) Code() errcode.Code {
	return ec.errorCode.Code()
}

func (ec ErrorCodeGoa) Error() string {
	return ec.err.Error()
}

func (ec ErrorCodeGoa) Unwrap() error {
	return ec.err
}

func (ec ErrorCodeGoa) MarshalJSON() ([]byte, error) {
	return json.Marshal(errcode.NewJSONFormat(ec.errorCode))
}

func AsErrorCodeGoa(err error) *ErrorCodeGoa {
	if ecg, ok := err.(ErrorCodeGoa); ok {
		return &ecg
	}
	if ecg, ok := err.(*ErrorCodeGoa); ok {
		return ecg
	}
	if errCode := errcode.CodeChain(err); errCode != nil {
		return &ErrorCodeGoa{
			errorCode: errCode,
			err:       err,
		}
	}

	return nil
}

func ErrorCodeToGoa(errCode errcode.ErrorCode) ErrorCodeGoa {
	return ErrorCodeGoa{
		errorCode: errCode,
		err:       errCode,
	}
}

var codeCache map[string]errcode.Code

func serviceErrorToHttpErr(goaErr *goalib.ServiceError) *goahttp.ErrorResponse {
	return &goahttp.ErrorResponse{
		Name:      goaErr.Name,
		ID:        goaErr.ID,
		Message:   goaErr.Message,
		Timeout:   goaErr.Timeout,
		Temporary: goaErr.Temporary,
		Fault:     goaErr.Fault,
	}
}

func serviceErrorToCode(goaErr *goalib.ServiceError) errcode.Code {
	switch goaErr.Name {
	case "missing_payload":
		return errcode.InvalidInputCode
	case "decode_payload":
		return errcode.InvalidInputCode
	case "invalid_field_type":
		return errcode.InvalidInputCode
	case "missing_field":
		return errcode.InvalidInputCode
	case "invalid_enum_value":
		return errcode.InvalidInputCode
	case "invalid_format":
		return errcode.InvalidInputCode
	case "invalid_pattern":
		return errcode.InvalidInputCode
	case "invalid_range":
		return errcode.InvalidInputCode
	case "invalid_length":
		return errcode.InvalidInputCode
	default:
		statusCode := serviceErrorToHttpErr(goaErr).StatusCode()
		var parentCode *errcode.Code
		// GOA only gives the following HTTP codes
		switch statusCode {
		case http.StatusGatewayTimeout:
			return errcode.TimeoutGatewayCode
		case http.StatusRequestTimeout:
			return errcode.TimeoutRequestCode
		case http.StatusInternalServerError:
			return errcode.InternalCode
		case http.StatusServiceUnavailable:
			return errcode.UnavailableCode
		case http.StatusBadRequest:
			parentCode = &errcode.InvalidInputCode
		}
		if codeCache == nil {
			codeCache = make(map[string]errcode.Code)
		}
		code, okCode := codeCache[goaErr.Name]
		if !okCode {
			codeStr := errcode.CodeStr(goaErr.Name)
			code = errcode.NewCode(codeStr)
			if parentCode != nil {
				code = parentCode.Child(codeStr)
			}
			code.SetHTTP(statusCode)
			codeCache[goaErr.Name] = code
		}
		return code
	}
}

func ErrorResponse(err error) ErrorCodeGoa {
	if ecg := AsErrorCodeGoa(err); ecg != nil {
		return *ecg
	}

	// Allow wrapping a ServiceError with text
	// The wrapped Error() text will show up in the Message field
	var goaErr *goalib.ServiceError = &goalib.ServiceError{}
	if ok := errors.As(err, &goaErr); ok {
		if _, ok := err.(*goalib.ServiceError); !ok {
			goaErr.Message = err.Error()
		}
		return ServiceErrorToErrorCode(goaErr)
	}

	// Use Goa default for all other error types
	return ErrorCodeToGoa(errcode.NewInternalErr(err))
}

func ServiceErrorToErrorCode(err *goalib.ServiceError) ErrorCodeGoa {
	code := serviceErrorToCode(err)
	var errorForCode error = err

	// adjust GOA error mesages to be user readable
	if errcode.GetUserMsg(err) == "" {
		switch err.Name {
		case "invalid_pattern":
			errorForCode = PatternErr{err: err}
		}
	}
	var errCode errcode.ErrorCode = errcode.NewCodedError(errorForCode, code)
	return ErrorCodeToGoa(errCode)
}

type PatternErr struct {
	err *goalib.ServiceError
}

func (pe PatternErr) Unwrap() error {
	return pe.err
}

func (pe PatternErr) Error() string {
	return pe.err.Error()
}

func (pe PatternErr) GetUserMsg() string {
	msg := strings.TrimPrefix(pe.err.Message, "body.")
	msg = strings.Split(msg, " must match ")[0]
	if msg != pe.err.Message {
		msg = msg + " is invalid"
		return strings.ReplaceAll(msg, "  ", " ")
	}
	return ""
}

func (pe PatternErr) GetClientData() interface{} {
	var value string
	valueSplit := strings.Split(pe.err.Message, " but got value ")
	if len(valueSplit) == 2 {
		value = valueSplit[1]
		after, found := strings.CutPrefix(value, `"`)
		if found {
			value, _ = strings.CutSuffix(after, `"`)
		}
	}
	var field string
	if pe.err.Field != nil {
		field = strings.TrimPrefix(*pe.err.Field, "body.")
	}
	return PatternErrClientData{
		ID:          pe.err.ID,
		Name:        pe.err.Name,
		Field:       field,
		Value:       value,
		FullMessage: pe.err.Message,
	}
}

// var _ errcode.HasClientData = PatternErr{}

type PatternErrClientData struct {
	ID          string
	Name        string
	Field       string
	Value       string
	FullMessage string
}
