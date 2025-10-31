# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-10-31

### Added
- Initial release of ipc-jsonrpc
- JSON-RPC 2.0 server implementation over IPC
- Cross-platform support (Unix sockets, Windows named pipes via winio)
- Type-safe handler system with generic TypedHandler helper
- Middleware support (logging, recovery, timeout)
- Server-to-client notifications and broadcasting
- Concurrent connection and request handling
- Graceful shutdown with context timeout
- Line-delimited JSON codec
- Comprehensive documentation and examples
- Compatible with node-ipc-jsonrpc Node.js package
- dependencies (github.com/Microsoft/go-winio for Windows)

### Changed
- N/A

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- N/A

### Security
- N/A

[Unreleased]: https://github.com/gnana997/ipc-jsonrpc/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/gnana997/ipc-jsonrpc/releases/tag/v0.1.0
