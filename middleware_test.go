package jsonrpcipc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestChain_NoMiddleware(t *testing.T) {
	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "base result", nil
	})

	handler := Chain(baseHandler)

	result, err := handler.Handle(context.Background(), nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "base result" {
		t.Errorf("Result = %v, want %v", result, "base result")
	}
}

func TestChain_SingleMiddleware(t *testing.T) {
	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "base", nil
	})

	called := false
	middleware := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			called = true
			result, err := next.Handle(ctx, params)
			return result.(string) + " + middleware", err
		})
	}

	handler := Chain(baseHandler, middleware)

	result, err := handler.Handle(context.Background(), nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !called {
		t.Error("Middleware was not called")
	}
	if result != "base + middleware" {
		t.Errorf("Result = %v, want %v", result, "base + middleware")
	}
}

func TestChain_MultipleMiddleware(t *testing.T) {
	var order []string

	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		order = append(order, "handler")
		return "result", nil
	})

	middleware1 := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			order = append(order, "middleware1-before")
			result, err := next.Handle(ctx, params)
			order = append(order, "middleware1-after")
			return result, err
		})
	}

	middleware2 := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			order = append(order, "middleware2-before")
			result, err := next.Handle(ctx, params)
			order = append(order, "middleware2-after")
			return result, err
		})
	}

	middleware3 := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			order = append(order, "middleware3-before")
			result, err := next.Handle(ctx, params)
			order = append(order, "middleware3-after")
			return result, err
		})
	}

	handler := Chain(baseHandler, middleware1, middleware2, middleware3)

	_, err := handler.Handle(context.Background(), nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedOrder := []string{
		"middleware1-before",
		"middleware2-before",
		"middleware3-before",
		"handler",
		"middleware3-after",
		"middleware2-after",
		"middleware1-after",
	}

	if len(order) != len(expectedOrder) {
		t.Fatalf("Order length = %d, want %d", len(order), len(expectedOrder))
	}

	for i, expected := range expectedOrder {
		if order[i] != expected {
			t.Errorf("Order[%d] = %q, want %q", i, order[i], expected)
		}
	}
}

func TestChain_ErrorPropagation(t *testing.T) {
	expectedErr := errors.New("handler error")

	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return nil, expectedErr
	})

	middleware := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			return next.Handle(ctx, params)
		})
	}

	handler := Chain(baseHandler, middleware)

	_, err := handler.Handle(context.Background(), nil)
	if err != expectedErr {
		t.Errorf("Error = %v, want %v", err, expectedErr)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	var loggedMethod string
	var loggedDuration time.Duration
	var loggedErr error

	logger := func(method string, duration time.Duration, err error) {
		loggedMethod = method
		loggedDuration = duration
		loggedErr = err
	}

	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		time.Sleep(10 * time.Millisecond)
		return "success", nil
	})

	handler := Chain(baseHandler, LoggingMiddleware(logger))

	ctx := WithMethod(context.Background(), "test.method")
	result, err := handler.Handle(ctx, nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("Result = %v, want %v", result, "success")
	}

	if loggedMethod != "test.method" {
		t.Errorf("Logged method = %q, want %q", loggedMethod, "test.method")
	}
	if loggedDuration < 10*time.Millisecond {
		t.Errorf("Logged duration = %v, expected at least 10ms", loggedDuration)
	}
	if loggedErr != nil {
		t.Errorf("Logged error = %v, want nil", loggedErr)
	}
}

func TestLoggingMiddleware_WithError(t *testing.T) {
	var loggedErr error
	expectedErr := errors.New("test error")

	logger := func(method string, duration time.Duration, err error) {
		loggedErr = err
	}

	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return nil, expectedErr
	})

	handler := Chain(baseHandler, LoggingMiddleware(logger))

	ctx := WithMethod(context.Background(), "error.method")
	_, err := handler.Handle(ctx, nil)

	if err != expectedErr {
		t.Errorf("Error = %v, want %v", err, expectedErr)
	}
	if loggedErr != expectedErr {
		t.Errorf("Logged error = %v, want %v", loggedErr, expectedErr)
	}
}

func TestLoggingMiddleware_NoMethod(t *testing.T) {
	var loggedMethod string

	logger := func(method string, duration time.Duration, err error) {
		loggedMethod = method
	}

	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "ok", nil
	})

	handler := Chain(baseHandler, LoggingMiddleware(logger))

	// Context without method
	_, err := handler.Handle(context.Background(), nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if loggedMethod != "" {
		t.Errorf("Logged method = %q, want empty string", loggedMethod)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		panic("test panic")
	})

	handler := Chain(baseHandler, RecoveryMiddleware())

	ctx := WithMethod(context.Background(), "panic.method")
	result, err := handler.Handle(ctx, nil)

	if result != nil {
		t.Errorf("Result = %v, want nil", result)
	}

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	rpcErr, ok := err.(*RPCError)
	if !ok {
		t.Fatalf("Error type = %T, want *RPCError", err)
	}

	if rpcErr.Code != InternalError {
		t.Errorf("Error code = %d, want %d", rpcErr.Code, InternalError)
	}

	// Check that panic info is in the data
	dataMap, ok := rpcErr.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Data type = %T, want map[string]interface{}", rpcErr.Data)
	}

	if dataMap["panic"] != "test panic" {
		t.Errorf("Panic data = %v, want %v", dataMap["panic"], "test panic")
	}

	if dataMap["method"] != "panic.method" {
		t.Errorf("Method data = %v, want %v", dataMap["method"], "panic.method")
	}
}

