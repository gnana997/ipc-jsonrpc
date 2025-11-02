import { EventEmitter } from 'node:events';
import { JSONRPCError as BaseJSONRPCError } from '@gnana997/node-jsonrpc';
import { JSONRPCClient as BaseJSONRPCClient } from '@gnana997/node-jsonrpc/client';
import { IPCTransport } from './transport.js';
import type {
  ClientConfig,
  ClientEvents,
  ConnectionState,
  JSONRPCError as IJSONRPCError,
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
 * JSON-RPC Client for IPC communication over Unix sockets/Named Pipes
 *
 * This is a wrapper around @gnana997/node-jsonrpc's JSONRPCClient that provides
 * IPC-specific transport and maintains backward compatibility with the original API.
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
  private transport: IPCTransport;
  private client: BaseJSONRPCClient;
  private state: ConnectionState = State.DISCONNECTED;
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

    // Create IPC transport
    this.transport = new IPCTransport({
      socketPath: this.config.socketPath,
      connectionTimeout: this.config.connectionTimeout,
      debug: this.config.debug,
    });

    // Create JSON-RPC client with IPC transport
    this.client = new BaseJSONRPCClient({
      transport: this.transport,
      requestTimeout: this.config.requestTimeout,
      debug: this.config.debug,
    });

    // Forward events from underlying client
    this.setupEventForwarding();
  }

  /**
   * Setup event forwarding from underlying client to this wrapper
   * @private
   */
  private setupEventForwarding(): void {
    // Forward connected event
    this.client.on('connected', () => {
      this.state = State.CONNECTED;
      this.reconnectAttempts = 0;
      this.emit('connected');
    });

    // Forward disconnected event
    this.client.on('disconnected', () => {
      const wasConnected = this.state === State.CONNECTED;
      this.state = State.DISCONNECTED;
      this.emit('disconnected');

      // Auto-reconnect if enabled and was previously connected
      if (
        wasConnected &&
        this.config.autoReconnect &&
        this.reconnectAttempts < this.config.maxReconnectAttempts
      ) {
        this.reconnectAttempts++;
        this.state = State.RECONNECTING;
        this.emit('reconnecting', this.reconnectAttempts);

        setTimeout(() => {
          this.connect().catch((error) => {
            if (this.config.debug) {
              console.log('[JSONRPCClient] Reconnection failed:', error.message);
            }
          });
        }, this.config.reconnectDelay);
      }
    });

    // Forward notification event
    this.client.on('notification', (method, params) => {
      this.emit('notification', method, params);
    });

    // Forward error event
    this.client.on('error', (error) => {
      this.emit('error', error);
    });
  }

  /**
   * Connect to the IPC server
   */
  async connect(): Promise<void> {
    if (this.state === State.CONNECTED) {
      if (this.config.debug) {
        console.log('[JSONRPCClient] Already connected');
      }
      return;
    }

    if (this.state === State.CONNECTING) {
      throw new Error('Connection already in progress');
    }

    this.state = State.CONNECTING;
    try {
      await this.client.connect();
      // Ensure state is updated and event is emitted
      // Event handler may have already set state to CONNECTED
      if (this.client.isConnected()) {
        if ((this.state as ConnectionState) !== State.CONNECTED) {
          this.state = State.CONNECTED;
          this.reconnectAttempts = 0;
          this.emit('connected');
        }
      }
    } catch (error) {
      this.state = State.DISCONNECTED;
      throw error;
    }
  }

  /**
   * Disconnect from the IPC server
   */
  async disconnect(): Promise<void> {
    if (this.state === State.DISCONNECTED || this.state === State.CLOSED) {
      return;
    }

    const wasConnected = this.state === State.CONNECTED;
    this.state = State.CLOSED;
    await this.client.disconnect();

    // Emit disconnected event if we were connected
    if (wasConnected) {
      this.emit('disconnected');
    }
  }

  /**
   * Send a JSON-RPC request and wait for response
   */
  async request<TResult = unknown>(method: string, params?: unknown): Promise<TResult> {
    try {
      return await this.client.request<TResult>(method, params);
    } catch (error) {
      throw this.transformError(error);
    }
  }

  /**
   * Send a JSON-RPC notification (no response expected)
   */
  notify(method: string, params?: unknown): void {
    try {
      this.client.notify(method, params);
    } catch (error) {
      throw this.transformError(error);
    }
  }

  /**
   * Transform errors from underlying client to maintain backward compatibility
   * @private
   */
  private transformError(error: unknown): Error {
    if (!(error instanceof Error)) {
      return error as Error;
    }

    // Convert BaseJSONRPCError to local JSONRPCError for backward compatibility
    if (error instanceof BaseJSONRPCError) {
      return new JSONRPCError({
        code: error.code,
        message: error.message,
        data: error.data,
      });
    }

    // Transform error messages to match original client
    let message = error.message;

    // "Client not connected" → "Not connected"
    if (message === 'Client not connected') {
      message = 'Not connected';
    }

    // "Request N timed out after Xms" → "Request timeout after Xms"
    const timeoutMatch = message.match(/Request \d+ timed out after (\d+)ms/);
    if (timeoutMatch) {
      message = `Request timeout after ${timeoutMatch[1]}ms`;
    }

    // Create a new error with the transformed message
    const transformedError = new Error(message);
    transformedError.name = error.name;
    transformedError.stack = error.stack;
    return transformedError;
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
}
