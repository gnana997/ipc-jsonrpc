package jsonrpcipc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Server is a JSON-RPC 2.0 server over IPC.
//
// The server listens on a Unix socket (Linux/Mac) or Named Pipe (Windows)
// and handles JSON-RPC requests from multiple clients concurrently.
type Server struct {
	config    ServerConfig
	listener  net.Listener
	registry  *HandlerRegistry
	broadcast *BroadcastManager

	middleware []Middleware

	// Connection management
	connections sync.Map // map[*Connection]bool
	wg          sync.WaitGroup

	// Lifecycle
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownCh chan struct{}
	startOnce  sync.Once
	stopOnce   sync.Once
}

// ServerConfig holds configuration options for the Server.
type ServerConfig struct {
	// SocketPath is the path to the Unix socket or Windows named pipe.
	//
	// Examples:
	//   - Unix/Linux/Mac: "/tmp/myapp.sock"
	//   - Windows: "myapp" (automatically converted to "\\.\pipe\myapp")
	SocketPath string

	// Logger is used for server logging.
	// If nil, a default logger is used.
	Logger Logger

	// OnConnect is called when a new client connects.
	// Optional.
	OnConnect func(*Connection)

	// OnDisconnect is called when a client disconnects.
	// Optional.
	OnDisconnect func(*Connection)

	// OnError is called when an error occurs that isn't tied to a specific request.
	// Optional.
	OnError func(error)
}

// NewServer creates a new JSON-RPC server.
//
// The server must be started by calling Start().
//
// Example:
//
//	server, err := NewServer(ServerConfig{
//	    SocketPath: "/tmp/myapp.sock",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewServer(config ServerConfig) (*Server, error) {
	if config.SocketPath == "" {
		return nil, fmt.Errorf("SocketPath is required")
	}

	// Set defaults
	if config.Logger == nil {
		config.Logger = func(method string, duration time.Duration, err error) {
			if err != nil {
				log.Printf("[JSON-RPC] method=%s duration=%v error=%v", method, duration, err)
			}
		}
	}
	if config.OnError == nil {
		config.OnError = func(err error) {
			log.Printf("[JSON-RPC] error: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		config:     config,
		registry:   NewHandlerRegistry(),
		broadcast:  NewBroadcastManager(),
		ctx:        ctx,
		cancel:     cancel,
		shutdownCh: make(chan struct{}),
	}, nil
}

// RegisterHandler registers a handler for the specified JSON-RPC method.
//
// If a handler is already registered for the method, it will be replaced.
//
// Example:
//
//	server.RegisterHandler("echo", HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
//	    return string(params), nil
//	}))
func (s *Server) RegisterHandler(method string, handler Handler) {
	s.registry.Register(method, handler)
}

// RegisterFunc is a convenience method to register a HandlerFunc.
//
// Example:
//
//	server.RegisterFunc("getData", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
//	    return map[string]string{"data": "value"}, nil
//	})
func (s *Server) RegisterFunc(method string, fn func(ctx context.Context, params json.RawMessage) (interface{}, error)) {
	s.RegisterHandler(method, HandlerFunc(fn))
}

// RegisterMiddleware adds middleware to the server.
//
// Middleware is applied to all handlers in the order they are registered.
//
// Example:
//
//	server.RegisterMiddleware(LoggingMiddleware(logger))
//	server.RegisterMiddleware(RecoveryMiddleware())
func (s *Server) RegisterMiddleware(mw Middleware) {
	s.middleware = append(s.middleware, mw)
}

// Start starts the server and begins accepting connections.
//
// This method blocks until the server is stopped or an error occurs.
//
// Example:
//
//	if err := server.Start(); err != nil {
//	    log.Fatal(err)
//	}
func (s *Server) Start() error {
	var err error

	s.startOnce.Do(func() {
		// Create listener
		s.listener, err = Listen(s.config.SocketPath)
		if err != nil {
			return
		}

		log.Printf("[JSON-RPC] Server listening on %s", GetSocketPath(s.config.SocketPath))

		// Accept connections
		err = s.acceptLoop()
	})

	return err
}

// acceptLoop accepts new connections in a loop.
func (s *Server) acceptLoop() error {
	for {
		select {
		case <-s.shutdownCh:
			return nil
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.shutdownCh:
				// Server is shutting down
				return nil
			default:
				// Log error but continue accepting
				s.config.OnError(fmt.Errorf("accept error: %w", err))
				continue
			}
		}

		// Handle connection in a goroutine
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single client connection.
func (s *Server) handleConnection(netConn net.Conn) {
	defer s.wg.Done()

	// Create connection
	conn := newConnection(netConn, s.registry, s.middleware, s)

	// Track connection
	s.connections.Store(conn, true)
	s.broadcast.Add(conn)

	// Call OnConnect hook
	if s.config.OnConnect != nil {
		s.config.OnConnect(conn)
	}

	// Serve requests (blocks until connection closes)
	conn.Serve()

	// Cleanup
	s.connections.Delete(conn)
	s.broadcast.Remove(conn)

	// Call OnDisconnect hook
	if s.config.OnDisconnect != nil {
		s.config.OnDisconnect(conn)
	}
}

// Stop gracefully stops the server.
//
// It closes the listener, waits for all active connections to complete,
// and cleans up resources.
//
// The context can be used to force shutdown after a timeout:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	server.Stop(ctx)
func (s *Server) Stop(ctx context.Context) error {
	var err error

	s.stopOnce.Do(func() {
		// Signal shutdown
		close(s.shutdownCh)

		// Stop accepting new connections
		if s.listener != nil {
			if e := s.listener.Close(); e != nil {
				err = fmt.Errorf("listener close error: %w", e)
			}
		}

		// Wait for connections to finish or timeout
		done := make(chan struct{})
		go func() {
			s.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// All connections finished gracefully
		case <-ctx.Done():
			// Timeout - force close all connections
			s.connections.Range(func(key, value interface{}) bool {
				conn := key.(*Connection)
				conn.Close()
				return true
			})

			// Wait for cleanup
			<-done
		}

		// Cancel server context
		s.cancel()

		// Clean up socket file (Unix only)
		if e := CleanupSocket(s.config.SocketPath); e != nil && err == nil {
			err = fmt.Errorf("socket cleanup error: %w", e)
		}

		log.Printf("[JSON-RPC] Server stopped")
	})

	return err
}

// Broadcast sends a notification to all connected clients.
//
// Returns the number of clients the notification was sent to.
//
// Example:
//
//	count := server.Broadcast("progress", map[string]interface{}{
//	    "percentage": 50,
//	    "message": "Processing...",
//	})
//	log.Printf("Sent notification to %d clients", count)
//
// Thread-safety: This method is safe to call concurrently.
func (s *Server) Broadcast(method string, params interface{}) int {
	return s.broadcast.Broadcast(method, params)
}

// ConnectionCount returns the number of active client connections.
func (s *Server) ConnectionCount() int {
	return s.broadcast.Count()
}

// Methods returns a list of all registered method names.
func (s *Server) Methods() []string {
	return s.registry.Methods()
}

// Context returns the server's context.
// The context is canceled when the server is stopped.
func (s *Server) Context() context.Context {
	return s.ctx
}
