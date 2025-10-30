package jsonrpcipc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// Test Handler interface implementation
type testHandler struct {
	result interface{}
	err    error
}

func (h *testHandler) Handle(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return h.result, h.err
}

func TestHandler_Interface(t *testing.T) {
	handler := &testHandler{
		result: "test result",
		err:    nil,
	}

	result, err := handler.Handle(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "test result" {
		t.Errorf("Result = %v, want %v", result, "test result")
	}
}

func TestHandlerFunc(t *testing.T) {
	called := false
	handler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		called = true
		return "func result", nil
	})

	result, err := handler.Handle(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "func result" {
		t.Errorf("Result = %v, want %v", result, "func result")
	}
	if !called {
		t.Error("Handler function was not called")
	}
}

func TestHandlerFunc_WithError(t *testing.T) {
	expectedErr := errors.New("handler error")
	handler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return nil, expectedErr
	})

	result, err := handler.Handle(context.Background(), json.RawMessage(`{}`))
	if err != expectedErr {
		t.Errorf("Error = %v, want %v", err, expectedErr)
	}
	if result != nil {
		t.Errorf("Result = %v, want nil", result)
	}
}

func TestTypedHandler(t *testing.T) {
	type Params struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	type Result struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}

	handler := TypedHandler(func(ctx context.Context, params Params) (Result, error) {
		return Result{
			Message: "Hello " + params.Name,
			Count:   params.Value * 2,
		}, nil
	})

	params := Params{Name: "World", Value: 21}
	paramsJSON, _ := json.Marshal(params)

	result, err := handler.Handle(context.Background(), paramsJSON)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultTyped, ok := result.(Result)
	if !ok {
		t.Fatalf("Result type = %T, want Result", result)
	}

	if resultTyped.Message != "Hello World" {
		t.Errorf("Message = %q, want %q", resultTyped.Message, "Hello World")
	}
	if resultTyped.Count != 42 {
		t.Errorf("Count = %d, want %d", resultTyped.Count, 42)
	}
}

func TestTypedHandler_EmptyParams(t *testing.T) {
	type Result struct {
		Success bool `json:"success"`
	}

	handler := TypedHandler(func(ctx context.Context, params struct{}) (Result, error) {
		return Result{Success: true}, nil
	})

	// Empty params
	result, err := handler.Handle(context.Background(), json.RawMessage(``))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultTyped := result.(Result)
	if !resultTyped.Success {
		t.Error("Expected Success to be true")
	}
}

func TestTypedHandler_InvalidParams(t *testing.T) {
	type Params struct {
		Count int `json:"count"`
	}

	handler := TypedHandler(func(ctx context.Context, params Params) (string, error) {
		return "ok", nil
	})

	// Invalid JSON
	_, err := handler.Handle(context.Background(), json.RawMessage(`{invalid json}`))
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	// Check that it's an InvalidParams error
	if !IsRPCError(err) {
		t.Error("Expected RPCError")
	}

	rpcErr := err.(*RPCError)
	if rpcErr.Code != InvalidParams {
		t.Errorf("Error code = %d, want %d", rpcErr.Code, InvalidParams)
	}
}

func TestTypedHandler_HandlerError(t *testing.T) {
	expectedErr := errors.New("handler failed")

	handler := TypedHandler(func(ctx context.Context, params struct{}) (string, error) {
		return "", expectedErr
	})

	_, err := handler.Handle(context.Background(), json.RawMessage(``))
	if err != expectedErr {
		t.Errorf("Error = %v, want %v", err, expectedErr)
	}
}

func TestNewHandlerRegistry(t *testing.T) {
	registry := NewHandlerRegistry()

	if registry == nil {
		t.Fatal("NewHandlerRegistry returned nil")
	}

	if registry.handlers == nil {
		t.Error("Registry handlers map is nil")
	}

	if len(registry.handlers) != 0 {
		t.Errorf("New registry should be empty, got %d handlers", len(registry.handlers))
	}
}

func TestHandlerRegistry_Register(t *testing.T) {
	registry := NewHandlerRegistry()
	handler := &testHandler{result: "test", err: nil}

	registry.Register("test.method", handler)

	retrieved, ok := registry.Get("test.method")
	if !ok {
		t.Error("Handler not found after registration")
	}
	if retrieved != handler {
		t.Error("Retrieved handler does not match registered handler")
	}
}

