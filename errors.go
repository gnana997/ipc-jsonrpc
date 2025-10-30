package jsonrpcipc

import "fmt"

// Standard JSON-RPC 2.0 error codes as defined in the specification.
// See: https://www.jsonrpc.org/specification#error_object
const (
	// ParseError indicates invalid JSON was received by the server.
	// An error occurred on the server while parsing the JSON text.
	ParseError = -32700

	// InvalidRequest indicates the JSON sent is not a valid Request object.
	InvalidRequest = -32600

	// MethodNotFound indicates the method does not exist or is not available.
	MethodNotFound = -32601

	// InvalidParams indicates invalid method parameter(s).
	InvalidParams = -32602

	// InternalError indicates an internal JSON-RPC error.
	InternalError = -32603

	// ServerErrorStart is the start of the reserved range for implementation-defined server errors.
	ServerErrorStart = -32099

	// ServerErrorEnd is the end of the reserved range for implementation-defined server errors.
	ServerErrorEnd = -32000
)

// Standard error messages for common error codes.
const (
	parseErrorMessage     = "Parse error"
	invalidRequestMessage = "Invalid Request"
	methodNotFoundMessage = "Method not found"
	invalidParamsMessage  = "Invalid params"
	internalErrorMessage  = "Internal error"
)

// NewError creates a new RPCError with the given code, message, and optional data.
//
// Example:
//
//	err := NewError(-32601, "Method not found", map[string]string{"method": "unknownMethod"})
func NewError(code int, message string, data interface{}) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewParseError creates a standard Parse Error (-32700).
// This error is returned when invalid JSON is received.
func NewParseError(data interface{}) *RPCError {
	return NewError(ParseError, parseErrorMessage, data)
}

// NewInvalidRequestError creates a standard Invalid Request Error (-32600).
// This error is returned when the JSON sent is not a valid Request object.
func NewInvalidRequestError(data interface{}) *RPCError {
	return NewError(InvalidRequest, invalidRequestMessage, data)
}

// NewMethodNotFoundError creates a standard Method Not Found Error (-32601).
// This error is returned when the requested method does not exist or is not available.
func NewMethodNotFoundError(method string) *RPCError {
	return NewError(MethodNotFound, methodNotFoundMessage, map[string]string{
		"method": method,
	})
}

// NewInvalidParamsError creates a standard Invalid Params Error (-32602).
// This error is returned when the method parameters are invalid.
func NewInvalidParamsError(data interface{}) *RPCError {
	return NewError(InvalidParams, invalidParamsMessage, data)
}

// NewInternalError creates a standard Internal Error (-32603).
// This error is returned when an internal error occurs in the server.
func NewInternalError(data interface{}) *RPCError {
	return NewError(InternalError, internalErrorMessage, data)
}

// WrapError wraps a Go error into a JSON-RPC error with the given code and message.
// The original error message is included in the data field.
//
// Example:
//
//	if err := doSomething(); err != nil {
//	    return nil, WrapError(InternalError, "Operation failed", err)
//	}
func WrapError(code int, message string, err error) *RPCError {
	if err == nil {
		return NewError(code, message, nil)
	}
	return NewError(code, message, err.Error())
}

// IsRPCError checks if the error is an RPCError.
func IsRPCError(err error) bool {
	_, ok := err.(*RPCError)
	return ok
}

// ToRPCError converts an error to an RPCError.
// If the error is already an RPCError, it is returned as-is.
// Otherwise, it is wrapped in an InternalError.
func ToRPCError(err error) *RPCError {
	if err == nil {
		return nil
	}

	if rpcErr, ok := err.(*RPCError); ok {
		return rpcErr
	}

	return NewInternalError(err.Error())
}

// ErrorFromCode returns a standard error for the given JSON-RPC error code.
// If the code is not a standard error code, it returns a generic error.
func ErrorFromCode(code int) *RPCError {
	switch code {
	case ParseError:
		return NewParseError(nil)
	case InvalidRequest:
		return NewInvalidRequestError(nil)
	case MethodNotFound:
		return NewMethodNotFoundError("")
	case InvalidParams:
		return NewInvalidParamsError(nil)
	case InternalError:
		return NewInternalError(nil)
	default:
		if code >= ServerErrorEnd && code <= ServerErrorStart {
			return NewError(code, "Server error", nil)
		}
		return NewError(code, fmt.Sprintf("Error code %d", code), nil)
	}
}
