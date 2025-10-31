# VSCode Extension Example

This example demonstrates how to create a VSCode extension that communicates with a Go server using `@gnana997/ipc-jsonrpc` over IPC (Inter-Process Communication).

## Features

This extension showcases production-ready patterns for VSCode extensions:

- ✅ **IPC Communication** - Connect to Go server via Unix sockets/Windows named pipes
- ✅ **Command Palette Integration** - All features accessible via commands
- ✅ **Status Bar Integration** - Visual connection state indicator
- ✅ **Output Channel** - Detailed logging of IPC communication
- ✅ **Progress Notifications** - Native VSCode progress UI for server notifications
- ✅ **Error Handling** - User-friendly error messages
- ✅ **Lifecycle Management** - Proper activation/deactivation with cleanup
- ✅ **Type Safety** - Full TypeScript support

## Architecture

```
┌─────────────────────────────────────┐
│       VSCode Extension              │
│  ┌──────────────────────────────┐  │
│  │  extension.ts                │  │
│  │  - Commands                  │  │
│  │  - UI Integration            │  │
│  └────────────┬─────────────────┘  │
│               │                     │
│  ┌────────────▼─────────────────┐  │
│  │  client.ts                   │  │
│  │  - JSONRPCClient wrapper     │  │
│  │  - Event handling            │  │
│  │  - VSCode integration        │  │
│  └────────────┬─────────────────┘  │
└───────────────┼─────────────────────┘
                │
        IPC (Socket/Pipe)
                │
┌───────────────▼─────────────────────┐
│       Go Echo Server                │
│  - echo method                      │
│  - uppercase method                 │
│  - startNotifications method        │
└─────────────────────────────────────┘
```

## Prerequisites

1. **Go** (1.21 or higher)
2. **Node.js** (18 or higher)
3. **VSCode** (1.80 or higher)

## Setup

### Step 1: Install Dependencies

```bash
cd ipc-jsonrpc/examples/vscode-extension
npm install
```

### Step 2: Build the Extension

```bash
npm run compile
# or
npm run package  # For production build
```

### Step 3: Start the Go Server

In a separate terminal:

```bash
cd ipc-jsonrpc/examples/echo
go run main.go
```

You should see:
```
Starting echo server...
[JSON-RPC] Server listening on \\.\pipe\echo-server  # Windows
# or
[JSON-RPC] Server listening on /tmp/echo-server       # Unix/Mac
```

## Running the Extension

### Option 1: Debug Mode (F5)

1. Open this folder in VSCode
2. Press `F5` to launch Extension Development Host
3. A new VSCode window will open with the extension loaded

### Option 2: Install Locally

```bash
# Package the extension
npm run package

# Install the VSIX (if you create one)
code --install-extension ipc-jsonrpc-vscode-example-0.0.1.vsix
```

## Using the Extension

Once the extension is active in the Extension Development Host:

### 1. Connect to Server

Open the Command Palette (`Ctrl+Shift+P` or `Cmd+Shift+P`) and run:
```
IPC Example: Connect to Server
```

The status bar should update to show `✓ Connected`.

### 2. Echo Message

Run command:
```
IPC Example: Echo Message
```

- Enter a message in the input box
- The server will echo it back
- Result displayed in an information message

### 3. Uppercase Text

Run command:
```
IPC Example: Uppercase Text
```

- Enter text to uppercase
- Server converts it to uppercase
- Result displayed in an information message

### 4. Start Notifications

Run command:
```
IPC Example: Start Notifications
```

- Select notification count (3, 5, 10, or 20)
- Select interval (100ms, 250ms, 500ms, 1000ms)
- Progress notifications will stream from server
- Visual progress bar shown in VSCode
- Progress updates also visible in status bar

### 5. Disconnect

Run command:
```
IPC Example: Disconnect from Server
```

### 6. View Logs

Open the Output panel (`View` → `Output`) and select **IPC JSON-RPC Example** from the dropdown to see detailed logs of all IPC communication.

## File Structure

```
vscode-extension/
├── .vscode/
│   └── launch.json          # Debug configuration
├── src/
│   ├── extension.ts         # Main extension entry point
│   └── client.ts            # IPC client wrapper
├── dist/                    # Built extension (after compile)
├── package.json             # Extension manifest + dependencies
├── tsconfig.json            # TypeScript configuration
├── esbuild.js              # Build script
├── .vscodeignore           # Files to exclude from VSIX
└── README.md               # This file
```

