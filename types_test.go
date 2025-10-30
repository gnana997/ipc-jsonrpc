package jsonrpcipc

import (
	"encoding/json"
	"testing"
)

func TestMessage_TypeDetection(t *testing.T) {
	tests := []struct {
		name          string
		message       *Message
		isRequest     bool
		isNotification bool
		isResponse    bool
	}{
		{
			name: "request with ID",
			message: &Message{
				Method: "test",
				ID:     1.0,
			},
			isRequest:     true,
			isNotification: false,
			isResponse:    false,
		},
		{
			name: "notification without ID",
			message: &Message{
				Method: "notify",
			},
			isRequest:     false,
			isNotification: true,
			isResponse:    false,
		},
		{
			name: "success response",
			message: &Message{
				Result: json.RawMessage(`"success"`),
				ID:     1.0,
			},
			isRequest:     false,
			isNotification: false,
			isResponse:    true,
		},
		{
			name: "error response",
			message: &Message{
				Error: &RPCError{Code: -32600, Message: "error"},
				ID:    1.0,
			},
			isRequest:     false,
			isNotification: false,
			isResponse:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.message.IsRequest(); got != tt.isRequest {
				t.Errorf("IsRequest() = %v, want %v", got, tt.isRequest)
			}
			if got := tt.message.IsNotification(); got != tt.isNotification {
				t.Errorf("IsNotification() = %v, want %v", got, tt.isNotification)
			}
			if got := tt.message.IsResponse(); got != tt.isResponse {
				t.Errorf("IsResponse() = %v, want %v", got, tt.isResponse)
			}
		})
	}
}

func TestMessage_ToRequest(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
		wantErr bool
	}{
		{
			name: "valid request",
			message: &Message{
				JSONRPC: "2.0",
				Method:  "test",
				Params:  json.RawMessage(`{"key":"value"}`),
				ID:      1.0,
			},
			wantErr: false,
		},
		{
			name: "invalid - no method",
			message: &Message{
				JSONRPC: "2.0",
				ID:      1.0,
			},
			wantErr: true,
		},
		{
			name: "invalid - no ID",
			message: &Message{
				JSONRPC: "2.0",
				Method:  "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := tt.message.ToRequest()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if req.Method != tt.message.Method {
				t.Errorf("Method = %q, want %q", req.Method, tt.message.Method)
			}
		})
	}
}

func TestMessage_ToNotification(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
		wantErr bool
	}{
		{
			name: "valid notification",
			message: &Message{
				JSONRPC: "2.0",
				Method:  "notify",
				Params:  json.RawMessage(`{"data":"test"}`),
			},
			wantErr: false,
		},
		{
			name: "invalid - no method",
			message: &Message{
				JSONRPC: "2.0",
			},
			wantErr: true,
		},
		{
			name: "invalid - has ID (should be request)",
			message: &Message{
				JSONRPC: "2.0",
				Method:  "test",
				ID:      1.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notif, err := tt.message.ToNotification()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if notif.Method != tt.message.Method {
				t.Errorf("Method = %q, want %q", notif.Method, tt.message.Method)
			}
		})
	}
}

func TestMessage_IDTypes(t *testing.T) {
	tests := []struct {
		name string
		id   interface{}
	}{
		{"string ID", "test-123"},
		{"integer ID", 42},
		{"float ID", 3.14},
		{"null ID", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{
				JSONRPC: "2.0",
				Method:  "test",
				ID:      tt.id,
			}

			data, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			var decoded Request
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Compare IDs (accounting for JSON number conversion)
			if !compareIDs(decoded.ID, tt.id) {
				t.Errorf("ID mismatch: got %v (%T), want %v (%T)", decoded.ID, decoded.ID, tt.id, tt.id)
			}
		})
	}
}

func TestRequest_Marshal(t *testing.T) {
	req := &Request{
		JSONRPC: "2.0",
		Method:  "test",
		Params:  json.RawMessage(`{"key":"value"}`),
		ID:      1,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.JSONRPC != req.JSONRPC {
		t.Errorf("JSONRPC = %q, want %q", decoded.JSONRPC, req.JSONRPC)
	}
	if decoded.Method != req.Method {
		t.Errorf("Method = %q, want %q", decoded.Method, req.Method)
	}
}

func TestResponse_Marshal(t *testing.T) {
	resp := &Response{
		JSONRPC: "2.0",
		Result:  "success",
		ID:      1,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.JSONRPC != resp.JSONRPC {
		t.Errorf("JSONRPC = %q, want %q", decoded.JSONRPC, resp.JSONRPC)
	}
	if decoded.Result != resp.Result {
		t.Errorf("Result = %v, want %v", decoded.Result, resp.Result)
	}
}

func TestErrorResponse_Marshal(t *testing.T) {
	errResp := &ErrorResponse{
		JSONRPC: "2.0",
		Error:   &RPCError{Code: -32600, Message: "Invalid Request"},
		ID:      1,
	}

	data, err := json.Marshal(errResp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded ErrorResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Error.Code != errResp.Error.Code {
		t.Errorf("Error.Code = %d, want %d", decoded.Error.Code, errResp.Error.Code)
	}
	if decoded.Error.Message != errResp.Error.Message {
		t.Errorf("Error.Message = %q, want %q", decoded.Error.Message, errResp.Error.Message)
	}
}

func TestNotification_Marshal(t *testing.T) {
	notif := &Notification{
		JSONRPC: "2.0",
		Method:  "notify",
		Params:  map[string]string{"key": "value"},
	}

	data, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Notification
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.JSONRPC != notif.JSONRPC {
		t.Errorf("JSONRPC = %q, want %q", decoded.JSONRPC, notif.JSONRPC)
	}
	if decoded.Method != notif.Method {
		t.Errorf("Method = %q, want %q", decoded.Method, notif.Method)
	}
}

func TestMessage_IsSuccessResponse(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
		want bool
	}{
		{
			name: "success response",
			msg: &Message{
				Result: json.RawMessage(`"success"`),
				ID:     1.0,
			},
			want: true,
		},
		{
			name: "error response",
			msg: &Message{
				Error: &RPCError{Code: -32600, Message: "error"},
				ID:    1.0,
			},
			want: false,
		},
		{
			name: "request",
			msg: &Message{
				Method: "test",
				ID:     1.0,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.IsSuccessResponse(); got != tt.want {
				t.Errorf("IsSuccessResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_IsErrorResponse(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
		want bool
	}{
		{
			name: "error response",
			msg: &Message{
				Error: &RPCError{Code: -32600, Message: "error"},
				ID:    1.0,
			},
			want: true,
		},
		{
			name: "success response",
			msg: &Message{
				Result: json.RawMessage(`"success"`),
				ID:     1.0,
			},
			want: false,
		},
		{
			name: "request",
			msg: &Message{
				Method: "test",
				ID:     1.0,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.IsErrorResponse(); got != tt.want {
				t.Errorf("IsErrorResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}
