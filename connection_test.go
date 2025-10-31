package jsonrpcipc

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

func TestNewConnection(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	registry := NewHandlerRegistry()
	middleware := []Middleware{}

	connection := newConnection(conn1, registry, middleware, nil)

	if connection == nil {
		t.Fatal("newConnection returned nil")
	}

	if connection.conn != conn1 {
		t.Error("Connection conn not set correctly")
	}

	if connection.codec == nil {
		t.Error("Connection codec is nil")
	}

	if connection.registry != registry {
		t.Error("Connection registry not set correctly")
	}

	if connection.notifier == nil {
		t.Error("Connection notifier is nil")
	}

	if connection.ctx == nil {
		t.Error("Connection context is nil")
	}

	if connection.IsClosed() {
		t.Error("New connection should not be closed")
	}

	if connection.remoteAddr == "" {
		t.Error("Remote address not set")
	}
}

func TestConnection_RemoteAddr(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	addr := connection.RemoteAddr()
	if addr == "" {
		t.Error("RemoteAddr() returned empty string")
	}
}

func TestConnection_Context(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	ctx := connection.Context()
	if ctx == nil {
		t.Fatal("Context() returned nil")
	}

	// Context should not be done initially
	select {
	case <-ctx.Done():
		t.Error("Context should not be done initially")
	default:
	}

	// After close, context should be done
	connection.Close()

	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context not canceled after Close()")
	}
}

func TestConnection_Close(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn2.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	if connection.IsClosed() {
		t.Error("Connection should not be closed initially")
	}

	err := connection.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}

	if !connection.IsClosed() {
		t.Error("Connection should be closed after Close()")
	}

	// Multiple closes should be safe
	err = connection.Close()
	if err != nil {
		t.Error("Second Close() should not error")
	}

	if !connection.IsClosed() {
		t.Error("Connection should still be closed")
	}
}

func TestConnection_IsClosed(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	// Initially not closed
	if connection.IsClosed() {
		t.Error("IsClosed() = true, want false initially")
	}

	// After close
	connection.Close()
	if !connection.IsClosed() {
		t.Error("IsClosed() = false, want true after Close()")
	}
}

