package jsonrpcipc

import (
	"context"
	"encoding/json"
	"time"
)

// Middleware is a function that wraps a Handler to add pre/post processing.
//
// Middleware can be used for:
//   - Logging requests and responses
//   - Authentication and authorization
//   - Rate limiting
//   - Request/response transformation
//   - Error handling
//   - Metrics collection
//
// Example:
//
//	func LoggingMiddleware() Middleware {
//	    return func(next Handler) Handler {
//	        return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
//	            method := MethodFromContext(ctx)
//	            start := time.Now()
//
//	            log.Printf("[Request] method=%s", method)
//	            result, err := next.Handle(ctx, params)
//	            duration := time.Since(start)
//
//	            if err != nil {
//	                log.Printf("[Error] method=%s duration=%v error=%v", method, duration, err)
//	            } else {
//	                log.Printf("[Success] method=%s duration=%v", method, duration)
//	            }
//
//	            return result, err
//	        })
//	    }
//	}
type Middleware func(Handler) Handler

// Chain applies multiple middleware to a handler in order.
//
// Middleware are applied from first to last, meaning the first middleware
// in the chain is the outermost wrapper.
//
// Example:
//
//	handler := Chain(
//	    baseHandler,
//	    LoggingMiddleware(),
//	    AuthMiddleware(),
//	    RateLimitMiddleware(),
//	)
func Chain(handler Handler, middleware ...Middleware) Handler {
	// Apply middleware in reverse order so that the first middleware
	// is the outermost wrapper
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

// Logger is the logging function type used by LoggingMiddleware.
//
// It receives:
//   - method: The JSON-RPC method name
//   - duration: How long the request took
//   - err: Any error that occurred (nil if successful)
type Logger func(method string, duration time.Duration, err error)

// LoggingMiddleware creates middleware that logs each request using the provided logger.
func LoggingMiddleware(logger Logger) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			method := MethodFromContext(ctx)
			start := time.Now()

			result, err := next.Handle(ctx, params)
			duration := time.Since(start)

			logger(method, duration, err)

			return result, err
		})
	}
}

// RecoveryMiddleware creates a middleware that recovers from panics.
//
// If a handler panics, this middleware catches it and returns an InternalError.
func RecoveryMiddleware() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, params json.RawMessage) (result interface{}, err error) {
			defer func() {
				if r := recover(); r != nil {
					method := MethodFromContext(ctx)
					err = NewInternalError(map[string]interface{}{
						"panic":  r,
						"method": method,
					})
					result = nil
				}
			}()

			return next.Handle(ctx, params)
		})
	}
}

// TimeoutMiddleware creates a middleware that enforces a timeout on handlers.
//
// If a handler takes longer than the specified duration, it returns a timeout error.
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			type result struct {
				value interface{}
				err   error
			}

			resultChan := make(chan result, 1)

			go func() {
				value, err := next.Handle(ctx, params)
				resultChan <- result{value: value, err: err}
			}()

			select {
			case res := <-resultChan:
				return res.value, res.err
			case <-ctx.Done():
				return nil, NewInternalError("request timeout")
			}
		})
	}
}
