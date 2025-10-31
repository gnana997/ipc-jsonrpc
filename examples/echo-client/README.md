# Echo Client Example

This example demonstrates how to use the `@gnana997/ipc-jsonrpc` Node.js/TypeScript client to communicate with the Go echo server.

## Features Demonstrated

- ✅ Connecting to IPC server (Unix sockets/Windows named pipes)
- ✅ Calling RPC methods with different parameter types
- ✅ Handling server responses
- ✅ Receiving server-initiated notifications
- ✅ Event-driven architecture
- ✅ Error handling
- ✅ Graceful connection management

## Prerequisites

1. **Go** (1.21 or higher)
2. **Node.js** (18 or higher)
3. **npm** or **pnpm**

## Running the Example

### Step 1: Start the Go Echo Server

In one terminal, navigate to the echo server directory and start it:

```bash
cd ipc-jsonrpc/examples/echo
go run main.go
```

You should see:
```
2024/10/31 05:13:29 Starting echo server...
2024/10/31 05:13:29 [JSON-RPC] Server listening on \\.\pipe\echo-server  # Windows
# or
2024/10/31 05:13:29 [JSON-RPC] Server listening on /tmp/echo-server       # Unix/Mac
```

### Step 2: Install Client Dependencies

In another terminal, navigate to the echo-client directory:

```bash
cd ipc-jsonrpc/examples/echo-client
npm install
```

### Step 3: Run the Client

```bash
npm start
```

## Expected Output

The client will demonstrate three main features:

### Demo 1: Echo Method
Tests the echo method with various data types (string, object, array, null):
```
Sending string...
Response: "Hello, World!"

Sending object...
Response: {
  "message": "Hello from client",
  "timestamp": "2024-10-31T10:30:00.000Z",
  "nested": { "value": 42 }
}
```

### Demo 2: Uppercase Method
Tests the typed handler with validation:
```
Sending text to uppercase...
Response: {"result":"HELLO WORLD"}

Testing validation (empty text)...
✓ Validation error caught: Invalid params
```

### Demo 3: Server Notifications
Tests asynchronous server-to-client notifications:
```
Starting notification sequence...
Progress [████░░░░░░░░░░░░░░░░] 20.0% (1/5)
Progress [████████░░░░░░░░░░░░] 40.0% (2/5)
Progress [████████████░░░░░░░░] 60.0% (3/5)
Progress [████████████████░░░░] 80.0% (4/5)
Progress [████████████████████] 100.0% (5/5)

✓ Received 5 notifications
```

## Code Structure

```
echo-client/
├── package.json      # Dependencies and scripts
├── tsconfig.json     # TypeScript configuration
├── src/
│   └── index.ts     # Main client implementation
└── README.md        # This file
```

## What the Client Does

1. **Creates a JSON-RPC client** with connection configuration
2. **Sets up event listeners**:
   - `connected` - Called when connection is established
   - `disconnected` - Called when connection is lost
   - `error` - Called on errors
   - `notification` - Called when server sends notifications
3. **Connects to the server** using IPC (Unix socket or Windows named pipe)
4. **Calls RPC methods**:
   - `echo` - Returns the exact input (any type)
   - `uppercase` - Returns uppercased text (typed parameters)
   - `startNotifications` - Triggers server to send progress notifications
5. **Handles responses** with full type safety
6. **Disconnects gracefully** when done

## API Usage Patterns

### Basic Request
```typescript
const result = await client.request('methodName', params);
```

### Typed Request
```typescript
interface Result {
  value: string;
}

const result = await client.request<Result>('methodName', params);
console.log(result.value); // Type-safe!
```

### Notifications
```typescript
client.on('notification', (method, params) => {
  console.log(`Received ${method}:`, params);
});
```

### Error Handling
```typescript
try {
  await client.request('method', params);
} catch (error) {
  if (error instanceof JSONRPCError) {
    console.error(`RPC Error ${error.code}: ${error.message}`);
  } else {
    console.error('Connection error:', error.message);
  }
}
```

## Cross-Platform Support

The client automatically handles platform-specific socket paths:

- **Unix/Linux/macOS**: Uses Unix domain sockets (e.g., `/tmp/echo-server`)
- **Windows**: Uses named pipes (e.g., `\\.\pipe\echo-server`)

You only need to specify the simple socket name: `'echo-server'`

## Troubleshooting

### "Connection refused" or "ENOENT"
- Make sure the Go echo server is running first
- Check that the socket path matches (`echo-server`)
- On Windows, ensure no other process is using the named pipe

### "Request timeout"
- The server might be slow to respond
- Increase `requestTimeout` in client config
- Check server logs for errors

### TypeScript errors
- Run `npm install` to ensure all dependencies are installed
- Make sure the `@gnana997/ipc-jsonrpc` package is built: `cd ../../node && npm run build`

## Next Steps

- Try modifying the parameters sent to methods
- Add your own RPC methods to the server
- Explore the VSCode extension example for a more complex use case
- Check out the `@gnana997/ipc-jsonrpc` [documentation](../../node/README.md)

## License

MIT
