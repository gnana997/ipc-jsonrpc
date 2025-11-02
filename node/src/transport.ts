import { EventEmitter } from 'node:events';
import * as net from 'node:net';
import * as os from 'node:os';
import type { Transport } from '@gnana997/node-jsonrpc/transport';

export interface IPCTransportConfig {
  /**
   * Socket path for IPC connection
   * - Unix/Linux/Mac: Path to Unix socket (e.g., '/tmp/myapp.sock')
   * - Windows: Named pipe name (e.g., '\\\\.\\pipe\\myapp') or simple name (e.g., 'myapp')
   */
  socketPath: string;

  /**
   * Connection timeout in milliseconds
   * @default 10000 (10 seconds)
   */
  connectionTimeout?: number;

  /**
   * Enable debug logging
   * @default false
   */
  debug?: boolean;
}

/**
 * IPC Transport implementation for JSON-RPC over Unix sockets and Windows named pipes
 *
 * Implements the Transport interface from @gnana997/node-jsonrpc for IPC communication.
 * Handles platform-specific socket path normalization and line-delimited message framing.
 *
 * @example
 * ```typescript
 * import { JSONRPCClient } from '@gnana997/node-jsonrpc';
 * import { IPCTransport } from 'node-ipc-jsonrpc';
 *
 * const transport = new IPCTransport({ socketPath: 'myapp' });
 * const client = new JSONRPCClient({ transport });
 *
 * await client.connect();
 * const result = await client.request('method', params);
 * ```
 */
export class IPCTransport extends EventEmitter implements Transport {
  private config: Required<IPCTransportConfig>;
  private socket: net.Socket | null = null;
  private buffer = '';
  private connected = false;

  constructor(config: IPCTransportConfig) {
    super();
    this.config = {
      socketPath: config.socketPath,
      connectionTimeout: config.connectionTimeout ?? 10000,
      debug: config.debug ?? false,
    };
  }

  /**
   * Connect to the IPC server
   */
  async connect(): Promise<void> {
    if (this.connected) {
      this.log('Already connected');
      return;
    }

    this.log('Connecting to', this.config.socketPath);

    return new Promise((resolve, reject) => {
      const socket = new net.Socket();
      this.socket = socket;

      const timeout = setTimeout(() => {
        socket.destroy();
        this.connected = false;
        reject(new Error(`Connection timeout after ${this.config.connectionTimeout}ms`));
      }, this.config.connectionTimeout);

      socket.on('connect', () => {
        clearTimeout(timeout);
        this.connected = true;
        this.log('Connected');
        resolve();
      });

      socket.on('data', (data) => {
        this.handleData(data);
      });

      socket.on('error', (error) => {
        clearTimeout(timeout);
        this.log('Socket error:', error.message);
        this.emit('error', error);
      });

      socket.on('close', () => {
        this.log('Socket closed');
        this.connected = false;
        this.emit('close');
      });

      socket.on('end', () => {
        this.log('Connection ended');
        this.connected = false;
        this.emit('close');
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
    if (!this.socket) {
      return;
    }

    this.log('Disconnecting');
    this.socket.destroy();
    this.socket = null;
    this.connected = false;
    this.buffer = '';
  }

  /**
   * Send a message to the server
   */
  send(message: string): void {
    if (!this.socket || !this.connected) {
      this.emit('error', new Error('Cannot send - not connected'));
      return;
    }

    try {
      this.log('Sending:', message);
      this.socket.write(`${message}\n`);
    } catch (error) {
      this.emit('error', error as Error);
    }
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.connected && this.socket !== null && !this.socket.destroyed;
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
   * Handle incoming data from socket
   * Implements line-delimited JSON framing
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

      this.log('Received:', line);
      this.emit('message', line);
    }
  }

  /**
   * Debug logging
   * @private
   */
  private log(...args: unknown[]): void {
    if (this.config.debug) {
      console.log('[IPCTransport]', ...args);
    }
  }
}
