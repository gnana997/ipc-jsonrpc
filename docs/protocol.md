# Protocol Specification

## JSON-RPC 2.0 over IPC

This document describes the JSON-RPC 2.0 protocol implementation used by `ipc-jsonrpc`.

## Message Format

All messages are **line-delimited JSON** with newline (`\n`) terminator:

```
{"jsonrpc":"2.0","method":"echo","params":{"message":"hello"},"id":1}\n
```

### Line-Delimited JSON

- Each message MUST be a single line
- Each message MUST end with `\n`
- Messages are processed as they arrive
- Empty lines are skipped

## Message Types

### 1. Request (Client → Server)

```json
{
  "jsonrpc": "2.0",
  "method": "methodName",
  "params": { "key": "value" },
  "id": 1
}
```

**Fields:**
- `jsonrpc` (string, required): MUST be exactly "2.0"
- `method` (string, required): The method to invoke
- `params` (any, optional): Parameters for the method
- `id` (number|string, required): Unique request identifier

### 2. Success Response (Server → Client)

```json
{
  "jsonrpc": "2.0",
  "result": { "data": "value" },
  "id": 1
}
```

**Fields:**
- `jsonrpc` (string, required): MUST be exactly "2.0"
- `result` (any, required): The result of the method call
- `id` (number|string, required): Same as request id

### 3. Error Response (Server → Client)

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32601,
    "message": "Method not found",
    "data": { "method": "unknownMethod" }
  },
  "id": 1
}
```

**Fields:**
- `jsonrpc` (string, required): MUST be exactly "2.0"
- `error` (object, required): Error information
  - `code` (number, required): Error code
  - `message` (string, required): Error message
  - `data` (any, optional): Additional error data
- `id` (number|string, required): Same as request id (or null if id couldn't be determined)

### 4. Notification (Server → Client, no response)

```json
{
  "jsonrpc": "2.0",
  "method": "progress",
  "params": { "percentage": 50 }
}
```

**Fields:**
- `jsonrpc` (string, required): MUST be exactly "2.0"
- `method` (string, required): Notification name
- `params` (any, optional): Notification data
- **NO `id` field** - This indicates it's a notification

## Error Codes

### Standard JSON-RPC 2.0 Errors

| Code | Message | Meaning |
|------|---------|---------|
| `-32700` | Parse error | Invalid JSON received |
| `-32600` | Invalid Request | Request object is invalid |
| `-32601` | Method not found | Method doesn't exist |
| `-32602` | Invalid params | Invalid method parameters |
| `-32603` | Internal error | Internal JSON-RPC error |

### Server Error Range

| Code Range | Usage |
|------------|-------|
| `-32000` to `-32099` | Server-defined errors |

### Implementation-Specific Errors

```go
// Go server can define custom errors:
return nil, jsonrpc.NewError(-32001, "Database connection failed", details)
```

```typescript
// Node client receives errors:
try {
  const result = await client.request('getData');
} catch (error) {
  if (error.code === -32001) {
    console.error('Database error:', error.message, error.data);
  }
}
```

## Transport Details

### IPC Mechanism

#### Unix Sockets (Linux/macOS)

- **Default location**: Current working directory
- **Socket path**: Can be absolute or relative
- **Example**: `"/tmp/myapp.sock"` or `"myapp"`
- **Cleanup**: Socket file automatically removed on server start/stop

#### Named Pipes (Windows)

- **Format**: `\\.\pipe\{name}`
- **Auto-prefix**: If path doesn't start with `\\.\pipe\`, it's added automatically
- **Example**: `"myapp"` → `\\.\pipe\myapp`
- **Cleanup**: Automatic by Windows

### Connection Lifecycle

```
Client                          Server
  |                               |
  |-------- Connect ------------->|
  |                               |
  |<------- Connected ------------|
  |                               |
  |---- Request (id: 1) --------->|
  |                               |
  |<--- Response (id: 1) ---------|
  |                               |
  |<--- Notification -------------|
  |                               |
  |---- Request (id: 2) --------->|
  |                               |
  |<--- Error (id: 2) ------------|
  |                               |
  |------- Disconnect ----------->|
