/**
 * JSON-RPC 2.0 Types
 */

/** JSON-RPC 2.0 Request */
export interface JSONRPCRequest {
  jsonrpc?: '2.0';
  method: string;
  params?: unknown;
  id: string | number;
}

/** JSON-RPC 2.0 Response (success) */
export interface JSONRPCResponse {
  jsonrpc?: '2.0';
  result: unknown;
  error?: never;
  id: string | number;
}

/** JSON-RPC 2.0 Error object */
export interface JSONRPCError {
  code: number;
  message: string;
  data?: unknown;
}

/** JSON-RPC 2.0 Response (error) */
export interface JSONRPCErrorResponse {
  jsonrpc?: '2.0';
  result?: never;
  error: JSONRPCError;
  id: string | number;
}

/** JSON-RPC 2.0 Notification (no response expected) */
export interface JSONRPCNotification {
  jsonrpc?: '2.0';
  method: string;
  params?: unknown;
  id?: never;
}

/** Union of all possible JSON-RPC messages */
export type JSONRPCMessage =
  | JSONRPCRequest
  | JSONRPCResponse
  | JSONRPCErrorResponse
  | JSONRPCNotification;

/**
 * Client Configuration Options
 */
export interface ClientConfig {
  /**
   * IPC socket path
   * - Unix/Mac: absolute path (e.g., '/tmp/myapp.sock')
   * - Windows: named pipe name (e.g., 'myapp') -> will be converted to '\\.\pipe\myapp'
   * - Or full Windows pipe path (e.g., '\\\\.\\pipe\\myapp')
   */
  socketPath: string;

  /**
   * Connection timeout in milliseconds
   * @default 10000 (10 seconds)
   */
  connectionTimeout?: number;

  /**
   * Request timeout in milliseconds
   * @default 30000 (30 seconds)
   */
  requestTimeout?: number;

  /**
   * Enable debug logging
   * @default false
   */
  debug?: boolean;

  /**
   * Auto-reconnect on connection loss
   * @default false
   */
  autoReconnect?: boolean;

  /**
   * Maximum reconnection attempts
   * @default 3
   */
  maxReconnectAttempts?: number;

  /**
   * Delay between reconnection attempts in milliseconds
   * @default 1000
   */
  reconnectDelay?: number;
}

/**
 * Connection Events
 *
 * Event parameter arrays for EventEmitter
 */
export interface ClientEvents {
  /** Fired when connected to the server */
  connected: [];

  /** Fired when disconnected from the server */
  disconnected: [];

  /** Fired when a connection error occurs */
  error: [error: Error];

  /** Fired when a notification is received from the server */
  notification: [method: string, params: unknown];

  /** Fired on reconnection attempt */
  reconnecting: [attempt: number];
}

/**
 * Pending Request
 */
export interface PendingRequest {
  resolve: (result: unknown) => void;
  reject: (error: Error) => void;
  timeout: NodeJS.Timeout;
}

/**
 * Connection State
 */
export enum ConnectionState {
  DISCONNECTED = 'disconnected',
  CONNECTING = 'connecting',
  CONNECTED = 'connected',
  RECONNECTING = 'reconnecting',
  CLOSED = 'closed',
}
