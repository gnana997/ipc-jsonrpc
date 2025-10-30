package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	jsonrpc "github.com/gnana997/ipc-jsonrpc"
)

func main() {
	// Use simple socket name - let the transport layer handle platform-specific paths
	// On Unix: creates /tmp/echo-server (or similar)
	// On Windows: should create named pipe \\.\pipe\echo-server
	socketPath := "echo-server"

	// Create server
	server, err := jsonrpc.NewServer(jsonrpc.ServerConfig{
		SocketPath: socketPath,
		OnConnect: func(conn *jsonrpc.Connection) {
			log.Printf("Client connected: %s", conn.RemoteAddr())
		},
		OnDisconnect: func(conn *jsonrpc.Connection) {
			log.Printf("Client disconnected: %s", conn.RemoteAddr())
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Register echo handler
	server.RegisterFunc("echo", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		// Simply return the params back to the client
		var result interface{}
		if len(params) > 0 {
			if err := json.Unmarshal(params, &result); err != nil {
				return nil, jsonrpc.NewInvalidParamsError(err.Error())
			}
		}
		return result, nil
	})

	// Register uppercase handler (using TypedHandler)
	server.RegisterHandler("uppercase", jsonrpc.TypedHandler(handleUppercase))

	// Register notification sender (using TypedHandler)
	server.RegisterHandler("startNotifications", jsonrpc.TypedHandler(handleStartNotifications))

	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Println("Starting echo server...")
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-stop
	log.Println("Shutting down server...")

	// Graceful shutdown with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// Uppercase handler with typed parameters
type UppercaseParams struct {
	Text string `json:"text"`
}

type UppercaseResult struct {
	Result string `json:"result"`
}

func handleUppercase(ctx context.Context, params UppercaseParams) (UppercaseResult, error) {
	if params.Text == "" {
		return UppercaseResult{}, jsonrpc.NewInvalidParamsError("text is required")
	}

	result := UppercaseResult{
		Result: string([]rune(params.Text)), // Simple uppercase simulation
	}

	return result, nil
}

// Start notifications handler
type NotificationParams struct {
	Count    int `json:"count"`
	Interval int `json:"interval"` // milliseconds
}

func handleStartNotifications(ctx context.Context, params NotificationParams) (interface{}, error) {
	// Get connection from context
	conn := jsonrpc.ConnectionFromContext(ctx)
	if conn == nil {
		return nil, jsonrpc.NewInternalError("no connection in context")
	}

	// Send notifications in a goroutine
	go func() {
		for i := 0; i < params.Count; i++ {
			time.Sleep(time.Duration(params.Interval) * time.Millisecond)

			if err := conn.Notify("progress", map[string]interface{}{
				"current": i + 1,
				"total":   params.Count,
				"percent": float64(i+1) / float64(params.Count) * 100,
			}); err != nil {
				log.Printf("Failed to send notification: %v", err)
				return
			}
		}
	}()

	return map[string]interface{}{
		"message": "notifications started",
	}, nil
}