```

### Flow Control

- **No** built-in message queuing
- **No** flow control at protocol level
- Relies on IPC transport buffering
- Client should handle backpressure if sending many requests

## Request/Response Matching

### Request ID Rules

1. **Client generates IDs**: Each request must have a unique ID
2. **Server echoes IDs**: Response MUST have same ID as request
3. **ID types**: Can be number or string
4. **Monotonic IDs recommended**: Use incrementing numbers for simplicity

### Concurrent Requests

Multiple requests can be in-flight simultaneously:

```typescript
// Client can send multiple requests without waiting
const [result1, result2, result3] = await Promise.all([
  client.request('method1', params1), // id: 1
  client.request('method2', params2), // id: 2
  client.request('method3', params3), // id: 3
]);
```

Server processes concurrently and responses may arrive out-of-order:

```
Request (id: 1) --->
Request (id: 2) --->
Request (id: 3) --->
                <--- Response (id: 2)
                <--- Response (id: 1)
                <--- Response (id: 3)
```

## Notifications

### Server-to-Client Notifications

Server can push notifications to clients:

```go
// Go server
func handleTask(ctx context.Context, params TaskParams) (interface{}, error) {
    conn := jsonrpc.ConnectionFromContext(ctx)

    // Send progress notification
    conn.Notify("progress", map[string]interface{}{
        "percentage": 50,
        "message": "Processing...",
    })

    return "completed", nil
}
```

```typescript
// Node client
client.on('notification', (method, params) => {
  if (method === 'progress') {
    console.log(`Progress: ${params.percentage}%`);
  }
});
```

### Broadcast to All Clients

```go
// Go server broadcasts to all connected clients
count := server.Broadcast("heartbeat", map[string]interface{}{
    "timestamp": time.Now().Unix(),
    "clients": server.ConnectionCount(),
})
```

### Notification Rules

1. Notifications have **NO** `id` field
2. No response is expected or sent
3. Client MUST NOT reply to notifications
4. Notifications can be sent at any time

## Examples

### Basic Request/Response

**Request:**
```json
{"jsonrpc":"2.0","method":"add","params":{"a":5,"b":3},"id":1}
```

**Response:**
```json
{"jsonrpc":"2.0","result":8,"id":1}
```

### Method Not Found

**Request:**
```json
{"jsonrpc":"2.0","method":"unknownMethod","params":{},"id":2}
```

**Response:**
```json
{
  "jsonrpc":"2.0",
  "error":{
    "code":-32601,
    "message":"Method not found",
    "data":{"method":"unknownMethod"}
  },
  "id":2
}
```

### Server Notification

**Notification:**
```json
{"jsonrpc":"2.0","method":"statusUpdate","params":{"status":"running","cpu":45}}
```

*(No response expected)*

## Implementation Notes

### Go Server

```go
// Handlers can access connection and send notifications
func handler(ctx context.Context, params json.RawMessage) (interface{}, error) {
    // Get connection
    conn := jsonrpc.ConnectionFromContext(ctx)

    // Get request ID
    reqID := jsonrpc.RequestIDFromContext(ctx)

    // Get method name
    method := jsonrpc.MethodFromContext(ctx)

    // Send notification to this client
    conn.Notify("event", data)

    return result, nil
}
```

### TypeScript Client

```typescript
// Configure timeout and debug
const client = new JSONRPCClient({
  socketPath: 'myapp',
  timeout: 30000,    // 30 second timeout
  debug: true,       // Enable debug logging
});

// Handle events
client.on('connect', () => console.log('Connected'));
client.on('disconnect', () => console.log('Disconnected'));
client.on('notification', (method, params) => {
  console.log(`Notification: ${method}`, params);
});
client.on('error', (error) => console.error('Error:', error));

// Make requests with timeout
try {
  const result = await client.request('method', params);
} catch (error) {
  if (error.code === 'TIMEOUT') {
    console.error('Request timed out');
  }
}
```

## Security Considerations

### IPC Security

1. **No encryption**: IPC sockets provide NO encryption by default
2. **Local only**: IPC is designed for same-machine communication
3. **File permissions**: Unix socket permissions control access
4. **Named pipe security**: Windows ACLs control pipe access

### Recommendations

- **Validate all input**: Never trust client data
- **Authentication**: Implement at application layer if needed
- **Rate limiting**: Protect against request flooding
- **Timeout handling**: Prevent resource exhaustion
- **Error messages**: Don't leak sensitive information

### Not Suitable For

- ❌ Remote/network communication (use HTTPS or WebSocket with TLS)
- ❌ Untrusted clients (implement authentication)
- ❌ Sensitive data without encryption (wrap in secure tunnel)

## Compatibility

This implementation is compatible with:
- JSON-RPC 2.0 Specification
- Any JSON-RPC 2.0 client/server over IPC
- Language Server Protocol (LSP) base protocol
- Debug Adapter Protocol (DAP) base protocol

## References

- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- [Language Server Protocol](https://microsoft.github.io/language-server-protocol/)
- [Debug Adapter Protocol](https://microsoft.github.io/debug-adapter-protocol/)
