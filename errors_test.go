package jsonrpcipc

import (
	"errors"
	"testing"
)

func TestNewError(t *testing.T) {
	code := -32000
	message := "Custom error"
	data := "test data"

	err := NewError(code, message, data)

	if err.Code != code {
		t.Errorf("Code = %d, want %d", err.Code, code)
	}
	if err.Message != message {
		t.Errorf("Message = %q, want %q", err.Message, message)
	}
	if err.Data != data {
		t.Errorf("Data = %v, want %v", err.Data, data)
	}
}

func TestStandardErrors(t *testing.T) {
	tests := []struct {
		name     string
		createFn func() *RPCError
		wantCode int
		wantMsg  string
		checkData bool
	}{
		{
			name:     "parse error",
			createFn: func() *RPCError { return NewParseError("test data") },
			wantCode: ParseError,
			wantMsg:  "Parse error",
			checkData: true,
		},
		{
			name:     "invalid request",
			createFn: func() *RPCError { return NewInvalidRequestError("test data") },
			wantCode: InvalidRequest,
			wantMsg:  "Invalid Request",
			checkData: true,
		},
		{
			name:     "method not found",
			createFn: func() *RPCError { return NewMethodNotFoundError("testMethod") },
			wantCode: MethodNotFound,
			wantMsg:  "Method not found",
			checkData: false, // NewMethodNotFoundError wraps method name in a map
		},
		{
			name:     "invalid params",
			createFn: func() *RPCError { return NewInvalidParamsError("test data") },
			wantCode: InvalidParams,
			wantMsg:  "Invalid params",
			checkData: true,
		},
		{
			name:     "internal error",
			createFn: func() *RPCError { return NewInternalError("test data") },
			wantCode: InternalError,
			wantMsg:  "Internal error",
			checkData: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createFn()

			if err.Code != tt.wantCode {
				t.Errorf("Code = %d, want %d", err.Code, tt.wantCode)
			}

			if err.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", err.Message, tt.wantMsg)
			}

			if tt.checkData && err.Data != "test data" {
				t.Errorf("Data = %v, want %v", err.Data, "test data")
			}
		})
	}
}

func TestRPCError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *RPCError
		want string
	}{
		{
			name: "error without data",
			err:  &RPCError{Code: -32600, Message: "Invalid Request"},
			want: "JSON-RPC error -32600: Invalid Request",
		},
		{
			name: "error with data",
			err:  &RPCError{Code: -32601, Message: "Method not found", Data: "testMethod"},
			want: "JSON-RPC error -32601: Method not found (data: testMethod)",
		},
		{
			name: "error with nil data",
			err:  &RPCError{Code: -32603, Message: "Internal error", Data: nil},
			want: "JSON-RPC error -32603: Internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsRPCError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "RPC error",
			err:  NewInvalidParamsError("test"),
			want: true,
		},
		{
			name: "standard error",
			err:  errors.New("regular error"),
			want: false,
		},
		{
			name: "wrapped RPC error",
			err:  WrapError(InternalError, "wrapped", NewInvalidRequestError("test")),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRPCError(tt.err)
			if got != tt.want {
				t.Errorf("IsRPCError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToRPCError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantNil  bool
	}{
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
		{
			name:     "RPC error",
			err:      NewInvalidParamsError("test"),
			wantCode: InvalidParams,
		},
		{
			name:     "standard error",
			err:      errors.New("regular error"),
			wantCode: InternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpcErr := ToRPCError(tt.err)

			if tt.wantNil {
				if rpcErr != nil {
					t.Errorf("Expected nil, got %v", rpcErr)
				}
				return
			}

			if rpcErr == nil {
				t.Fatal("Expected non-nil error, got nil")
			}

			if rpcErr.Code != tt.wantCode {
				t.Errorf("Code = %d, want %d", rpcErr.Code, tt.wantCode)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	code := InternalError
	message := "wrapped error"

	rpcErr := WrapError(code, message, originalErr)

	if rpcErr.Code != code {
		t.Errorf("Code = %d, want %d", rpcErr.Code, code)
	}

	if rpcErr.Message != message {
		t.Errorf("Message = %q, want %q", rpcErr.Message, message)
	}

	// Check that original error message is in data as string
	dataStr, ok := rpcErr.Data.(string)
	if !ok {
		t.Fatalf("Data type = %T, want string", rpcErr.Data)
	}

	if dataStr != originalErr.Error() {
		t.Errorf("Data = %q, want %q", dataStr, originalErr.Error())
	}
}

func TestErrorCodes(t *testing.T) {
	// Verify standard JSON-RPC error codes
	codes := map[string]int{
		"ParseError":     ParseError,
		"InvalidRequest": InvalidRequest,
		"MethodNotFound": MethodNotFound,
		"InvalidParams":  InvalidParams,
		"InternalError":  InternalError,
	}

	expectedCodes := map[string]int{
		"ParseError":     -32700,
		"InvalidRequest": -32600,
		"MethodNotFound": -32601,
		"InvalidParams":  -32602,
		"InternalError":  -32603,
	}

	for name, code := range codes {
		expected := expectedCodes[name]
		if code != expected {
			t.Errorf("%s = %d, want %d", name, code, expected)
		}
	}
}

func TestErrorFromCode(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		wantMsg  string
	}{
		{
			name:    "parse error",
			code:    ParseError,
			wantMsg: "Parse error",
		},
		{
			name:    "invalid request",
			code:    InvalidRequest,
			wantMsg: "Invalid Request",
		},
		{
			name:    "method not found",
			code:    MethodNotFound,
			wantMsg: "Method not found",
		},
		{
			name:    "invalid params",
			code:    InvalidParams,
			wantMsg: "Invalid params",
		},
		{
			name:    "internal error",
			code:    InternalError,
			wantMsg: "Internal error",
		},
		{
			name:    "unknown code",
			code:    -32000,
			wantMsg: "Server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.code, tt.wantMsg, nil)
			if err.Code != tt.code {
				t.Errorf("Code = %d, want %d", err.Code, tt.code)
			}
			if err.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", err.Message, tt.wantMsg)
			}
		})
	}
}
