# ipc-jsonrpc

[![Go Reference](https://pkg.go.dev/badge/github.com/gnana997/ipc-jsonrpc.svg)](https://pkg.go.dev/github.com/gnana997/ipc-jsonrpc)
[![Go Report Card](https://goreportcard.com/badge/github.com/gnana997/ipc-jsonrpc)](https://goreportcard.com/report/github.com/gnana997/ipc-jsonrpc)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A pure Go implementation of JSON-RPC 2.0 server over IPC (Unix sockets and Windows named pipes).

## Features

✅ **JSON-RPC 2.0 Compliant** - Full implementation of the JSON-RPC 2.0 specification
✅ **Cross-Platform** - Works on Windows (Named Pipes), macOS, Linux (Unix Sockets)
✅ **Zero Dependencies** - Uses only Go standard library
✅ **Type-Safe** - Typed handler helpers for compile-time safety
✅ **Concurrent** - Handles multiple clients and requests concurrently
✅ **Middleware Support** - Chain middleware for logging, auth, recovery, etc.
✅ **Notifications** - Server can push notifications to clients
✅ **Graceful Shutdown** - Clean shutdown with context timeout
✅ **Well-Tested** - Comprehensive test coverage
✅ **Compatible** - Works with [node-ipc-jsonrpc](https://www.npmjs.com/package/node-ipc-jsonrpc) Node.js package

## Installation

```bash
go get github.com/gnana997/ipc-jsonrpc
```

## Quick Start

```go
package main

import (
    "context"
    "encoding/json"
    "log"

    jsonrpc "github.com/gnana997/ipc-jsonrpc"
)

func main() {
    // Create server
    server, err := jsonrpc.NewServer(jsonrpc.ServerConfig{
        SocketPath: "myapp", // Windows: \\.\pipe\myapp, Unix: myapp
    })
    if err != nil {
        log.Fatal(err)
    }

    // Register handler
    server.RegisterFunc("echo", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
        return string(params), nil
    })

    // Start server (blocks)
    log.Println("Starting server...")
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}
```

## API Reference

### Creating a Server

```go
server, err := jsonrpc.NewServer(jsonrpc.ServerConfig{
    SocketPath: "myapp",  // Required

    // Optional callbacks
    OnConnect: func(conn *jsonrpc.Connection) {
        log.Printf("Client connected: %s", conn.RemoteAddr())
    },
    OnDisconnect: func(conn *jsonrpc.Connection) {
        log.Printf("Client disconnected: %s", conn.RemoteAddr())
    },
    OnError: func(err error) {
        log.Printf("Error: %v", err)
    },
})
```

### Registering Handlers

#### Basic Handler

```go
server.RegisterFunc("method", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
    var p MyParams
    if err := json.Unmarshal(params, &p); err != nil {
        return nil, jsonrpc.NewInvalidParamsError(err.Error())
    }

    // Process request...
    return MyResult{Data: "value"}, nil
})
```

#### Typed Handler (Recommended)

```go
type SearchParams struct {
    Query string `json:"query"`
    Limit int    `json:"limit"`
}

type SearchResult struct {
    Items []string `json:"items"`
    Total int      `json:"total"`
}

func handleSearch(ctx context.Context, params SearchParams) (SearchResult, error) {
    // Type-safe parameter access!
    items := performSearch(params.Query, params.Limit)
    return SearchResult{Items: items, Total: len(items)}, nil
}

server.RegisterHandler("search", jsonrpc.TypedHandler(handleSearch))
```

### Middleware

```go
// Logging middleware
server.RegisterMiddleware(jsonrpc.LoggingMiddleware(func(method string, duration time.Duration, err error) {
    if err != nil {
        log.Printf("[ERROR] %s took %v: %v", method, duration, err)
    } else {
        log.Printf("[SUCCESS] %s took %v", method, duration)
    }
}))

// Recovery middleware (catches panics)
server.RegisterMiddleware(jsonrpc.RecoveryMiddleware())

// Timeout middleware
server.RegisterMiddleware(jsonrpc.TimeoutMiddleware(30 * time.Second))

// Custom middleware
server.RegisterMiddleware(func(next jsonrpc.Handler) jsonrpc.Handler {
    return jsonrpc.HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
        // Pre-processing
        method := jsonrpc.MethodFromContext(ctx)
        log.Printf("Handling: %s", method)

        // Call next handler
        result, err := next.Handle(ctx, params)

        // Post-processing
        return result, err
    })
})
```

### Notifications

Notifications are one-way messages from server to client (no response expected).

#### Send to Specific Client

```go
func handleLongTask(ctx context.Context, params TaskParams) (interface{}, error) {
    conn := jsonrpc.ConnectionFromContext(ctx)

    for i := 0; i < 100; i++ {
        // Send progress notification
        conn.Notify("progress", map[string]interface{}{
            "percentage": i,
            "message":    "Processing...",
        })

        // Do work...
        time.Sleep(100 * time.Millisecond)
    }

    return "completed", nil
}
```

#### Broadcast to All Clients

```go
// In a goroutine, send updates to all connected clients
go func() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        count := server.Broadcast("heartbeat", map[string]interface{}{
            "timestamp": time.Now().Unix(),
            "clients":   server.ConnectionCount(),
        })
        log.Printf("Sent heartbeat to %d clients", count)
    }
}()
```

### Error Handling

The package provides helpers for standard JSON-RPC errors:

```go
// Standard errors
return nil, jsonrpc.NewParseError(data)
return nil, jsonrpc.NewInvalidRequestError(data)
return nil, jsonrpc.NewMethodNotFoundError("unknownMethod")
return nil, jsonrpc.NewInvalidParamsError("missing field: query")
return nil, jsonrpc.NewInternalError(data)

// Custom errors
return nil, jsonrpc.NewError(-32001, "Database connection failed", details)

// Wrap Go errors
if err := db.Query(); err != nil {
    return nil, jsonrpc.WrapError(jsonrpc.InternalError, "Query failed", err)
}
```

### Graceful Shutdown

```go
// Setup signal handling
stop := make(chan os.Signal, 1)
signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

// Start server in goroutine
go server.Start()

// Wait for shutdown signal
<-stop

// Graceful shutdown with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := server.Stop(ctx); err != nil {
    log.Printf("Shutdown error: %v", err)
}
```

### Context Values

Access request metadata from context:

```go
func handleRequest(ctx context.Context, params json.RawMessage) (interface{}, error) {
    // Get method name
    method := jsonrpc.MethodFromContext(ctx)

    // Get request ID
    requestID := jsonrpc.RequestIDFromContext(ctx)

    // Get connection
    conn := jsonrpc.ConnectionFromContext(ctx)

    // Use context for cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Process request...
    }

    return result, nil
}
```

## Platform-Specific Behavior

### Unix/Linux/macOS

Uses Unix domain sockets. Socket path is used as-is:

```go
server, _ := jsonrpc.NewServer(jsonrpc.ServerConfig{
    SocketPath: "/tmp/myapp.sock",
})
// Listens on: /tmp/myapp.sock
```

The socket file is automatically removed when the server starts and when it stops.

### Windows

Uses Named Pipes. Path is automatically prefixed if needed:

```go
server, _ := jsonrpc.NewServer(jsonrpc.ServerConfig{
    SocketPath: "myapp",
})
// Listens on: \\.\pipe\myapp

// Or use full path:
server, _ := jsonrpc.NewServer(jsonrpc.ServerConfig{
    SocketPath: `\\.\pipe\myapp`,
})
// Listens on: \\.\pipe\myapp
```

Named pipes are automatically cleaned up by Windows.

## Example: Complete Server

See [examples/echo/main.go](examples/echo/main.go) for a complete working example.

```go
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
    server, err := jsonrpc.NewServer(jsonrpc.ServerConfig{
        SocketPath: "myapp",
        OnConnect: func(conn *jsonrpc.Connection) {
            log.Printf("Client connected: %s", conn.RemoteAddr())
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // Register handlers
    server.RegisterHandler("echo", jsonrpc.TypedHandler(handleEcho))
    server.RegisterHandler("search", jsonrpc.TypedHandler(handleSearch))

    // Add middleware
    server.RegisterMiddleware(jsonrpc.LoggingMiddleware(logRequest))
    server.RegisterMiddleware(jsonrpc.RecoveryMiddleware())

    // Graceful shutdown
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

    go func() {
        if err := server.Start(); err != nil {
            log.Fatal(err)
        }
    }()

    <-stop
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    server.Stop(ctx)
}

type EchoParams struct {
    Message string `json:"message"`
}

func handleEcho(ctx context.Context, params EchoParams) (string, error) {
    return params.Message, nil
}

type SearchParams struct {
    Query string `json:"query"`
    Limit int    `json:"limit"`
}

type SearchResult struct {
    Items []string `json:"items"`
    Total int      `json:"total"`
}

func handleSearch(ctx context.Context, params SearchParams) (SearchResult, error) {
    // Perform search...
    return SearchResult{
        Items: []string{"result1", "result2"},
        Total: 2,
    }, nil
}

func logRequest(method string, duration time.Duration, err error) {
    if err != nil {
        log.Printf("[%s] %v (took %v)", method, err, duration)
    } else {
        log.Printf("[%s] success (took %v)", method, duration)
    }
}
```

## Testing with Node.js Client

This server is compatible with the [node-ipc-jsonrpc](https://www.npmjs.com/package/node-ipc-jsonrpc) Node.js package:

```typescript
import { JSONRPCClient } from 'node-ipc-jsonrpc';

const client = new JSONRPCClient({
  socketPath: 'myapp', // Same as Go server
});

await client.connect();

// Send request
const result = await client.request('search', {
  query: 'test',
  limit: 10,
});

console.log('Results:', result);

// Listen for notifications
client.on('notification', (method, params) => {
  console.log(`Notification: ${method}`, params);
});

await client.disconnect();
```

## Protocol Details

### Message Format

Line-delimited JSON with newline (`\n`) terminator:

```
{"jsonrpc":"2.0","method":"echo","params":{"message":"hello"},"id":1}\n
```

### Message Types

**Request** (Client → Server):
```json
{
  "jsonrpc": "2.0",
  "method": "search",
  "params": {"query": "test"},
  "id": 1
}
```

**Success Response** (Server → Client):
```json
{
  "jsonrpc": "2.0",
  "result": {"items": ["result1"]},
  "id": 1
}
```

**Error Response** (Server → Client):
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32601,
    "message": "Method not found",
    "data": {"method": "unknownMethod"}
  },
  "id": 1
}
```

**Notification** (Server → Client, no response):
```json
{
  "jsonrpc": "2.0",
  "method": "progress",
  "params": {"percentage": 50}
}
```

### Standard Error Codes

- `-32700` - Parse error (invalid JSON)
- `-32600` - Invalid Request
- `-32601` - Method not found
- `-32602` - Invalid params
- `-32603` - Internal error
- `-32000` to `-32099` - Server errors (reserved)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT © gnana997

## Related Packages

- **Node.js Client**: [node-ipc-jsonrpc](https://www.npmjs.com/package/node-ipc-jsonrpc)

---

**Made with ❤️ for the Go community**
