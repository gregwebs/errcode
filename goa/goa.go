package goa

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gregwebs/errcode"
	goahttp "goa.design/goa/v3/http"
	goalib "goa.design/goa/v3/pkg"
)

type ErrorCodeGoa struct {
	errcode.JSONFormat
	errorCode errcode.ErrorCode
	err       error
}

// fulfill GOA expectation
func (ec *ErrorCodeGoa) GoaErrorName() string {
	return string(ec.errorCode.Code().CodeStr())
}

// fulfill http.Statuser
func (ec ErrorCodeGoa) StatusCode() int {
	httpCode := errcode.HTTPCode(ec.Code())
	if httpCode == nil {
		slog.Error("no HTTP Status Code", "code", ec.Code(), "error", ec.Error())
		return 400
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

func AsErrorCodeGoa(err error) *ErrorCodeGoa {
	if errCode := errcode.CodeChain(err); errCode != nil {
		return &ErrorCodeGoa{
			errorCode:  errCode,
			err:        err,
			JSONFormat: errcode.NewJSONFormat(errCode),
		}
	}

	return nil
}

func ErrorCodeToGoa(errCode errcode.ErrorCode) ErrorCodeGoa {
	return ErrorCodeGoa{
		errorCode:  errCode,
		err:        errCode,
		JSONFormat: errcode.NewJSONFormat(errCode),
	}
}

var codeCache map[string]errcode.Code

func goaErrorResponseToErrorCode(goaErr *goalib.ServiceError) errcode.Code {
	errResp := goahttp.ErrorResponse{
		Name:      goaErr.Name,
		ID:        goaErr.ID,
		Message:   goaErr.Message,
		Timeout:   goaErr.Timeout,
		Temporary: goaErr.Temporary,
		Fault:     goaErr.Fault,
	}

	switch errResp.Name {
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
		// GOA only gives the following HTTP codes
		switch errResp.StatusCode() {
		case http.StatusGatewayTimeout:
			return errcode.TimeoutGatewayCode
		case http.StatusRequestTimeout:
			return errcode.TimeoutRequestCode
		case http.StatusInternalServerError:
			return errcode.InternalCode
		case http.StatusServiceUnavailable:
			return errcode.UnavailableCode
		case http.StatusBadRequest:
			return errcode.InvalidInputCode
		}
		slog.Error("unexpected status code", "httpCode", errResp.StatusCode())
		if codeCache == nil {
			codeCache = make(map[string]errcode.Code)
		}
		code, okCode := codeCache[errResp.Name]
		if !okCode {
			code = errcode.NewCode(errcode.CodeStr(errResp.Name)).SetHTTP(errResp.StatusCode())
			codeCache[errResp.Name] = code
		}
		return code
	}
}

func ErrorResponse(err error) goahttp.Statuser {
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
		code := goaErrorResponseToErrorCode(goaErr)
		return ErrorCodeToGoa(errcode.NewCodedError(err, code))
	}

	// Use Goa default for all other error types
	return ErrorCodeToGoa(errcode.NewInternalErr(err))
}
