// Package jsonrpcipc provides a JSON-RPC 2.0 server over IPC
// (Unix sockets and Windows named pipes).
//
// This package implements the JSON-RPC 2.0 specification with line-delimited
// JSON encoding for communication over IPC transports. It is designed to be
// compatible with the node-ipc-jsonrpc Node.js package.
package jsonrpcipc

import (
	"encoding/json"
	"fmt"
)

// Request represents a JSON-RPC 2.0 request message.
//
// Example:
//
//	{
//	  "jsonrpc": "2.0",
//	  "method": "search",
//	  "params": {"query": "test", "limit": 10},
//	  "id": 1
//	}
type Request struct {
	JSONRPC string          `json:"jsonrpc,omitempty"` // "2.0" (optional but recommended)
	Method  string          `json:"method"`            // Method name to invoke
	Params  json.RawMessage `json:"params,omitempty"`  // Method parameters (can be object or array)
	ID      interface{}     `json:"id"`                // Request ID (string or number)
}

// Response represents a JSON-RPC 2.0 success response.
//
// Example:
//
//	{
//	  "jsonrpc": "2.0",
//	  "result": {"data": "value"},
//	  "id": 1
//	}
type Response struct {
	JSONRPC string      `json:"jsonrpc,omitempty"` // "2.0" (optional)
	Result  interface{} `json:"result"`            // Result data
	ID      interface{} `json:"id"`                // Request ID (must match request)
}

// ErrorResponse represents a JSON-RPC 2.0 error response.
//
// Example:
//
//	{
//	  "jsonrpc": "2.0",
//	  "error": {
//	    "code": -32601,
//	    "message": "Method not found",
//	    "data": {"method": "unknownMethod"}
//	  },
//	  "id": 1
//	}
type ErrorResponse struct {
	JSONRPC string      `json:"jsonrpc,omitempty"` // "2.0" (optional)
	Error   *RPCError   `json:"error"`             // Error object
	ID      interface{} `json:"id"`                // Request ID (must match request, null if parse error)
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int         `json:"code"`           // Error code
	Message string      `json:"message"`        // Error message
	Data    interface{} `json:"data,omitempty"` // Additional error data (optional)
}

// Error implements the error interface for RPCError.
func (e *RPCError) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("JSON-RPC error %d: %s (data: %v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// Notification represents a JSON-RPC 2.0 notification.
// Notifications are requests without an ID field and don't expect a response.
//
// Example:
//
//	{
//	  "jsonrpc": "2.0",
//	  "method": "progress",
//	  "params": {"percentage": 50}
//	}
type Notification struct {
	JSONRPC string      `json:"jsonrpc,omitempty"` // "2.0" (optional)
	Method  string      `json:"method"`            // Notification method name
	Params  interface{} `json:"params,omitempty"`  // Notification parameters
}

// Message is a union type that can represent any JSON-RPC message.
// Used for parsing incoming messages when the type is unknown.
type Message struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`     // Present in requests/responses, absent in notifications
	Result  json.RawMessage `json:"result,omitempty"` // Present in success responses
	Error   *RPCError       `json:"error,omitempty"`  // Present in error responses
}

// IsRequest returns true if the message is a request (has method and ID).
func (m *Message) IsRequest() bool {
	return m.Method != "" && m.ID != nil
}

// IsNotification returns true if the message is a notification (has method but no ID).
func (m *Message) IsNotification() bool {
	return m.Method != "" && m.ID == nil
}

// IsResponse returns true if the message is a response (has ID but no method).
func (m *Message) IsResponse() bool {
	return m.ID != nil && m.Method == ""
}

// IsErrorResponse returns true if the message is an error response.
func (m *Message) IsErrorResponse() bool {
	return m.Error != nil && m.ID != nil
}

// IsSuccessResponse returns true if the message is a success response.
func (m *Message) IsSuccessResponse() bool {
	return m.Result != nil && m.Error == nil && m.ID != nil
}

// ToRequest converts the message to a Request.
// Returns an error if the message is not a valid request.
func (m *Message) ToRequest() (*Request, error) {
	if !m.IsRequest() {
		return nil, fmt.Errorf("message is not a request")
	}
	return &Request{
		JSONRPC: m.JSONRPC,
		Method:  m.Method,
		Params:  m.Params,
		ID:      m.ID,
	}, nil
}

// ToNotification converts the message to a Notification.
// Returns an error if the message is not a valid notification.
func (m *Message) ToNotification() (*Notification, error) {
	if !m.IsNotification() {
		return nil, fmt.Errorf("message is not a notification")
	}

	var params interface{}
	if len(m.Params) > 0 {
		if err := json.Unmarshal(m.Params, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
	}

	return &Notification{
		JSONRPC: m.JSONRPC,
		Method:  m.Method,
		Params:  params,
	}, nil
}
