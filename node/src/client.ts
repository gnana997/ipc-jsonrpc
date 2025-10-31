import { EventEmitter } from 'node:events';
import * as net from 'node:net';
import * as os from 'node:os';
import type {
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
import { ConnectionState as State } from './types.js';

/**
 * JSON-RPC Error class
 * Extends Error with JSON-RPC error properties
 */
export class JSONRPCError extends Error implements IJSONRPCError {
  code: number;
  data?: unknown;

  constructor(error: IJSONRPCError) {
    super(error.message);
    this.name = 'JSONRPCError';
    this.code = error.code;
    this.data = error.data;

    // Maintain proper stack trace
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, JSONRPCError);
    }
  }
}

/**
 * Type guard to check if a message is an error response
 */
function isErrorResponse(message: JSONRPCMessage): message is JSONRPCErrorResponse {
  return 'error' in message && message.error !== undefined;
}

/**
 * Type guard to check if a message is a success response
 */
function isSuccessResponse(message: JSONRPCMessage): message is JSONRPCResponse {
  return 'result' in message && !('error' in message);
}

/**
 * Type guard to check if a message is a notification
 */
function isNotification(message: JSONRPCMessage): message is JSONRPCNotification {
  return 'method' in message && !('id' in message);
}

/**
 * JSON-RPC Client for IPC communication over Unix sockets/Named Pipes
 *
 * @example
 * ```typescript
 * const client = new JSONRPCClient({ socketPath: '/tmp/myapp.sock' });
 *
 * // Connect
 * await client.connect();
 *
 * // Send request
 * const result = await client.request('search', { query: 'test' });
 *
 * // Listen for notifications
 * client.on('notification', (method, params) => {
 *   console.log('Received:', method, params);
 * });
 *
 * // Disconnect
 * await client.disconnect();
 * ```
 */
export class JSONRPCClient extends EventEmitter<ClientEvents> {
  private config: Required<ClientConfig>;
  private socket: net.Socket | null = null;
  private state: ConnectionState = State.DISCONNECTED;
  private requestId = 0;
  private pendingRequests = new Map<string | number, PendingRequest>();
  private buffer = '';
  private reconnectAttempts = 0;

  constructor(config: ClientConfig) {
    super();
    this.config = {
      socketPath: config.socketPath,
      connectionTimeout: config.connectionTimeout ?? 10000,
      requestTimeout: config.requestTimeout ?? 30000,
      debug: config.debug ?? false,
      autoReconnect: config.autoReconnect ?? false,
      maxReconnectAttempts: config.maxReconnectAttempts ?? 3,
      reconnectDelay: config.reconnectDelay ?? 1000,
    };
  }

  /**
   * Connect to the IPC server
   */
  async connect(): Promise<void> {
    if (this.state === State.CONNECTED) {
      this.log('Already connected');
      return;
    }

    if (this.state === State.CONNECTING) {
      throw new Error('Connection already in progress');
    }

    this.state = State.CONNECTING;
    this.log('Connecting to', this.config.socketPath);

    return new Promise((resolve, reject) => {
      const socket = new net.Socket();
      this.socket = socket;

      const timeout = setTimeout(() => {
        socket.destroy();
        reject(new Error(`Connection timeout after ${this.config.connectionTimeout}ms`));
      }, this.config.connectionTimeout);

      socket.on('connect', () => {
        clearTimeout(timeout);
        this.state = State.CONNECTED;
        this.reconnectAttempts = 0;
        this.log('Connected');
        this.emit('connected');
        resolve();
      });

      socket.on('data', (data) => {
        this.handleData(data);
      });

      socket.on('end', () => {
        this.log('Connection ended');
        this.handleDisconnect();
      });

      socket.on('error', (error) => {
        clearTimeout(timeout);
        this.log('Socket error:', error.message);
        this.emit('error', error);

        if (this.state === State.CONNECTING) {
          reject(error);
        } else {
          this.handleDisconnect();
        }
      });

      socket.on('close', () => {
        this.log('Socket closed');
        this.handleDisconnect();
      });

      // Connect to socket
      const socketPath = this.normalizeSocketPath(this.config.socketPath);
      socket.connect(socketPath);
    });
  }

  /**
   * Disconnect from the IPC server
   */
  async disconnect(): Promise<void> {
    if (this.state === State.DISCONNECTED || this.state === State.CLOSED) {
      return;
    }

    this.state = State.CLOSED;
    this.log('Disconnecting');

    // Reject all pending requests
    for (const [id, pending] of this.pendingRequests.entries()) {
      clearTimeout(pending.timeout);
      pending.reject(new Error('Client disconnected'));
      this.pendingRequests.delete(id);
    }

    if (this.socket) {
      this.socket.destroy();
      this.socket = null;
    }

    this.emit('disconnected');
  }

