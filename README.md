# ipc-jsonrpc

[![Go Reference](https://pkg.go.dev/badge/github.com/gnana997/ipc-jsonrpc.svg)](https://pkg.go.dev/github.com/gnana997/ipc-jsonrpc)
[![npm version](https://img.shields.io/npm/v/node-ipc-jsonrpc.svg)](https://www.npmjs.com/package/node-ipc-jsonrpc)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**JSON-RPC 2.0 over IPC** - A complete solution for inter-process communication using JSON-RPC 2.0 protocol over Unix sockets and Windows named pipes.

## 📦 Packages

| Package | Language | Description | Docs |
|---------|----------|-------------|------|
| [`github.com/gnana997/ipc-jsonrpc`](https://pkg.go.dev/github.com/gnana997/ipc-jsonrpc) | Go | JSON-RPC 2.0 server implementation | [Go Documentation](./GO_README.md) |
| [`node-ipc-jsonrpc`](https://www.npmjs.com/package/node-ipc-jsonrpc) | TypeScript/Node.js | JSON-RPC 2.0 client library | [Node Documentation](./node/README.md) |

## ✨ Features

### Server (Go)
- ✅ **JSON-RPC 2.0 Compliant** - Full specification implementation
- ✅ **Cross-Platform** - Unix sockets (Linux/macOS) and Named Pipes (Windows)
- ✅ **Zero Dependencies** - Uses only Go standard library
- ✅ **Type-Safe Handlers** - Typed handler helpers for compile-time safety
- ✅ **Concurrent** - Handles multiple clients and requests concurrently
- ✅ **Middleware Support** - Chain middleware for logging, auth, recovery
- ✅ **Server Notifications** - Push notifications to connected clients
- ✅ **Graceful Shutdown** - Clean shutdown with context timeout

### Client (Node.js)
- ✅ **TypeScript-First** - Full type definitions and safety
- ✅ **Modern ESM + CJS** - Works with both module systems
- ✅ **Event-Driven** - Subscribe to notifications from server
- ✅ **Auto-Reconnect** - Automatic reconnection with backoff
- ✅ **Request Timeouts** - Configurable timeout handling
- ✅ **Debug Logging** - Built-in debug logging support
- ✅ **Zero Dependencies** - Minimal footprint
- ✅ **Well-Tested** - Comprehensive test coverage

## 🚀 Quick Start

### Go Server

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

### Node.js Client

```typescript
import { JSONRPCClient } from 'node-ipc-jsonrpc';

// Create client
const client = new JSONRPCClient({
  socketPath: 'myapp', // Same as Go server
});

// Connect to server
await client.connect();

// Send request
const result = await client.request('echo', {
  message: 'Hello from Node.js!',
});

console.log('Server response:', result);

// Listen for notifications
client.on('notification', (method, params) => {
  console.log(`Notification: ${method}`, params);
});

// Disconnect when done
await client.disconnect();
```

## 📖 Documentation

- **[Go Server Documentation](./GO_README.md)** - Complete Go API reference
- **[Node.js Client Documentation](./node/README.md)** - Complete TypeScript API reference
- **[Examples](./examples/)** - End-to-end examples
- **[CHANGELOG](./CHANGELOG.md)** - Version history

## 💡 Use Cases

### VSCode Extensions
Build VSCode extensions with Go backends:

```typescript
// Extension communicates with Go server via IPC
const client = new JSONRPCClient({ socketPath: 'my-vscode-extension' });
await client.connect();

const diagnostics = await client.request('analyzecode', {
  file: document.fileName,
  content: document.getText(),
});
```

### Electron Apps
Connect Electron apps to native Go services:

```typescript
// Main process communicates with Go backend
const backendClient = new JSONRPCClient({
  socketPath: 'electron-backend'
});

ipcMain.handle('backend-request', async (event, method, params) => {
  return await backendClient.request(method, params);
});
```

### Language Servers
Implement language servers with Go:

```go
server.RegisterHandler("textDocument/completion",
  jsonrpc.TypedHandler(handleCompletion))

server.RegisterHandler("textDocument/hover",
  jsonrpc.TypedHandler(handleHover))
```

## 🏗️ Examples

- **[Basic Echo Server](./examples/echo/)** - Simple request/response example
- **[Middleware](./examples/echo/)** - Authentication, logging, recovery

## 🛠️ Development

### Prerequisites
- Go 1.21+
- Node.js 18+
- npm 9+

### Setup

```bash
# Clone repository
git clone https://github.com/gnana997/ipc-jsonrpc.git
cd ipc-jsonrpc

# Install Node.js dependencies
npm install

# Build Node.js package
npm run build

# Run all tests
npm test
```

### Running Tests

```bash
# Go tests
npm run test:go
# or: go test -v ./...

# Node.js tests
npm run test:node
# or: npm test --workspace=node

# All tests
npm test
```

### Building

```bash
# Build Node.js package
npm run build

# Build Go (optional - source-only distribution)
go build ./...
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

### Quick Contribution Guide

1. Fork the repository
2. Create your feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Run tests: `npm test`
5. Commit: `git commit -m 'Add amazing feature'`
6. Push: `git push origin feature/amazing-feature`
7. Open a Pull Request

## 📄 License

MIT © LLM Copilot Team

## 🔗 Links

- **Go Documentation**: https://pkg.go.dev/github.com/gnana997/ipc-jsonrpc
- **npm Package**: https://www.npmjs.com/package/node-ipc-jsonrpc
- **Issues**: https://github.com/gnana997/ipc-jsonrpc/issues
- **Discussions**: https://github.com/gnana997/ipc-jsonrpc/discussions

## 🌟 Star History

If you find this project useful, please consider giving it a star! ⭐

---

**Made with ❤️ for the Go and Node.js communities**