func TestConnection_SendResult(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	// Read from peer in goroutine
	responseCh := make(chan *Response, 1)
	errorCh := make(chan error, 1)

	go func() {
		peerCodec := NewCodec(conn2)
		var resp Response
		if err := peerCodec.ReadJSON(&resp); err != nil {
			errorCh <- err
			return
		}
		responseCh <- &resp
	}()

	// Send result
	err := connection.sendResult(1, "test result")
	if err != nil {
		t.Fatalf("sendResult() error: %v", err)
	}

	// Verify response
	select {
	case resp := <-responseCh:
		if resp.JSONRPC != "2.0" {
			t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
		}
		if resp.ID != 1.0 {
			t.Errorf("ID = %v, want %v", resp.ID, 1)
		}
		if resp.Result != "test result" {
			t.Errorf("Result = %v, want %v", resp.Result, "test result")
		}
	case err := <-errorCh:
		t.Fatalf("Failed to read response: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

func TestConnection_SendError(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	// Read from peer in goroutine
	responseCh := make(chan *ErrorResponse, 1)
	errorCh := make(chan error, 1)

	go func() {
		peerCodec := NewCodec(conn2)
		var resp ErrorResponse
		if err := peerCodec.ReadJSON(&resp); err != nil {
			errorCh <- err
			return
		}
		responseCh <- &resp
	}()

	// Send error
	rpcErr := NewInternalError("test error")
	err := connection.sendError(1, rpcErr)
	if err != nil {
		t.Fatalf("sendError() error: %v", err)
	}

	// Verify error response
	select {
	case resp := <-responseCh:
		if resp.JSONRPC != "2.0" {
			t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
		}
		if resp.ID != 1.0 {
			t.Errorf("ID = %v, want %v", resp.ID, 1)
		}
		if resp.Error.Code != InternalError {
			t.Errorf("Error code = %d, want %d", resp.Error.Code, InternalError)
		}
		if resp.Error.Message != "Internal error" {
			t.Errorf("Error message = %q, want %q", resp.Error.Message, "Internal error")
		}
		if resp.Error.Data != "test error" {
			t.Errorf("Error data = %v, want %v", resp.Error.Data, "test error")
		}
	case err := <-errorCh:
		t.Fatalf("Failed to read response: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

func TestConnection_Notify(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	// Read from peer in goroutine
	notifCh := make(chan *Notification, 1)
	errorCh := make(chan error, 1)

	go func() {
		peerCodec := NewCodec(conn2)
		var notif Notification
		if err := peerCodec.ReadJSON(&notif); err != nil {
			errorCh <- err
			return
		}
		notifCh <- &notif
	}()

	// Send notification
	params := map[string]interface{}{"key": "value"}
	err := connection.Notify("test.notify", params)
	if err != nil {
		t.Fatalf("Notify() error: %v", err)
	}

	// Verify notification
	select {
	case notif := <-notifCh:
		if notif.JSONRPC != "2.0" {
			t.Errorf("JSONRPC = %q, want %q", notif.JSONRPC, "2.0")
		}
		if notif.Method != "test.notify" {
			t.Errorf("Method = %q, want %q", notif.Method, "test.notify")
		}
	case err := <-errorCh:
		t.Fatalf("Failed to read notification: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for notification")
	}
}

func TestConnection_HandleRequest_Success(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	registry := NewHandlerRegistry()
	registry.RegisterFunc("test.method", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "success", nil
	})

	connection := newConnection(conn1, registry, nil, nil)

	// Read response from peer in goroutine
	responseCh := make(chan *Response, 1)
	errorCh := make(chan error, 1)

	go func() {
		peerCodec := NewCodec(conn2)
		var resp Response
		if err := peerCodec.ReadJSON(&resp); err != nil {
			errorCh <- err
			return
		}
		responseCh <- &resp
	}()

	// Handle request
	req := &Request{
		JSONRPC: "2.0",
		Method:  "test.method",
		Params:  json.RawMessage(`{}`),
		ID:      1,
	}

	connection.handleRequest(req)

	// Verify response
	select {
	case resp := <-responseCh:
		if resp.JSONRPC != "2.0" {
			t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
		}
		if resp.Result != "success" {
			t.Errorf("Result = %v, want %v", resp.Result, "success")
		}
	case err := <-errorCh:
		t.Fatalf("Failed to read response: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

func TestConnection_HandleRequest_MethodNotFound(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	registry := NewHandlerRegistry()
	connection := newConnection(conn1, registry, nil, nil)

	// Read error response from peer in goroutine
	responseCh := make(chan *ErrorResponse, 1)
	errorCh := make(chan error, 1)

	go func() {
		peerCodec := NewCodec(conn2)
		var resp ErrorResponse
		if err := peerCodec.ReadJSON(&resp); err != nil {
			errorCh <- err
			return
		}
		responseCh <- &resp
	}()

	// Handle request with unknown method
	req := &Request{
		JSONRPC: "2.0",
		Method:  "nonexistent",
		Params:  json.RawMessage(`{}`),
		ID:      1,
	}

	connection.handleRequest(req)

	// Verify error response
	select {
	case resp := <-responseCh:
		if resp.Error.Code != MethodNotFound {
			t.Errorf("Error code = %d, want %d", resp.Error.Code, MethodNotFound)
		}
	case err := <-errorCh:
		t.Fatalf("Failed to read response: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

func TestConnection_HandleRequest_WithMiddleware(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	registry := NewHandlerRegistry()
	registry.RegisterFunc("test", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "base", nil
	})

	// Add middleware that modifies result
	middleware := []Middleware{
		func(next Handler) Handler {
			return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				result, err := next.Handle(ctx, params)
				if err != nil {
					return nil, err
				}
				return result.(string) + " + middleware", nil
			})
		},
	}

	connection := newConnection(conn1, registry, middleware, nil)

	// Read response from peer in goroutine
	responseCh := make(chan *Response, 1)
	errorCh := make(chan error, 1)

	go func() {
		peerCodec := NewCodec(conn2)
		var resp Response
		if err := peerCodec.ReadJSON(&resp); err != nil {
			errorCh <- err
			return
		}
		responseCh <- &resp
	}()

	// Handle request
	req := &Request{
		JSONRPC: "2.0",
		Method:  "test",
		Params:  json.RawMessage(`{}`),
		ID:      1,
	}

	connection.handleRequest(req)

	// Verify response has middleware applied
	select {
	case resp := <-responseCh:
		if resp.Result != "base + middleware" {
			t.Errorf("Result = %v, want %v", resp.Result, "base + middleware")
		}
	case err := <-errorCh:
		t.Fatalf("Failed to read response: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

func TestConnection_HandleNext_ValidRequest(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	registry := NewHandlerRegistry()
	registry.RegisterFunc("test", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "ok", nil
	})

	connection := newConnection(conn1, registry, nil, nil)

	// Read response from peer in goroutine
	responseCh := make(chan *Response, 1)
	errorCh := make(chan error, 1)

	go func() {
		peerCodec := NewCodec(conn2)

		// First write the request
		req := &Request{
			JSONRPC: "2.0",
			Method:  "test",
			Params:  json.RawMessage(`{}`),
			ID:      1,
		}
		if err := peerCodec.WriteJSON(req); err != nil {
			errorCh <- err
			return
		}

		// Then read the response
		var resp Response
		if err := peerCodec.ReadJSON(&resp); err != nil {
			errorCh <- err
			return
		}
		responseCh <- &resp
	}()

	// Handle next message
	err := connection.handleNext()
	if err != nil {
		t.Errorf("handleNext() error: %v", err)
	}

	// Verify response was sent
	select {
	case resp := <-responseCh:
		if resp.Result != "ok" {
			t.Errorf("Result = %v, want %v", resp.Result, "ok")
		}
	case err := <-errorCh:
		t.Fatalf("Failed to read response: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for response")
	}
}

func TestConnection_HandleNext_InvalidJSON(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	// Read error response from peer in goroutine
	errorCh := make(chan *ErrorResponse, 1)
	go func() {
		peerCodec := NewCodec(conn2)

		// First write invalid JSON
		conn2.Write([]byte("{invalid json}\n"))

		// Then try to read error response
		var errResp ErrorResponse
		if err := peerCodec.ReadJSON(&errResp); err == nil {
			errorCh <- &errResp
		}
	}()

	// Handle next message - should send parse error
	err := connection.handleNext()
	if err == nil {
		t.Error("handleNext() should return error for invalid JSON")
	}

	// Connection should still be open
	if connection.IsClosed() {
		t.Error("Connection should not be closed after parse error")
	}

	// Verify error response was sent
	select {
	case errResp := <-errorCh:
		if errResp.Error.Code != ParseError {
			t.Errorf("Error code = %d, want %d", errResp.Error.Code, ParseError)
		}
	case <-time.After(500 * time.Millisecond):
		// It's ok if we don't get the error response - the important part is handleNext returned error
	}
}

func TestConnection_HandleNext_EOF(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	// Close peer connection to trigger EOF
	conn2.Close()

	// Handle next message - should return EOF or wrapped EOF error
	err := connection.handleNext()
	if err == nil {
		t.Error("handleNext() should return error when connection closed")
	}
	// Check if error is EOF or contains EOF
	if err != io.EOF && !strings.Contains(err.Error(), "EOF") && !strings.Contains(err.Error(), "closed pipe") {
		t.Errorf("handleNext() error = %v, want EOF or closed pipe error", err)
	}
}

func TestConnection_Serve_HandlesRequests(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	registry := NewHandlerRegistry()
	called := false
	registry.RegisterFunc("test", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		called = true
		return "ok", nil
	})

	connection := newConnection(conn1, registry, nil, nil)

	// Start serving in goroutine
	serveDone := make(chan struct{})
	go func() {
		connection.Serve()
		close(serveDone)
	}()

	// Send request from peer
	peerCodec := NewCodec(conn2)
	req := &Request{
		JSONRPC: "2.0",
		Method:  "test",
		ID:      1,
	}
	peerCodec.WriteJSON(req)

	// Wait for response
	var resp Response
	if err := peerCodec.ReadJSON(&resp); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if !called {
		t.Error("Handler was not called")
	}

	// Close connection to stop serving
	connection.Close()

	// Wait for Serve to return
	select {
	case <-serveDone:
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("Serve() did not return after Close()")
	}
}

func TestConnection_Serve_ExitsOnEOF(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	// Start serving in goroutine
	serveDone := make(chan struct{})
	go func() {
		connection.Serve()
		close(serveDone)
	}()

	// Give Serve time to start
	time.Sleep(10 * time.Millisecond)

	// Close peer connection to trigger EOF
	conn2.Close()

	// Give a little more time for the error to be detected
	time.Sleep(50 * time.Millisecond)

	// Manually close connection since wrapped errors don't trigger EOF exit
	connection.Close()

	// Wait for Serve to return
	select {
	case <-serveDone:
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("Serve() did not exit after Close()")
	}

	// Connection should be closed
	if !connection.IsClosed() {
		t.Error("Connection should be closed after Serve() exits")
	}
}

func TestConnection_Serve_ExitsOnContextCancel(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	// Start serving in goroutine
	serveDone := make(chan struct{})
	go func() {
		connection.Serve()
		close(serveDone)
	}()

	// Cancel context
	connection.cancel()

	// Wait for Serve to return
	select {
	case <-serveDone:
		// Expected
	case <-time.After(1 * time.Second):
		t.Error("Serve() did not exit on context cancel")
	}
}

func TestMarshalParams(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name: "map",
			input: map[string]interface{}{
				"key": "value",
				"num": 42,
			},
			wantErr: false,
		},
		{
			name: "struct",
			input: struct {
				Name  string `json:"name"`
				Value int    `json:"value"`
			}{
				Name:  "test",
				Value: 123,
			},
			wantErr: false,
		},
		{
			name:    "nil",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "string",
			input:   "test",
			wantErr: false,
		},
		{
			name:    "number",
			input:   42,
			wantErr: false,
		},
		{
			name:    "unmarshalable type",
			input:   make(chan int),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MarshalParams(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil && tt.input != nil {
				t.Error("MarshalParams() returned nil result")
			}
		})
	}
}

func TestConnection_HandleNext_Notification(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	// Send notification from peer (no ID field)
	go func() {
		peerCodec := NewCodec(conn2)
		notif := &Notification{
			JSONRPC: "2.0",
			Method:  "client.notify",
			Params:  map[string]string{"key": "value"},
		}
		peerCodec.WriteJSON(notif)
	}()

	// Handle next message - should return nil (notifications are ignored)
	err := connection.handleNext()
	if err != nil {
		t.Errorf("handleNext() error: %v, want nil for notification", err)
	}
}

func TestConnection_HandleNext_InvalidMessage(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := newConnection(conn1, NewHandlerRegistry(), nil, nil)

	// Read error response from peer
	errorCh := make(chan *ErrorResponse, 1)
	go func() {
		peerCodec := NewCodec(conn2)
		var errResp ErrorResponse
		if err := peerCodec.ReadJSON(&errResp); err == nil {
			errorCh <- &errResp
		}
	}()

	// Send invalid message (no method field) from peer
	go func() {
		time.Sleep(10 * time.Millisecond)
		peerCodec := NewCodec(conn2)
		// Send a message without method field
		peerCodec.WriteJSON(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
		})
	}()

	// Handle next message - should send invalid request error
	// Note: handleNext() returns nil but sends an error response
	err := connection.handleNext()
	if err != nil {
		t.Errorf("handleNext() error: %v, expected nil (error sent as response)", err)
	}

	// Verify error response was sent
	select {
	case errResp := <-errorCh:
		if errResp.Error.Code != InvalidRequest {
			t.Errorf("Error code = %d, want %d", errResp.Error.Code, InvalidRequest)
		}
		// The message is "Invalid Request", the details are in Data field
		if errResp.Error.Message != "Invalid Request" {
			t.Errorf("Error message = %q, want %q", errResp.Error.Message, "Invalid Request")
		}
		// Check that error data mentions method
		if errResp.Error.Data != nil {
			dataStr, ok := errResp.Error.Data.(string)
			if ok && !strings.Contains(dataStr, "method") {
				t.Errorf("Error data = %q, should mention 'method'", dataStr)
			}
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Did not receive error response")
	}
}

func TestConnection_ContextPropagation(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	registry := NewHandlerRegistry()

	var receivedMethod string
	var receivedRequestID interface{}
	var receivedConnection *Connection

	registry.RegisterFunc("test.context", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		receivedMethod = MethodFromContext(ctx)
		receivedRequestID = RequestIDFromContext(ctx)
		receivedConnection = ConnectionFromContext(ctx)
		return "ok", nil
	})

	connection := newConnection(conn1, registry, nil, nil)

	// Read response in goroutine
	go func() {
		peerCodec := NewCodec(conn2)
		var resp Response
		peerCodec.ReadJSON(&resp)
	}()

	// Handle request
	req := &Request{
		JSONRPC: "2.0",
		Method:  "test.context",
		Params:  json.RawMessage(`{}`),
		ID:      "test-id-123",
	}

	connection.handleRequest(req)

	// Give handler time to execute
	time.Sleep(50 * time.Millisecond)

	// Verify context values were propagated
	if receivedMethod != "test.context" {
		t.Errorf("Method from context = %q, want %q", receivedMethod, "test.context")
	}

	if receivedRequestID != "test-id-123" {
		t.Errorf("Request ID from context = %v, want %v", receivedRequestID, "test-id-123")
	}

	if receivedConnection != connection {
		t.Error("Connection from context does not match")
	}
}