  /**
   * Send a JSON-RPC request and wait for response
   */
  async request<TResult = unknown>(method: string, params?: unknown): Promise<TResult> {
    if (this.state !== State.CONNECTED) {
      throw new Error('Not connected');
    }

    const id = ++this.requestId;
    const request: JSONRPCRequest = {
      jsonrpc: '2.0',
      method,
      params,
      id,
    };

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.pendingRequests.delete(id);
        reject(new Error(`Request timeout after ${this.config.requestTimeout}ms`));
      }, this.config.requestTimeout);

      this.pendingRequests.set(id, {
        resolve: resolve as (result: unknown) => void,
        reject,
        timeout,
      });

      this.send(request);
    }) as Promise<TResult>;
  }

  /**
   * Send a JSON-RPC notification (no response expected)
   */
  notify(method: string, params?: unknown): void {
    if (this.state !== State.CONNECTED) {
      throw new Error('Not connected');
    }

    const notification: JSONRPCNotification = {
      jsonrpc: '2.0',
      method,
      params,
    };

    this.send(notification);
  }

  /**
   * Get current connection state
   */
  getState(): ConnectionState {
    return this.state;
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.state === State.CONNECTED;
  }

  /**
   * Normalize socket path for platform
   * @private
   */
  private normalizeSocketPath(path: string): string {
    if (os.platform() === 'win32') {
      // If it's already a named pipe path, return as-is
      if (path.startsWith('\\\\.\\pipe\\') || path.startsWith('\\\\?\\pipe\\')) {
        return path;
      }

      // If it's an absolute Windows path (contains drive letter), use as-is for Unix socket
      // Windows 10+ supports Unix domain sockets at file paths
      if (path.includes(':') || path.startsWith('\\') || path.startsWith('/')) {
        return path;
      }

      // For simple names (no path separators), convert to named pipe
      // This matches the Go server behavior with winio
      return `\\\\.\\pipe\\${path}`;
    }

    // Unix/Linux/Mac socket path normalization
    // If already has directory separator, keep it
    if (path.includes('/')) {
      return path;
    }

    // If has .sock extension, prepend /tmp/
    if (path.endsWith('.sock')) {
      return `/tmp/${path}`;
    }

    // Simple name - convert to /tmp/{name}.sock
    return `/tmp/${path}.sock`;
  }

  /**
   * Send a message to the server
   * @private
   */
  private send(message: JSONRPCMessage): void {
    if (!this.socket) {
      throw new Error('Socket not initialized');
    }

    const json = JSON.stringify(message);
    this.log('Sending:', json);
    this.socket.write(`${json}\n`);
  }

  /**
   * Handle incoming data
   * @private
   */
  private handleData(data: Buffer): void {
    this.buffer += data.toString();

    // Process complete JSON lines
    let newlineIndex: number;
    while ((newlineIndex = this.buffer.indexOf('\n')) !== -1) {
      const line = this.buffer.slice(0, newlineIndex).trim();
      this.buffer = this.buffer.slice(newlineIndex + 1);

      if (line.length === 0) continue;

      try {
        const message = JSON.parse(line) as JSONRPCMessage;
        this.handleMessage(message);
      } catch (error) {
        this.log('Failed to parse message:', line, error);
        this.emit('error', error instanceof Error ? error : new Error(String(error)));
      }
    }
  }

  /**
   * Handle a parsed JSON-RPC message
   * @private
   */
  private handleMessage(message: JSONRPCMessage): void {
    this.log('Received:', JSON.stringify(message));

    // Handle error response
    if (isErrorResponse(message)) {
      const pending = this.pendingRequests.get(message.id);
      if (pending) {
        clearTimeout(pending.timeout);
        this.pendingRequests.delete(message.id);
        const error = new JSONRPCError(message.error);
        pending.reject(error);
      }
      return;
    }

    // Handle success response
    if (isSuccessResponse(message)) {
      const pending = this.pendingRequests.get(message.id);
      if (pending) {
        clearTimeout(pending.timeout);
        this.pendingRequests.delete(message.id);
        pending.resolve(message.result);
      }
      return;
    }

    // Handle notification
    if (isNotification(message)) {
      this.emit('notification', message.method, message.params);
      return;
    }

    // Unknown message type
    this.log('Unknown message type:', message);
  }

  /**
   * Handle disconnection
   * @private
   */
  private handleDisconnect(): void {
    if (this.state === State.CLOSED) {
      return;
    }

    const wasConnected = this.state === State.CONNECTED;
    this.state = State.DISCONNECTED;

    // Reject all pending requests
    for (const [id, pending] of this.pendingRequests.entries()) {
      clearTimeout(pending.timeout);
      pending.reject(new Error('Connection lost'));
      this.pendingRequests.delete(id);
    }

    if (wasConnected) {
      this.emit('disconnected');

      // Auto-reconnect if enabled
      if (
        this.config.autoReconnect &&
        this.reconnectAttempts < this.config.maxReconnectAttempts
      ) {
        this.reconnectAttempts++;
        this.state = State.RECONNECTING;
        this.log(`Reconnecting (attempt ${this.reconnectAttempts})`);
        this.emit('reconnecting', this.reconnectAttempts);

        setTimeout(() => {
          this.connect().catch((error) => {
            this.log('Reconnection failed:', error.message);
          });
        }, this.config.reconnectDelay);
      }
    }
  }

  /**
   * Debug logging
   * @private
   */
  private log(...args: unknown[]): void {
    if (this.config.debug) {
      console.log('[JSONRPCClient]', ...args);
    }
  }
}
