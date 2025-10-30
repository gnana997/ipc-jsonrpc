package jsonrpcipc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

// Connection represents a single client connection to the JSON-RPC server.
//
// Each connection has its own goroutine that reads requests from the client,
// dispatches them to handlers, and sends back responses.
type Connection struct {
	conn     net.Conn
	codec    *LineDelimitedCodec
	registry *HandlerRegistry
	notifier *NotificationManager

	ctx    context.Context
	cancel context.CancelFunc

	// Middleware chain
	middleware []Middleware

	// Connection metadata
	remoteAddr string

	// Lifecycle
	closeOnce sync.Once
	closed    chan struct{}

	// Server reference (for callbacks)
	server *Server
}

// newConnection creates a new connection.
// This is an internal function called by the Server.
func newConnection(conn net.Conn, registry *HandlerRegistry, middleware []Middleware, server *Server) *Connection {
	ctx, cancel := context.WithCancel(context.Background())

	codec := NewCodec(conn)

	return &Connection{
		conn:       conn,
		codec:      codec,
		registry:   registry,
		notifier:   NewNotificationManager(codec),
		ctx:        ctx,
		cancel:     cancel,
		middleware: middleware,
		remoteAddr: conn.RemoteAddr().String(),
		closed:     make(chan struct{}),
		server:     server,
	}
}

// Serve starts serving requests on this connection.
// This method blocks until the connection is closed.
func (c *Connection) Serve() {
	defer c.Close()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.closed:
			return
		default:
			if err := c.handleNext(); err != nil {
				if err == io.EOF {
					// Client disconnected gracefully
					return
				}
				// Log error but continue processing
				// (errors might be temporary or due to malformed messages)
				continue
			}
		}
	}
}

// handleNext reads and handles the next message from the client.
func (c *Connection) handleNext() error {
	// Read raw message
	var msg Message
	if err := c.codec.ReadJSON(&msg); err != nil {
		if err == io.EOF {
			return err
		}
		// Send parse error
		c.sendError(nil, NewParseError(err.Error()))
		return fmt.Errorf("failed to read message: %w", err)
	}

	// Handle based on message type
	if msg.IsRequest() {
		req, err := msg.ToRequest()
		if err != nil {
			c.sendError(msg.ID, NewInvalidRequestError(err.Error()))
			return err
		}
		c.handleRequest(req)
	} else if msg.IsNotification() {
		// Server can receive notifications from clients (though uncommon)
		// For now, we just ignore them
	} else {
		// Invalid message (not a request or notification)
		c.sendError(msg.ID, NewInvalidRequestError("message must have method field"))
	}

	return nil
}

// handleRequest processes a JSON-RPC request.
func (c *Connection) handleRequest(req *Request) {
	// Look up handler
	handler, ok := c.registry.Get(req.Method)
	if !ok {
		c.sendError(req.ID, NewMethodNotFoundError(req.Method))
		return
	}

	// Apply middleware
	for i := len(c.middleware) - 1; i >= 0; i-- {
		handler = c.middleware[i](handler)
	}

	// Create request context with metadata
	ctx := c.ctx
	ctx = WithMethod(ctx, req.Method)
	ctx = WithRequestID(ctx, req.ID)
	ctx = WithConnection(ctx, c)

	// Execute handler
	result, err := handler.Handle(ctx, req.Params)

	// Send response
	if err != nil {
		rpcErr := ToRPCError(err)
		c.sendError(req.ID, rpcErr)
	} else {
		c.sendResult(req.ID, result)
	}
}

// sendResult sends a success response to the client.
func (c *Connection) sendResult(id interface{}, result interface{}) error {
	response := &Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}

	return c.codec.WriteJSON(response)
}

// sendError sends an error response to the client.
func (c *Connection) sendError(id interface{}, rpcErr *RPCError) error {
	response := &ErrorResponse{
		JSONRPC: "2.0",
		Error:   rpcErr,
		ID:      id,
	}

	return c.codec.WriteJSON(response)
}

// Notify sends a notification to the client.
//
// Notifications are one-way messages from server to client that don't expect a response.
//
// Example:
//
//	conn.Notify("progress", map[string]interface{}{
//	    "percentage": 50,
//	    "message": "Processing...",
//	})
//
// Thread-safety: This method is safe to call concurrently.
func (c *Connection) Notify(method string, params interface{}) error {
	return c.notifier.Send(method, params)
}

// RemoteAddr returns the remote address of the client.
func (c *Connection) RemoteAddr() string {
	return c.remoteAddr
}

// Close closes the connection and cancels all pending requests.
//
// This method is safe to call multiple times.
func (c *Connection) Close() error {
	var err error

	c.closeOnce.Do(func() {
		// Signal closed
		close(c.closed)

		// Cancel context
		c.cancel()

		// Close notifier
		c.notifier.Close()

		// Close underlying connection
		if c.conn != nil {
			err = c.conn.Close()
		}
	})

	return err
}

// IsClosed returns true if the connection has been closed.
func (c *Connection) IsClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

// Context returns the connection's context.
// The context is canceled when the connection is closed.
func (c *Connection) Context() context.Context {
	return c.ctx
}

// MarshalParams is a helper to marshal parameters for use with Notify.
func MarshalParams(v interface{}) (interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}
