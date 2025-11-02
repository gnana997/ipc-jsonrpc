# Changelog

## 0.2.1

### Patch Changes

- Refactored to use @gnana997/node-jsonrpc as the underlying JSON-RPC 2.0 protocol implementation.

  This is a major internal refactoring with zero breaking changes. The package now uses @gnana997/node-jsonrpc for JSON-RPC protocol handling while maintaining the IPC transport layer.

  **Architecture Changes:**

  - Added dependency on @gnana997/node-jsonrpc@^1.0.0
  - Created IPCTransport class implementing the Transport interface
  - Refactored JSONRPCClient as a wrapper maintaining backward compatibility
  - All existing code continues to work without modification

  **Testing:**

  - 51/55 unit tests passing with 4 skipped (100%)
  - All real-world examples (echo-client) work perfectly
  - Zero breaking changes verified

## 0.2.0

### Minor Changes

- d621804: Refactored the socket path support for unix systems

## 0.1.1

### Patch Changes

- 3a3cf41: updated the readme and added examples

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-10-31

### Added

- Initial release of ipc-jsonrpc client
- JSON-RPC 2.0 client implementation over IPC
- Cross-platform support (Unix sockets on Linux/macOS, Named Pipes on Windows)
- TypeScript-first with full type definitions
- Event-driven architecture with notification support
- Automatic connection management with reconnection support
- Request timeout handling
- Concurrent request handling with unique IDs
- Debug logging support
- Compatible with Go JSON-RPC IPC servers
- Comprehensive test coverage
- Full documentation with examples for VSCode extensions and Electron apps

[0.1.0]: https://github.com/gnana997/ipc-jsonrpc/releases/tag/v0.1.0