func TestHandlerRegistry_Register_Replace(t *testing.T) {
	registry := NewHandlerRegistry()
	handler1 := &testHandler{result: "first", err: nil}
	handler2 := &testHandler{result: "second", err: nil}

	registry.Register("method", handler1)
	registry.Register("method", handler2)

	retrieved, _ := registry.Get("method")
	if retrieved != handler2 {
		t.Error("Handler was not replaced")
	}
}

func TestHandlerRegistry_RegisterFunc(t *testing.T) {
	registry := NewHandlerRegistry()

	called := false
	registry.RegisterFunc("func.method", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		called = true
		return "result", nil
	})

	handler, ok := registry.Get("func.method")
	if !ok {
		t.Fatal("Handler not found")
	}

	result, err := handler.Handle(context.Background(), nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "result" {
		t.Errorf("Result = %v, want %v", result, "result")
	}
	if !called {
		t.Error("Handler function was not called")
	}
}

func TestHandlerRegistry_Get_NotFound(t *testing.T) {
	registry := NewHandlerRegistry()

	handler, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Expected ok=false for nonexistent method")
	}
	if handler != nil {
		t.Error("Expected nil handler for nonexistent method")
	}
}

func TestHandlerRegistry_Has(t *testing.T) {
	registry := NewHandlerRegistry()
	handler := &testHandler{}

	if registry.Has("test") {
		t.Error("Expected Has to return false for unregistered method")
	}

	registry.Register("test", handler)

	if !registry.Has("test") {
		t.Error("Expected Has to return true for registered method")
	}
}

func TestHandlerRegistry_Unregister(t *testing.T) {
	registry := NewHandlerRegistry()
	handler := &testHandler{}

	registry.Register("test", handler)

	if !registry.Has("test") {
		t.Fatal("Handler not registered")
	}

	registry.Unregister("test")

	if registry.Has("test") {
		t.Error("Handler still exists after Unregister")
	}
}

func TestHandlerRegistry_Unregister_NonExistent(t *testing.T) {
	registry := NewHandlerRegistry()

	// Should not panic
	registry.Unregister("nonexistent")
}

func TestHandlerRegistry_Methods(t *testing.T) {
	registry := NewHandlerRegistry()

	// Empty registry
	methods := registry.Methods()
	if len(methods) != 0 {
		t.Errorf("Expected 0 methods, got %d", len(methods))
	}

	// Register some methods
	registry.Register("method1", &testHandler{})
	registry.Register("method2", &testHandler{})
	registry.Register("method3", &testHandler{})

	methods = registry.Methods()
	if len(methods) != 3 {
		t.Errorf("Expected 3 methods, got %d", len(methods))
	}

	// Verify all methods are present
	methodMap := make(map[string]bool)
	for _, m := range methods {
		methodMap[m] = true
	}

	for _, expected := range []string{"method1", "method2", "method3"} {
		if !methodMap[expected] {
			t.Errorf("Expected method %q not found in Methods()", expected)
		}
	}
}

func TestHandlerRegistry_Clear(t *testing.T) {
	registry := NewHandlerRegistry()

	// Register some handlers
	registry.Register("method1", &testHandler{})
	registry.Register("method2", &testHandler{})
	registry.Register("method3", &testHandler{})

	if len(registry.Methods()) != 3 {
		t.Fatal("Failed to register handlers")
	}

	registry.Clear()

	if len(registry.Methods()) != 0 {
		t.Errorf("Expected 0 methods after Clear, got %d", len(registry.Methods()))
	}

	if registry.Has("method1") {
		t.Error("method1 still exists after Clear")
	}
}

func TestMethodFromContext(t *testing.T) {
	ctx := context.Background()

	// No method in context
	method := MethodFromContext(ctx)
	if method != "" {
		t.Errorf("Expected empty string, got %q", method)
	}

	// With method in context
	ctx = WithMethod(ctx, "test.method")
	method = MethodFromContext(ctx)
	if method != "test.method" {
		t.Errorf("Method = %q, want %q", method, "test.method")
	}
}

