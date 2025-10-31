package jsonrpcipc

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	var socketPath string
	if runtime.GOOS == "windows" {
		socketPath = "test-server-" + time.Now().Format("20060102150405")
	} else {
		tmpDir := t.TempDir()
		socketPath = filepath.Join(tmpDir, "test.sock")
	}

	server, err := NewServer(ServerConfig{
		SocketPath: socketPath,
	})

	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	if server.registry == nil {
		t.Error("Server registry is nil")
	}

	if server.broadcast == nil {
		t.Error("Server broadcast manager is nil")
	}

	if server.ctx == nil {
		t.Error("Server context is nil")
	}
}

func TestNewServer_EmptySocketPath(t *testing.T) {
	server, err := NewServer(ServerConfig{
		SocketPath: "",
	})

	if err == nil {
		t.Error("NewServer() with empty SocketPath should return error")
	}

	if server != nil {
		t.Error("NewServer() with empty SocketPath should return nil server")
	}
}

func TestNewServer_DefaultLogger(t *testing.T) {
	var socketPath string
	if runtime.GOOS == "windows" {
		socketPath = "test-default-logger"
	} else {
		tmpDir := t.TempDir()
		socketPath = filepath.Join(tmpDir, "logger.sock")
	}

	server, err := NewServer(ServerConfig{
		SocketPath: socketPath,
		Logger:     nil, // Should use default
	})

	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	if server.config.Logger == nil {
		t.Error("Default logger not set")
	}
}

func TestServer_RegisterHandler(t *testing.T) {
	server, err := NewServer(ServerConfig{
		SocketPath: "test-handlers",
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	handler := HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "test", nil
	})

	server.RegisterHandler("test.method", handler)

	if !server.registry.Has("test.method") {
		t.Error("Handler not registered")
	}
}

