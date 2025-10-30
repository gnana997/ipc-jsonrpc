# @gnana997/ipc-jsonrpc

Modern TypeScript client for JSON-RPC 2.0 over IPC (Unix sockets/Named Pipes). Designed for communication with Go servers, perfect for VSCode extensions and Electron apps.

[![npm version](https://img.shields.io/npm/v/@gnana997/ipc-jsonrpc.svg)](https://www.npmjs.com/package/@gnana997/ipc-jsonrpc)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Why This Package?

- ✅ **Modern**: Built with TypeScript 5+, ESM + CommonJS support
- ✅ **Zero Dependencies**: Only uses Node.js stdlib
- ✅ **Cross-Platform**: Works on Windows (Named Pipes), macOS, Linux (Unix Sockets)
- ✅ **Type-Safe**: Full TypeScript definitions included
- ✅ **Promise-Based**: Clean async/await API
- ✅ **Event-Driven**: EventEmitter for notifications and lifecycle events
- ✅ **Reliable**: Auto-reconnect, timeout handling, comprehensive error handling
- ✅ **Well-Tested**: Comprehensive test suite with Vitest
- ✅ **Actively Maintained**: Replaces outdated packages (6-9 years old)

## Installation

```bash
npm install @gnana997/ipc-jsonrpc
```

```bash
yarn add @gnana997/ipc-jsonrpc
```

```bash
pnpm add @gnana997/ipc-jsonrpc
```

## Quick Start

```typescript
import { JSONRPCClient } from '@gnana997/ipc-jsonrpc';

// Create client
const client = new JSONRPCClient({
  socketPath: '/tmp/myapp.sock', // Unix/Mac
  // socketPath: 'myapp',         // Windows (auto-converted to \\.\pipe\myapp)
});

// Connect
await client.connect();

// Send request
const result = await client.request('search', {
  query: 'hello world',
  limit: 10,
});

console.log('Results:', result);

// Listen for server notifications
client.on('notification', (method, params) => {
  console.log(`Notification: ${method}`, params);
});

// Disconnect
await client.disconnect();
```

## API Reference

### Constructor

```typescript
new JSONRPCClient(config: ClientConfig)
```

#### ClientConfig

| Option                 | Type      | Default | Description                                      |
| ---------------------- | --------- | ------- | ------------------------------------------------ |
| `socketPath`           | `string`  | -       | IPC socket path (required)                       |
| `connectionTimeout`    | `number`  | 10000   | Connection timeout in ms                         |
| `requestTimeout`       | `number`  | 30000   | Request timeout in ms                            |
| `debug`                | `boolean` | false   | Enable debug logging                             |
| `autoReconnect`        | `boolean` | false   | Auto-reconnect on connection loss                |
| `maxReconnectAttempts` | `number`  | 3       | Maximum reconnection attempts                    |
| `reconnectDelay`       | `number`  | 1000    | Delay between reconnection attempts in ms        |

### Methods

#### `connect(): Promise<void>`

Connect to the IPC server.

```typescript
await client.connect();
```

#### `disconnect(): Promise<void>`

Disconnect from the IPC server.

```typescript
await client.disconnect();
```

#### `request<TResult>(method: string, params?: unknown): Promise<TResult>`

Send a JSON-RPC request and wait for response.

```typescript
const result = await client.request('getData', { id: 123 });
```

#### `notify(method: string, params?: unknown): void`

Send a JSON-RPC notification (no response expected).

```typescript
client.notify('logMessage', { level: 'info', message: 'Hello' });
```

#### `getState(): ConnectionState`

Get current connection state.

```typescript
const state = client.getState();
// 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'closed'
```

#### `isConnected(): boolean`

Check if currently connected.

```typescript
if (client.isConnected()) {
  // do something
}
```

### Events

The client extends EventEmitter and emits the following events:

#### `connected`

Fired when connected to the server.

```typescript
client.on('connected', () => {
  console.log('Connected!');
});
```

#### `disconnected`

Fired when disconnected from the server.

```typescript
client.on('disconnected', () => {
  console.log('Disconnected');
});
```

#### `error`

Fired when an error occurs.

```typescript
client.on('error', (error) => {
  console.error('Error:', error);
});
```

#### `notification`

Fired when a notification is received from the server.

```typescript
client.on('notification', (method, params) => {
  console.log(`Received ${method}:`, params);
});
```

#### `reconnecting`

Fired on reconnection attempt (when `autoReconnect` is enabled).

```typescript
client.on('reconnecting', (attempt) => {
  console.log(`Reconnection attempt ${attempt}`);
});
```

## Platform-Specific Details

### Unix/Linux/macOS

Use absolute paths for Unix sockets:

```typescript
const client = new JSONRPCClient({
  socketPath: '/tmp/myapp.sock',
});
```

### Windows

Use named pipe names (automatically converted to `\\.\pipe\{name}`):

```typescript
const client = new JSONRPCClient({
  socketPath: 'myapp', // Becomes \\.\pipe\myapp
});
```

Or use full pipe path:

```typescript
const client = new JSONRPCClient({
  socketPath: '\\\\.\\pipe\\myapp',
});
```

## Advanced Usage

### Auto-Reconnect

```typescript
const client = new JSONRPCClient({
  socketPath: '/tmp/myapp.sock',
  autoReconnect: true,
  maxReconnectAttempts: 5,
  reconnectDelay: 2000,
});

client.on('reconnecting', (attempt) => {
  console.log(`Reconnection attempt ${attempt}/5`);
});

await client.connect();
```

### Error Handling

The client throws `JSONRPCError` for server errors, which includes the JSON-RPC error code and optional data:

```typescript
import { JSONRPCError } from '@gnana997/ipc-jsonrpc';

try {
  const result = await client.request('getData', { id: 123 });
} catch (error) {
  if (error instanceof JSONRPCError) {
    // JSON-RPC error from server
    console.error(`JSON-RPC Error ${error.code}: ${error.message}`);
    if (error.data) {
      console.error('Additional data:', error.data);
    }

    // Standard JSON-RPC error codes
    if (error.code === -32601) {
      console.error('Method not found');
    } else if (error.code === -32602) {
      console.error('Invalid params');
    }
  } else if (error instanceof Error) {
    // Connection or timeout errors
    if (error.message.includes('timeout')) {
      console.error('Request timed out');
    } else {
      console.error('Error:', error.message);
    }
  }
}
```

### TypeScript Support

Full type safety with generics:

```typescript
interface SearchParams {
  query: string;
  limit: number;
}

interface SearchResult {
  items: Array<{
    id: string;
    title: string;
  }>;
  total: number;
}

const result = await client.request<SearchResult>('search', {
  query: 'test',
  limit: 10,
} as SearchParams);

// result is fully typed!
console.log(result.items[0].title);
```

## Use Cases

### VSCode Extension

Perfect for VSCode extensions that need to communicate with a native server:

```typescript
import * as vscode from 'vscode';
import { JSONRPCClient } from '@gnana997/ipc-jsonrpc';

export function activate(context: vscode.ExtensionContext) {
  const client = new JSONRPCClient({
    socketPath: '/tmp/my-language-server.sock',
    debug: true,
  });

  // Connect on activation
  client.connect().then(() => {
    vscode.window.showInformationMessage('Language server connected');
  });

  // Handle server notifications
  client.on('notification', (method, params) => {
    if (method === 'progress') {
      vscode.window.withProgress(
        {
          location: vscode.ProgressLocation.Notification,
          title: params.title,
        },
        async (progress) => {
          progress.report({ increment: params.percentage });
        }
      );
    }
  });

  // Cleanup on deactivation
  context.subscriptions.push({
    dispose: () => client.disconnect(),
  });
}
```

### Electron App

Communicate between Electron main process and native backend:

```typescript
import { app } from 'electron';
import { JSONRPCClient } from '@gnana997/ipc-jsonrpc';

const client = new JSONRPCClient({
  socketPath: process.platform === 'win32' ? 'myapp' : '/tmp/myapp.sock',
});

app.on('ready', async () => {
  await client.connect();

  // Use the client throughout your app
  const data = await client.request('getData');
});

app.on('quit', () => {
  client.disconnect();
});
```

### Multi-Process Node.js

Coordinate between Node.js processes:

```typescript
// worker.ts
import { JSONRPCClient } from '@gnana997/ipc-jsonrpc';

const client = new JSONRPCClient({
  socketPath: '/tmp/coordinator.sock',
});

await client.connect();

// Report progress
client.notify('workerProgress', {
  workerId: process.pid,
  progress: 50,
});

// Request work
const task = await client.request('getNextTask', {
  workerId: process.pid,
});
```

## Comparison with Other Packages

| Feature                 | @gnana997/ipc-jsonrpc | json-ipc (9 years old) | json-ipc-lib (6 years old) |
| ----------------------- | ------------------------------- | ---------------------- | -------------------------- |
| TypeScript              | ✅ Native                       | ❌                     | ❌                         |
| ESM + CJS               | ✅                              | ❌ CJS only            | ❌ CJS only                |
| Cross-Platform          | ✅                              | ⚠️ Unix only          | ⚠️ Unix only              |
| Auto-Reconnect          | ✅                              | ❌                     | ❌                         |
| Promise-Based           | ✅                              | ❌ Callbacks           | ⚠️ Partial                |
| Active Maintenance      | ✅                              | ❌                     | ❌                         |
| Zero Dependencies       | ✅                              | ✅                     | ✅                         |
| JSON-RPC 2.0 Compliant  | ✅                              | ❌ 1.0                 | ✅                         |
| Notifications Support   | ✅                              | ❌                     | ⚠️ Limited                |
| Timeout Handling        | ✅                              | ❌                     | ❌                         |
| Last Updated            | 2024                            | 2015                   | 2018                       |

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Testing

```bash
# Run tests
npm test

# Watch mode
npm run test:watch

# Coverage
npm run test:coverage
```

## Building

```bash
# Build the package
npm run build

# Development mode (watch)
npm run dev
```

## License

MIT © LLM Copilot Team

## Related Packages

- **Go Server**: [`ipc-jsonrpc`](https://github.com/gnana997/ipc-jsonrpc) - Companion Go server library

## Acknowledgments

This package was created to replace outdated IPC libraries and provide a modern, TypeScript-first solution for JSON-RPC over IPC communication. Special thanks to the open-source community for inspiration and feedback.

---

**Made with ❤️ for the developer community**