func TestWithMethod(t *testing.T) {
	ctx := context.Background()
	ctx = WithMethod(ctx, "my.method")

	value := ctx.Value(contextKeyMethod)
	if value == nil {
		t.Fatal("Method not stored in context")
	}

	method, ok := value.(string)
	if !ok {
		t.Fatalf("Value type = %T, want string", value)
	}

	if method != "my.method" {
		t.Errorf("Method = %q, want %q", method, "my.method")
	}
}

func TestRequestIDFromContext(t *testing.T) {
	ctx := context.Background()

	// No request ID in context
	id := RequestIDFromContext(ctx)
	if id != nil {
		t.Errorf("Expected nil, got %v", id)
	}

	// With request ID in context
	ctx = WithRequestID(ctx, 42)
	id = RequestIDFromContext(ctx)
	if id != 42 {
		t.Errorf("Request ID = %v, want %v", id, 42)
	}
}

func TestRequestIDFromContext_DifferentTypes(t *testing.T) {
	tests := []struct {
		name string
		id   interface{}
	}{
		{"integer", 123},
		{"string", "req-456"},
		{"float", 78.9},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithRequestID(context.Background(), tt.id)
			retrieved := RequestIDFromContext(ctx)
			if retrieved != tt.id {
				t.Errorf("Retrieved ID = %v, want %v", retrieved, tt.id)
			}
		})
	}
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "test-id-789")

	value := ctx.Value(contextKeyRequestID)
	if value == nil {
		t.Fatal("Request ID not stored in context")
	}

	id, ok := value.(string)
	if !ok {
		t.Fatalf("Value type = %T, want string", value)
	}

	if id != "test-id-789" {
		t.Errorf("Request ID = %q, want %q", id, "test-id-789")
	}
}

func TestConnectionFromContext(t *testing.T) {
	ctx := context.Background()

	// No connection in context
	conn := ConnectionFromContext(ctx)
	if conn != nil {
		t.Errorf("Expected nil, got %v", conn)
	}

	// With connection in context
	mockConn1, mockConn2 := newMockConnPair()
	defer mockConn1.Close()
	defer mockConn2.Close()

	testConn := &Connection{conn: mockConn1}
	ctx = WithConnection(ctx, testConn)

	conn = ConnectionFromContext(ctx)
	if conn != testConn {
		t.Error("Retrieved connection does not match stored connection")
	}
}

func TestWithConnection(t *testing.T) {
	ctx := context.Background()

	mockConn1, mockConn2 := newMockConnPair()
	defer mockConn1.Close()
	defer mockConn2.Close()

	testConn := &Connection{conn: mockConn1}
	ctx = WithConnection(ctx, testConn)

	value := ctx.Value(contextKeyConnection)
	if value == nil {
		t.Fatal("Connection not stored in context")
	}

	conn, ok := value.(*Connection)
	if !ok {
		t.Fatalf("Value type = %T, want *Connection", value)
	}

	if conn != testConn {
		t.Error("Stored connection does not match")
	}
}

func TestContext_MultipleValues(t *testing.T) {
	ctx := context.Background()

	// Add all context values
	ctx = WithMethod(ctx, "test.method")
	ctx = WithRequestID(ctx, 123)

	mockConn1, mockConn2 := newMockConnPair()
	defer mockConn1.Close()
	defer mockConn2.Close()
	testConn := &Connection{conn: mockConn1}
	ctx = WithConnection(ctx, testConn)

	// Verify all values are present
	if MethodFromContext(ctx) != "test.method" {
		t.Error("Method not preserved")
	}
	if RequestIDFromContext(ctx) != 123 {
		t.Error("Request ID not preserved")
	}
	if ConnectionFromContext(ctx) != testConn {
		t.Error("Connection not preserved")
	}
}

func TestTypedHandler_ContextPropagation(t *testing.T) {
	type Params struct {
		Value string `json:"value"`
	}

	receivedMethod := ""

	handler := TypedHandler(func(ctx context.Context, params Params) (string, error) {
		receivedMethod = MethodFromContext(ctx)
		return "ok", nil
	})

	ctx := WithMethod(context.Background(), "my.method")
	paramsJSON, _ := json.Marshal(Params{Value: "test"})

	_, err := handler.Handle(ctx, paramsJSON)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if receivedMethod != "my.method" {
		t.Errorf("Method in handler = %q, want %q", receivedMethod, "my.method")
	}
}