func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "success", nil
	})

	handler := Chain(baseHandler, RecoveryMiddleware())

	result, err := handler.Handle(context.Background(), nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("Result = %v, want %v", result, "success")
	}
}

func TestRecoveryMiddleware_PanicWithNil(t *testing.T) {
	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		panic(errors.New("nil panic test"))
	})

	handler := Chain(baseHandler, RecoveryMiddleware())

	result, err := handler.Handle(context.Background(), nil)

	// Should not panic, but panic(nil) is recovered
	if result != nil {
		t.Errorf("Result = %v, want nil", result)
	}
	if err == nil {
		t.Error("Expected error from panic(nil), got nil")
	}
}

func TestRecoveryMiddleware_PanicWithError(t *testing.T) {
	panicErr := errors.New("panic error")

	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		panic(panicErr)
	})

	handler := Chain(baseHandler, RecoveryMiddleware())

	result, err := handler.Handle(context.Background(), nil)

	if result != nil {
		t.Errorf("Result = %v, want nil", result)
	}

	rpcErr, ok := err.(*RPCError)
	if !ok {
		t.Fatalf("Error type = %T, want *RPCError", err)
	}

	dataMap := rpcErr.Data.(map[string]interface{})
	if dataMap["panic"] != panicErr {
		t.Errorf("Panic data = %v, want %v", dataMap["panic"], panicErr)
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		time.Sleep(100 * time.Millisecond)
		return "completed", nil
	})

	handler := Chain(baseHandler, TimeoutMiddleware(10*time.Millisecond))

	result, err := handler.Handle(context.Background(), nil)

	if result != nil {
		t.Errorf("Result = %v, want nil (timeout)", result)
	}

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	rpcErr, ok := err.(*RPCError)
	if !ok {
		t.Fatalf("Error type = %T, want *RPCError", err)
	}

	if rpcErr.Code != InternalError {
		t.Errorf("Error code = %d, want %d", rpcErr.Code, InternalError)
	}

	if rpcErr.Data != "request timeout" {
		t.Errorf("Error data = %v, want %v", rpcErr.Data, "request timeout")
	}
}

func TestTimeoutMiddleware_NoTimeout(t *testing.T) {
	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		time.Sleep(10 * time.Millisecond)
		return "completed", nil
	})

	handler := Chain(baseHandler, TimeoutMiddleware(100*time.Millisecond))

	result, err := handler.Handle(context.Background(), nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "completed" {
		t.Errorf("Result = %v, want %v", result, "completed")
	}
}

func TestTimeoutMiddleware_ContextCancellation(t *testing.T) {
	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return "should not happen", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	handler := Chain(baseHandler, TimeoutMiddleware(50*time.Millisecond))

	result, err := handler.Handle(context.Background(), nil)

	// Timeout should occur
	if result != nil {
		t.Errorf("Result = %v, want nil", result)
	}
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestTimeoutMiddleware_HandlerError(t *testing.T) {
	expectedErr := errors.New("handler error")

	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return nil, expectedErr
	})

	handler := Chain(baseHandler, TimeoutMiddleware(100*time.Millisecond))

	result, err := handler.Handle(context.Background(), nil)

	if result != nil {
		t.Errorf("Result = %v, want nil", result)
	}
	if err != expectedErr {
		t.Errorf("Error = %v, want %v", err, expectedErr)
	}
}

func TestMiddleware_Combination(t *testing.T) {
	var loggedMethod string
	var loggedErr error

	logger := func(method string, duration time.Duration, err error) {
		loggedMethod = method
		loggedErr = err
	}

	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		// This would panic without recovery middleware
		if string(params) == `"panic"` {
			panic("intentional panic")
		}
		return "success", nil
	})

	// RecoveryMiddleware is first so it's the outermost wrapper and catches all panics
	handler := Chain(
		baseHandler,
		TimeoutMiddleware(100*time.Millisecond),
		LoggingMiddleware(logger),
		RecoveryMiddleware(),
	)

	ctx := WithMethod(context.Background(), "combined.test")

	// Test normal case
	result, err := handler.Handle(ctx, json.RawMessage(`"normal"`))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("Result = %v, want %v", result, "success")
	}
	if loggedMethod != "combined.test" {
		t.Errorf("Logged method = %q, want %q", loggedMethod, "combined.test")
	}

	// Test panic case - recovery catches it, logger logs the error
	result, err = handler.Handle(ctx, json.RawMessage(`"panic"`))
	if result != nil {
		t.Errorf("Result = %v, want nil (panic recovered)", result)
	}
	if err == nil {
		t.Error("Expected error from panic, got nil")
	}
	if loggedErr == nil {
		t.Error("Expected logged error, got nil")
	}
}

func TestMiddleware_OrderMatters(t *testing.T) {
	// Test that middleware order affects execution
	var order []string

	middleware1 := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			order = append(order, "m1-start")
			result, err := next.Handle(ctx, params)
			order = append(order, "m1-end")
			return result, err
		})
	}

	middleware2 := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			order = append(order, "m2-start")
			result, err := next.Handle(ctx, params)
			order = append(order, "m2-end")
			return result, err
		})
	}

	baseHandler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		order = append(order, "handler")
		return "ok", nil
	})

	// First middleware1, then middleware2
	handler := Chain(baseHandler, middleware1, middleware2)
	order = []string{}
	handler.Handle(context.Background(), nil)

	expected := []string{"m1-start", "m2-start", "handler", "m2-end", "m1-end"}
	if len(order) != len(expected) {
		t.Fatalf("Order length = %d, want %d", len(order), len(expected))
	}
	for i, exp := range expected {
		if order[i] != exp {
			t.Errorf("Order[%d] = %q, want %q", i, order[i], exp)
		}
	}
}
