/**
 * node-ipc-jsonrpc
 *
 * Modern TypeScript client for JSON-RPC over IPC (Unix sockets/Named Pipes)
 * Designed for Go servers, VSCode extensions, and Electron apps
 *
 * @packageDocumentation
 */

export { JSONRPCClient, JSONRPCError } from './client.js';
export type {
  ClientConfig,
  ClientEvents,
  ConnectionState,
  JSONRPCError as IJSONRPCError,
  JSONRPCErrorResponse,
  JSONRPCMessage,
  JSONRPCNotification,
  JSONRPCRequest,
  JSONRPCResponse,
  PendingRequest,
} from './types.js';
export { ConnectionState as ConnectionStateEnum } from './types.js';
