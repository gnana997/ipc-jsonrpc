package jsonrpcipc

import (
	"context"
	"encoding/json"
	"fmt"
)

// Handler processes a JSON-RPC request and returns a result or error.
//
// The handler receives:
//   - ctx: Request context (can be used for cancellation, timeouts, or passing values)
//   - params: Raw JSON parameters from the request (use json.Unmarshal to parse)
//
// The handler should return:
//   - result: The result to send back to the client (will be JSON-marshaled)
//   - error: An error if the request failed (automatically converted to JSON-RPC error)
//
// Example:
//
//	type MyHandler struct{}
//
//	func (h *MyHandler) Handle(ctx context.Context, params json.RawMessage) (interface{}, error) {
//	    var p MyParams
//	    if err := json.Unmarshal(params, &p); err != nil {
//	        return nil, NewInvalidParamsError(err.Error())
//	    }
//	    // Process request...
//	    return MyResult{Data: "value"}, nil
//	}
type Handler interface {
	Handle(ctx context.Context, params json.RawMessage) (interface{}, error)
}

// HandlerFunc is an adapter to allow ordinary functions to be used as Handlers.
//
// Example:
//
//	server.RegisterHandler("echo", HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
//	    return string(params), nil
//	}))
type HandlerFunc func(ctx context.Context, params json.RawMessage) (interface{}, error)

// Handle calls the function itself.
func (f HandlerFunc) Handle(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return f(ctx, params)
}

// TypedHandler creates a Handler from a function with typed parameters and result.
//
// This is a convenience function that automatically handles JSON marshaling/unmarshaling
// of parameters and results. It provides type safety and reduces boilerplate.
//
// Type parameters:
//   - P: Parameter type (must be JSON-unmarshalable)
//   - R: Result type (must be JSON-marshalable)
//
// Example:
//
//	type SearchParams struct {
//	    Query string `json:"query"`
//	    Limit int    `json:"limit"`
//	}
//
//	type SearchResult struct {
//	    Items []string `json:"items"`
//	    Total int      `json:"total"`
//	}
//
//	func handleSearch(ctx context.Context, params SearchParams) (SearchResult, error) {
//	    // Type-safe parameter access
//	    items := search(params.Query, params.Limit)
//	    return SearchResult{Items: items, Total: len(items)}, nil
//	}
//
//	server.RegisterHandler("search", TypedHandler(handleSearch))
func TypedHandler[P any, R any](fn func(ctx context.Context, params P) (R, error)) Handler {
	return HandlerFunc(func(ctx context.Context, raw json.RawMessage) (interface{}, error) {
		// Parse parameters
		var params P
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &params); err != nil {
				return nil, NewInvalidParamsError(fmt.Sprintf("failed to parse parameters: %v", err))
			}
		}

		// Call the typed handler
		result, err := fn(ctx, params)
		if err != nil {
			return nil, err
		}

		return result, nil
	})
}

// HandlerRegistry manages registered handlers for JSON-RPC methods.
type HandlerRegistry struct {
	handlers map[string]Handler
}

// NewHandlerRegistry creates a new handler registry.
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]Handler),
	}
}

// Register adds a handler for the specified method.
//
// If a handler is already registered for the method, it will be replaced.
//
// Parameters:
//   - method: The JSON-RPC method name
//   - handler: The handler to invoke for this method
func (r *HandlerRegistry) Register(method string, handler Handler) {
	r.handlers[method] = handler
}

// RegisterFunc is a convenience method to register a HandlerFunc.
//
// Example:
//
//	registry.RegisterFunc("echo", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
//	    return string(params), nil
//	})
func (r *HandlerRegistry) RegisterFunc(method string, fn func(ctx context.Context, params json.RawMessage) (interface{}, error)) {
	r.Register(method, HandlerFunc(fn))
}

// Get retrieves the handler for the specified method.
//
// Returns:
//   - handler: The registered handler
//   - ok: true if a handler was found, false otherwise
func (r *HandlerRegistry) Get(method string) (Handler, bool) {
	handler, ok := r.handlers[method]
	return handler, ok
}

// Has checks if a handler is registered for the specified method.
func (r *HandlerRegistry) Has(method string) bool {
	_, ok := r.handlers[method]
	return ok
}

// Unregister removes the handler for the specified method.
func (r *HandlerRegistry) Unregister(method string) {
	delete(r.handlers, method)
}

// Methods returns a list of all registered method names.
func (r *HandlerRegistry) Methods() []string {
	methods := make([]string, 0, len(r.handlers))
	for method := range r.handlers {
		methods = append(methods, method)
	}
	return methods
}

// Clear removes all registered handlers.
func (r *HandlerRegistry) Clear() {
	r.handlers = make(map[string]Handler)
}

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// contextKeyMethod stores the current JSON-RPC method name in the context.
	contextKeyMethod contextKey = "jsonrpc.method"

	// contextKeyRequestID stores the current JSON-RPC request ID in the context.
	contextKeyRequestID contextKey = "jsonrpc.request_id"

	// contextKeyConnection stores the current connection in the context.
	contextKeyConnection contextKey = "jsonrpc.connection"
)

// MethodFromContext retrieves the JSON-RPC method name from the context.
// Returns an empty string if not found.
func MethodFromContext(ctx context.Context) string {
	if method, ok := ctx.Value(contextKeyMethod).(string); ok {
		return method
	}
	return ""
}

// WithMethod adds the JSON-RPC method name to the context.
func WithMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, contextKeyMethod, method)
}

// RequestIDFromContext retrieves the JSON-RPC request ID from the context.
// Returns nil if not found.
func RequestIDFromContext(ctx context.Context) interface{} {
	return ctx.Value(contextKeyRequestID)
}

// WithRequestID adds the JSON-RPC request ID to the context.
func WithRequestID(ctx context.Context, id interface{}) context.Context {
	return context.WithValue(ctx, contextKeyRequestID, id)
}

// ConnectionFromContext retrieves the Connection from the context.
// Returns nil if not found.
func ConnectionFromContext(ctx context.Context) *Connection {
	if conn, ok := ctx.Value(contextKeyConnection).(*Connection); ok {
		return conn
	}
	return nil
}

// WithConnection adds the Connection to the context.
func WithConnection(ctx context.Context, conn *Connection) context.Context {
	return context.WithValue(ctx, contextKeyConnection, conn)
}