## Key Implementation Details

### Extension Activation

The extension activates on `onStartupFinished`, which means it loads after VSCode finishes starting up. This provides better startup performance.

### Commands

All functionality is exposed through VSCode commands:

| Command | ID | Description |
|---------|----|----|
| Connect to Server | `ipcExample.connect` | Establish IPC connection |
| Disconnect | `ipcExample.disconnect` | Close IPC connection |
| Echo Message | `ipcExample.echo` | Test echo method |
| Uppercase Text | `ipcExample.uppercase` | Test uppercase method |
| Start Notifications | `ipcExample.startNotifications` | Trigger progress notifications |

### Status Bar

The status bar item shows the current connection state:

- 🚫 **Disconnected** - Not connected (click to connect)
- 🔄 **Connecting...** - Connection in progress
- ✅ **Connected** - Successfully connected
- ❌ **Error** - Connection error

### Output Channel

All IPC communication is logged to the **IPC JSON-RPC Example** output channel:

```
[2024-10-31T10:30:00.000Z] Connecting to IPC server...
[2024-10-31T10:30:00.100Z] ✓ Connected to server
[2024-10-31T10:30:05.000Z] → Request: echo "Hello, World!"
[2024-10-31T10:30:05.010Z] ← Response: "Hello, World!"
[2024-10-31T10:30:10.000Z] 📩 Notification: progress - {"current":1,"total":5,"percent":20}
```

### Error Handling

The extension provides user-friendly error messages:

- **Not connected** - Prompts user to connect first
- **Connection failed** - Shows connection error details
- **RPC errors** - Displays server error messages
- **Validation errors** - Shows parameter validation failures

## Development

### Watch Mode

For active development, use watch mode:

```bash
npm run watch
```

Then press `F5` in VSCode to launch Extension Development Host. Changes will automatically recompile.

### TypeScript Compilation

```bash
# Compile once
npm run compile

# Watch for changes
npm run watch
```

### Production Build

```bash
npm run package
```

This creates an optimized, minified bundle in `dist/extension.js`.

## Customization

### Change Socket Path

Edit `src/extension.ts` line 32:

```typescript
client = new IPCClient('your-socket-name', outputChannel, statusBarItem);
```

### Add New Commands

1. Add command to `package.json` under `contributes.commands`
2. Register command in `activate()` function
3. Implement command handler function

Example:

```typescript
// In package.json
{
  "command": "ipcExample.myCommand",
  "title": "IPC Example: My Command"
}

// In extension.ts
context.subscriptions.push(
  vscode.commands.registerCommand('ipcExample.myCommand', async () => {
    const result = await client!.request('myMethod', { param: 'value' });
    vscode.window.showInformationMessage(JSON.stringify(result));
  })
);
```

### Modify UI Elements

Status bar, output channel, and notifications can all be customized in `src/client.ts`.

## Troubleshooting

### Extension doesn't activate

- Check the Output → Extension Host logs
- Ensure `package.json` has correct `engines.vscode` version
- Verify `activationEvents` is configured

### Can't connect to server

- Ensure Go echo server is running (`go run examples/echo/main.go`)
- Check socket path matches between server and client
- On Windows, ensure no other process is using the named pipe
- Check Output → IPC JSON-RPC Example for detailed error logs

### Commands not showing

- Run "Developer: Reload Window" command
- Check `package.json` `contributes.commands` section
- Ensure extension is properly activated (check Extension Host logs)

### Build errors

- Run `npm install` to ensure dependencies are installed
- Ensure `@gnana997/ipc-jsonrpc` package is built: `cd ../../node && npm run build`
- Check TypeScript version compatibility

## Production Deployment

To prepare for production:

1. Update `package.json` with proper metadata (publisher, version, etc.)
2. Run `npm run package` to create optimized build
3. Package as VSIX: `vsce package` (requires `@vscode/vsce`)
4. Publish to VSCode Marketplace or distribute privately

## Use Cases

This example pattern is perfect for:

- **Language Servers** - Connect to native language analysis tools
- **Build Tools** - Integrate with Go/Rust/C++ build systems
- **Native Backends** - Use high-performance Go services from VSCode
- **System Integration** - Access OS-level features through Go
- **Database Tools** - Connect to databases via Go drivers

## Related Examples

- **echo-client** - Simple command-line client example
- **echo server** - Go server implementation

## License

MIT
