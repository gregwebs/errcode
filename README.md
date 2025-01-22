## About Error codes

Error codes reliably encode important error information such as the HTTP status code.
They can also encode specific error types so that clients across network boundaries can understand errors in detail.
Error monitoring systems can reliably understand (dedup) what errors are occurring and metrics can easily be generated.

Returning an ErrorCode helps modularize code- a function that is not written in the context of an (HTTP) handler can encode an HTTP code.

errcode supports hierachy and associating to metadata such as HTTP codes.
The common simple form of this is to properly attach an HTTP code to an error.
When clients need to react to specific errors, both the HTTP code and a more specific error code can be associated to the error.

## Status

This library has been used in production for years.
The core types and API have not changed but there is ongoing experimentation with newer APIs.

# errcode overview

This package extends go errors via interfaces to have error codes.

```go
type ErrorCode interface {
	error
	Code() Code
}
```

There are existing generic error codes and constructors for them such as `NewNotFoundErr`.

A Code is a string that can be placed in a hierarchy:

```go
type Code struct {
	codeStr CodeStr
	Parent  *Code
}

type CodeStr string
```

The package also provides `UserCode` designed to provide a user-facing message for end users
rather than technical error messages.


```go
type UserCode interface {
	ErrorCode
	HasUserMsg
}

type HasUserMsg interface {
	GetUserMsg() string
}
```

A UserCode can be created with `errcode.WithUserMsg` or `errcode.UserMsg`.

See the [go docs](https://godoc.org/github.com/gregwebs/errcode) for extensive API documentation.

## Features

* structured error representation
* Uses the Unwrap model where errors can be annotated and the underlying code can be unwrapped
* Internal errors show a stack trace but others don't.
* Operation annotation. This concept is [explained here](https://commandcenter.blogspot.com/2017/12/error-handling-in-upspin.html).
* Works for multiple errors when the Errors() interface is used. See the `Combine` function for constructing multiple error codes.
* Extensible metadata. See how SetHTTPCode is implemented.
* Integration with existing error codes
  * HTTP
  * GRPC (provided by separate grpc package)


## Comparison

There are other packages that add error code capabilities to errors.
However, most use a single underlying error struct.
A code or other annotation is a field on that struct.

errcode instead follows the model of wrapping and interfaces.
You can adapt your own error structures to fulfill the ErrorCode interfaces.
Additional features (for example annotating the operation) are done via wrapping.

This design makes errcode highly extensible, inter-operable, and structure preserving (of the original error).
It is easy to gradually introduce errcode into a project.


## Code Examples

### Using a built-in ErrorCode:

For code doing HTTP or GRPC and with simple client needs, the built in error codes may suffice.

``` go
err := errors.New("not found")
errCode := errcode.NewNotFoundErr(err)
```

## Sending an error code to a client

``` go
// Given just a type of error, give an error code to a client if it is present
if errCode := errcode.CodeChain(err); errCode != nil {
	// Setting the code in a header
	// Our error code inherits StatusBadRequest from its parent code "state"
	w.Header().Set("X-Error-Code", errCode.Code().CodeStr().String())

	// Using a JSON body.
	// If an unwrapped error defines HasClientData then that data will be written as part of the JSON response.
	// type HasClientData interface { GetClientData() interface{} }
	rd.JSON(w, errCode.Code().HTTPCode(), errcode.NewJSONFormat(errCode))
}
```

### Creating a custom ErrorCode:

``` go
// First define a normal error type
type PathBlocked struct {
	start     uint64 `json:"start"`
	end       uint64 `json:"end"`
	obstacle  uint64 `json:"obstacle"`
}

func (e PathBlocked) Error() string {
	return fmt.Sprintf("The path %d -> %d has obstacle %d", e.start, e.end, e.obstacle)
}

// Define a code. These often get re-used between different errors.
// Note that codes use a hierarchy to share metadata.
// This code is a child of the "state" code.
var PathBlockedCode = errcode.StateCode.Child("state.blocked")

// Now attach the code to your custom error type.
func (e PathBlocked) Code() Code {
	return PathBlockedCode
}

var _ ErrorCode = (*PathBlocked)(nil)  // assert implements the ErrorCode interface
```

Let see a usage site. This example will include an annotation concept of "operation".

``` go
func moveX(start, end, obstacle) error {}
	op := errcode.Op("path.move.x")

	if start < obstable && obstacle < end  {
		return op.AddTo(PathBlocked{start, end, obstacle})
	}
}
```