func TestServer_RegisterFunc(t *testing.T) {
	server, err := NewServer(ServerConfig{
		SocketPath: "test-func",
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	server.RegisterFunc("test", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "ok", nil
	})

	if !server.registry.Has("test") {
		t.Error("Function handler not registered")
	}
}

func TestServer_RegisterMiddleware(t *testing.T) {
	server, err := NewServer(ServerConfig{
		SocketPath: "test-middleware",
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	middleware := func(next Handler) Handler {
		return next
	}

	server.RegisterMiddleware(middleware)

	if len(server.middleware) != 1 {
		t.Errorf("Middleware count = %d, want 1", len(server.middleware))
	}
}

func TestServer_Methods(t *testing.T) {
	server, err := NewServer(ServerConfig{
		SocketPath: "test-methods",
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	// Initially empty
	methods := server.Methods()
	if len(methods) != 0 {
		t.Errorf("Initial methods count = %d, want 0", len(methods))
	}

	// Register some methods
	server.RegisterFunc("method1", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return nil, nil
	})
	server.RegisterFunc("method2", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return nil, nil
	})

	methods = server.Methods()
	if len(methods) != 2 {
		t.Errorf("Methods count = %d, want 2", len(methods))
	}
}

func TestServer_Context(t *testing.T) {
	server, err := NewServer(ServerConfig{
		SocketPath: "test-context",
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	ctx := server.Context()
	if ctx == nil {
		t.Fatal("Context() returned nil")
	}

	// Context should not be done initially
	select {
	case <-ctx.Done():
		t.Error("Context should not be done initially")
	default:
	}

	// After stop, context should be done
	server.Stop(context.Background())

	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Context not canceled after Stop()")
	}
}

func TestServer_ConnectionCount(t *testing.T) {
	server, err := NewServer(ServerConfig{
		SocketPath: "test-count",
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	// Initially zero
	count := server.ConnectionCount()
	if count != 0 {
		t.Errorf("Initial connection count = %d, want 0", count)
	}
}

func TestServer_Broadcast_NoConnections(t *testing.T) {
	server, err := NewServer(ServerConfig{
		SocketPath: "test-broadcast",
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	// Broadcast with no connections
	count := server.Broadcast("test", nil)
	if count != 0 {
		t.Errorf("Broadcast() count = %d, want 0", count)
	}
}

func TestServer_StartStop(t *testing.T) {
	var socketPath string
	if runtime.GOOS == "windows" {
		socketPath = "test-start-stop-" + time.Now().Format("20060102150405")
	} else {
		tmpDir := t.TempDir()
		socketPath = filepath.Join(tmpDir, "start-stop.sock")
	}

	server, err := NewServer(ServerConfig{
		SocketPath: socketPath,
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	server.RegisterFunc("echo", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return string(params), nil
	})

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error: %v", err)
	}

	// Wait for Start() to return
	select {
	case err := <-serverErr:
		if err != nil {
			t.Errorf("Start() error after Stop(): %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Start() did not return after Stop()")
	}
}

func TestServer_MultipleStop(t *testing.T) {
	var socketPath string
	if runtime.GOOS == "windows" {
		socketPath = "test-multi-stop"
	} else {
		tmpDir := t.TempDir()
		socketPath = filepath.Join(tmpDir, "multi-stop.sock")
	}

	server, err := NewServer(ServerConfig{
		SocketPath: socketPath,
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	// Start server
	go server.Start()
	time.Sleep(50 * time.Millisecond)

	// Stop multiple times (should be safe)
	ctx := context.Background()
	err1 := server.Stop(ctx)
	err2 := server.Stop(ctx)
	err3 := server.Stop(ctx)

	if err1 != nil {
		t.Errorf("First Stop() error: %v", err1)
	}
	if err2 != nil {
		t.Error("Second Stop() should not error")
	}
	if err3 != nil {
		t.Error("Third Stop() should not error")
	}
}

func TestServer_IntegrationWithClient(t *testing.T) {
	var socketPath string
	if runtime.GOOS == "windows" {
		socketPath = "test-integration-" + time.Now().Format("20060102150405")
	} else {
		tmpDir := t.TempDir()
		socketPath = filepath.Join(tmpDir, "integration.sock")
		defer os.Remove(socketPath)
	}

	// Create and start server
	server, err := NewServer(ServerConfig{
		SocketPath: socketPath,
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	// Register a test method
	server.RegisterFunc("add", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var nums struct {
			A int `json:"a"`
			B int `json:"b"`
		}
		if err := json.Unmarshal(params, &nums); err != nil {
			return nil, err
		}
		return nums.A + nums.B, nil
	})

	// Start server in goroutine
	go server.Start()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Connect client
	clientConn, err := Dial(socketPath)
	if err != nil {
		t.Fatalf("Dial() error: %v", err)
	}
	defer clientConn.Close()

	// Send request
	codec := NewCodec(clientConn)
	req := &Request{
		JSONRPC: "2.0",
		Method:  "add",
		Params:  json.RawMessage(`{"a": 5, "b": 3}`),
		ID:      1,
	}

	if err := codec.WriteJSON(req); err != nil {
		t.Fatalf("WriteJSON() error: %v", err)
	}

	// Read response
	var resp Response
	if err := codec.ReadJSON(&resp); err != nil {
		t.Fatalf("ReadJSON() error: %v", err)
	}

	// Verify response
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}

	result, ok := resp.Result.(float64)
	if !ok {
		t.Fatalf("Result type = %T, want float64", resp.Result)
	}

	if result != 8 {
		t.Errorf("Result = %v, want 8", result)
	}

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Stop(ctx)
}

func TestServer_OnConnectCallback(t *testing.T) {
	var socketPath string
	if runtime.GOOS == "windows" {
		socketPath = "test-onconnect-" + time.Now().Format("20060102150405")
	} else {
		tmpDir := t.TempDir()
		socketPath = filepath.Join(tmpDir, "onconnect.sock")
	}

	connectCalled := false
	server, err := NewServer(ServerConfig{
		SocketPath: socketPath,
		OnConnect: func(conn *Connection) {
			connectCalled = true
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	server.RegisterFunc("test", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "ok", nil
	})

	// Start server
	go server.Start()
	time.Sleep(50 * time.Millisecond)

	// Connect client
	clientConn, err := Dial(socketPath)
	if err != nil {
		t.Fatalf("Dial() error: %v", err)
	}

	// Give time for callback
	time.Sleep(50 * time.Millisecond)

	if !connectCalled {
		t.Error("OnConnect callback was not called")
	}

	clientConn.Close()

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Stop(ctx)
}

func TestServer_OnDisconnectCallback(t *testing.T) {
	// This test verifies OnDisconnect is called when server stops
	var socketPath string
	if runtime.GOOS == "windows" {
		socketPath = "test-ondisconnect-" + time.Now().Format("20060102150405")
	} else {
		tmpDir := t.TempDir()
		socketPath = filepath.Join(tmpDir, "ondisconnect.sock")
	}

	disconnectCalled := false
	server, err := NewServer(ServerConfig{
		SocketPath: socketPath,
		OnDisconnect: func(conn *Connection) {
			disconnectCalled = true
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	server.RegisterFunc("test", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		return "ok", nil
	})

	// Start server
	go server.Start()
	time.Sleep(100 * time.Millisecond)

	// Connect client
	clientConn, err := Dial(socketPath)
	if err != nil {
		t.Fatalf("Dial() error: %v", err)
	}

	// Give time for connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Stop server (this will close all connections and trigger OnDisconnect)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Stop(ctx)

	// OnDisconnect should have been called when server stopped
	if !disconnectCalled {
		t.Error("OnDisconnect callback was not called when server stopped")
	}

	clientConn.Close()
}

func TestServer_Broadcast_WithConnections(t *testing.T) {
	var socketPath string
	if runtime.GOOS == "windows" {
		socketPath = "test-broadcast-conn-" + time.Now().Format("20060102150405")
	} else {
		tmpDir := t.TempDir()
		socketPath = filepath.Join(tmpDir, "broadcast-conn.sock")
	}

	server, err := NewServer(ServerConfig{
		SocketPath: socketPath,
	})
	if err != nil {
		t.Fatalf("NewServer() error: %v", err)
	}

	// Start server
	go server.Start()
	time.Sleep(50 * time.Millisecond)

	// Connect 2 clients
	client1, err := Dial(socketPath)
	if err != nil {
		t.Fatalf("Dial() client1 error: %v", err)
	}
	defer client1.Close()

	client2, err := Dial(socketPath)
	if err != nil {
		t.Fatalf("Dial() client2 error: %v", err)
	}
	defer client2.Close()

	// Give time for connections to be established
	time.Sleep(100 * time.Millisecond)

	// Check connection count
	if server.ConnectionCount() != 2 {
		t.Errorf("ConnectionCount() = %d, want 2", server.ConnectionCount())
	}

	// Read notifications from clients in goroutines
	notifCh1 := make(chan *Notification, 1)
	notifCh2 := make(chan *Notification, 1)

	go func() {
		codec := NewCodec(client1)
		var notif Notification
		if err := codec.ReadJSON(&notif); err == nil {
			notifCh1 <- &notif
		}
	}()

	go func() {
		codec := NewCodec(client2)
		var notif Notification
		if err := codec.ReadJSON(&notif); err == nil {
			notifCh2 <- &notif
		}
	}()

	// Broadcast
	count := server.Broadcast("test.broadcast", map[string]string{"msg": "hello"})
	if count != 2 {
		t.Errorf("Broadcast() count = %d, want 2", count)
	}

	// Verify both clients received notification
	timeout := time.After(1 * time.Second)

	select {
	case notif := <-notifCh1:
		if notif.Method != "test.broadcast" {
			t.Errorf("Client1 notification method = %q, want %q", notif.Method, "test.broadcast")
		}
	case <-timeout:
		t.Error("Client1 did not receive notification")
	}

	select {
	case notif := <-notifCh2:
		if notif.Method != "test.broadcast" {
			t.Errorf("Client2 notification method = %q, want %q", notif.Method, "test.broadcast")
		}
	case <-timeout:
		t.Error("Client2 did not receive notification")
	}

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Stop(ctx)
}
