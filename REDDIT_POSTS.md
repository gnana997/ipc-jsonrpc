# Reddit Posts for ipc-jsonrpc

This file contains draft Reddit posts for promoting the ipc-jsonrpc library.

---

## Post for r/golang

**Title:** Built a lightweight JSON-RPC IPC library for Go + Node.js

Hey everyone,

I recently published **ipc-jsonrpc** - a simple library for JSON-RPC communication between Go servers and Node.js/TypeScript clients over IPC (Unix sockets / Windows named pipes).

### Why I built this

I was working on a VSCode extension and needed to offload heavy processing to a Go backend. The idea was to keep the Node.js extension responsive while Go handles the data-intensive work.

I looked at existing solutions:

**go-ethereum/rpc + jayson**: These work (I tested them), but:
- go-ethereum/rpc pulls in 20+ dependencies (the entire Ethereum client stack)
- No helper functions for IPC - you manually create `net.Listener` and handle platform differences yourself
- Windows named pipes need extra dependencies (gopkg.in/natefinch/npipe.v2)
- jayson's IPC support isn't well documented (turns out you use `Client.tcp(socketPath)`, not `.ipc()`)

**Other options** like golang-ipc use custom protocols (not JSON-RPC), so existing Node.js JSON-RPC clients won't work.

I wanted something simpler, so I built this from scratch.

### What it provides

**Go side:**
```go
server := jsonrpcipc.NewServer("my-socket")
server.RegisterMethod("process", func(params interface{}) (interface{}, error) {
    // your code here
    return result, nil
})
server.Start()
```

**TypeScript side:**
```typescript
const client = new JSONRPCClient({ socketPath: 'my-socket' });
await client.connect();
const result = await client.request('process', params);
```

**Key features:**
- Zero external dependencies (just go-winio for Windows named pipes)
- Cross-platform out of the box - automatic Unix socket / Windows named pipe selection
- JSON-RPC 2.0 compliant
- Simple API - no manual listener management
- TypeScript client included
- Examples with VSCode extension integration

### Trade-offs

This is NOT a replacement for go-ethereum/rpc if you're building Ethereum stuff. That library has way more features (HTTP, WebSocket, subscriptions, etc.).

This is for simpler use cases where you just want Go + Node.js to talk over a local socket without pulling in a ton of dependencies.

**Links:**
- GitHub: https://github.com/gnana997/ipc-jsonrpc
- Go: `go get github.com/gnana997/ipc-jsonrpc@v0.1.0`
- npm: `npm install node-ipc-jsonrpc`

Happy to hear feedback or suggestions for improvements!

---

## Post for r/node

**Title:** Built a lightweight IPC client to offload heavy work to Go

I recently published a TypeScript client that lets you easily communicate with Go backend processes using JSON-RPC over IPC.

### The problem

I was building a VSCode extension that needed to do some heavy data processing. Running this in Node.js would block the event loop and freeze the UI. I didn't want to deal with N-API native modules, so I decided to run the heavy stuff in a separate Go process via IPC.

### Why not existing solutions?

I tried a few options:

**jayson + go-ethereum/rpc**: This actually works (I tested it), but:
- go-ethereum/rpc is designed for Ethereum nodes and pulls in 20+ dependencies
- You have to manually set up the socket listener in Go
- Windows support requires additional dependencies
- jayson's Unix socket support isn't obvious (`Client.tcp(path)`, not `.ipc()` which doesn't exist)

**Other Node.js IPC libraries**: Most don't do JSON-RPC, or they're abandoned, or they expect Node.js on both ends.

I wanted something simpler and purpose-built, so I made this.

### What it looks like

**TypeScript client:**
```typescript
import { JSONRPCClient } from 'node-ipc-jsonrpc';

const client = new JSONRPCClient({
  socketPath: 'my-service',
  requestTimeout: 30000
});

await client.connect();
const result = await client.request<MyType>('heavyOperation', data);

// Listen for server notifications
client.on('notification', (method, params) => {
  console.log('Progress update:', params);
});
```

**Go server:**
```go
import "github.com/gnana997/ipc-jsonrpc"

server := jsonrpcipc.NewServer("my-service")
server.RegisterMethod("heavyOperation", func(params interface{}) (interface{}, error) {
    // Do expensive work here
    return result, nil
})
server.Start()
```

### Use cases

This works great for:
- VSCode extensions with Go backends
- Electron apps that need native performance for specific tasks
- Any Node.js app that wants to offload CPU-heavy work to Go
- Local-first applications

### Features

- Full TypeScript support with generics
- Works on Windows, Linux, and macOS automatically
- Request timeouts and error handling
- Server-to-client notifications
- Zero dependencies (uses only Node.js stdlib)
- Includes working VSCode extension example

### Links

- npm: `npm install node-ipc-jsonrpc`
- Go server: `go get github.com/gnana997/ipc-jsonrpc@v0.1.0`
- GitHub: https://github.com/gnana997/ipc-jsonrpc

This is my first published npm package, so feedback is welcome!

---

## Testing Notes

Before posting, the following was verified:

### go-ethereum/rpc + jayson Compatibility Test

Created a test project to verify if existing solutions would have worked:

**Results:**
- ✅ jayson successfully connects to Unix sockets via `Client.tcp(socketPath)`
- ✅ JSON-RPC 2.0 protocol is compatible between go-ethereum/rpc and jayson
- ✅ Communication works on Linux/macOS
- ❌ go-ethereum/rpc requires 20+ dependencies (entire Ethereum stack)
- ❌ No cross-platform helper functions provided
- ❌ Windows requires additional dependencies (gopkg.in/natefinch/npipe.v2)
- ❌ `jayson.Client.ipc()` doesn't exist (common misconception)

**Test location:** `C:/MyProjects/geth-jayson-test/`

### Why the Custom Solution Was Justified

1. **Dependency Weight**: go-ethereum/rpc v1.16.5 pulls in cryptography libraries, consensus mechanisms, and system utilities unnecessary for simple IPC
2. **Developer Experience**: No need to manually create `net.Listener` or handle platform differences
3. **Windows Support**: Built-in named pipe support without extra dependencies
4. **Documentation**: Clear examples for the specific use case (Go + Node.js IPC)
5. **Simplicity**: Purpose-built API vs. adapted Ethereum infrastructure

---

## Posting Guidelines

### Timing
- Post to both subreddits on the same day
- Best times: Weekday mornings (EST/PST)

### Subreddit-Specific Tips

**r/golang:**
- Add flair: "Show and Tell" or "Libraries"
- Be ready to discuss design decisions
- Expect questions about performance and architecture

**r/node:**
- Add flair: "Show & Tell" or "Package"
- Emphasize TypeScript support and developer experience
- Be prepared to compare with worker_threads and child_process

### Response Strategy

**If asked "Why not use X?":**
- Acknowledge that X works/exists
- Explain specific pain points encountered
- Focus on use case differences, not superiority

**If asked about performance:**
- Admit no formal benchmarks yet
- Explain it's optimized for developer simplicity, not extreme performance
- Note that IPC overhead is typically negligible for most use cases

**If asked about production readiness:**
- v0.1.0 - use with caution in production
- 85%+ test coverage
- Looking for feedback to improve before 1.0

---

## After Posting

Monitor both threads for:
- Bug reports or issues
- Feature requests
- Questions about usage
- Comparison requests with other libraries

Respond promptly and professionally. Acknowledge valid criticisms.
