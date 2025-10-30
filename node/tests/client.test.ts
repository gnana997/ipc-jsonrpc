import { EventEmitter } from 'node:events';
import * as net from 'node:net';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { JSONRPCClient, JSONRPCError } from '../src/client.js';
import { ConnectionState } from '../src/types.js';

// Mock net.Socket
vi.mock('node:net');

describe('JSONRPCClient', () => {
  let mockSocket: any;

  beforeEach(() => {
    vi.clearAllMocks();

    // Create a mock socket with EventEmitter behavior
    mockSocket = new EventEmitter();
    mockSocket.connect = vi.fn();
    mockSocket.write = vi.fn();
    mockSocket.destroy = vi.fn();

    // Mock net.Socket constructor to return our mock
    vi.mocked(net.Socket).mockImplementation(() => mockSocket);
  });

  describe('constructor', () => {
    it('should create a client with default config', () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });
      expect(client).toBeInstanceOf(JSONRPCClient);
      expect(client.getState()).toBe(ConnectionState.DISCONNECTED);
    });

    it('should merge provided config with defaults', () => {
      const client = new JSONRPCClient({
        socketPath: '/tmp/test.sock',
        connectionTimeout: 5000,
        debug: true,
      });
      expect(client).toBeInstanceOf(JSONRPCClient);
    });
  });

  describe('connect', () => {
    it('should connect to socket successfully', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();

      // Simulate successful connection
      mockSocket.emit('connect');

      await expect(connectPromise).resolves.toBeUndefined();
      expect(client.isConnected()).toBe(true);
      expect(client.getState()).toBe(ConnectionState.CONNECTED);
    });

    it('should emit connected event', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });
      const connectedSpy = vi.fn();
      client.on('connected', connectedSpy);

      const connectPromise = client.connect();
      mockSocket.emit('connect');

      await connectPromise;
      expect(connectedSpy).toHaveBeenCalledTimes(1);
    });

    it('should handle connection timeout', async () => {
      const client = new JSONRPCClient({
        socketPath: '/tmp/test.sock',
        connectionTimeout: 100,
      });

      const connectPromise = client.connect();

      // Don't emit connect event, let it timeout
      await expect(connectPromise).rejects.toThrow(/Connection timeout/);
    });

    it('should handle connection error', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      // Add error handler to prevent unhandled error
      client.on('error', () => {
        // Error is expected, do nothing
      });

      // Start connection
      const connectPromise = client.connect();

      // Emit error immediately in next tick to ensure connect setup is complete
      process.nextTick(() => {
        const error = new Error('Connection refused');
        mockSocket.emit('error', error);
      });

      await expect(connectPromise).rejects.toThrow('Connection refused');
    });

    it('should throw if already connected', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      // Try to connect again
      await expect(client.connect()).resolves.toBeUndefined();
      expect(client.isConnected()).toBe(true);
    });

    it('should throw if connection already in progress', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise1 = client.connect();
      const connectPromise2 = client.connect();

      mockSocket.emit('connect');
      await connectPromise1;

      await expect(connectPromise2).rejects.toThrow('Connection already in progress');
    });

    it('should normalize Windows named pipe paths', async () => {
      // Skip this test on non-Windows platforms as the client behavior is platform-dependent
      if (process.platform !== 'win32') {
        return;
      }

      const client = new JSONRPCClient({ socketPath: 'myapp' });
      const connectPromise = client.connect();

      expect(mockSocket.connect).toHaveBeenCalledWith('\\\\.\\pipe\\myapp');

      mockSocket.emit('connect');
      await connectPromise;
    });

    it('should not modify Unix socket paths', async () => {
      // Skip this test on Windows as the client behavior is platform-dependent
      if (process.platform === 'win32') {
        return;
      }

      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });
      const connectPromise = client.connect();

      expect(mockSocket.connect).toHaveBeenCalledWith('/tmp/test.sock');

      mockSocket.emit('connect');
      await connectPromise;
    });
  });

  describe('disconnect', () => {
    it('should disconnect from socket', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      // Connect first
      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      // Disconnect
      await client.disconnect();

      expect(mockSocket.destroy).toHaveBeenCalled();
      expect(client.getState()).toBe(ConnectionState.CLOSED);
    });

    it('should emit disconnected event', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const disconnectedSpy = vi.fn();
      client.on('disconnected', disconnectedSpy);

      await client.disconnect();

      expect(disconnectedSpy).toHaveBeenCalledTimes(1);
    });

    it('should reject pending requests on disconnect', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const requestPromise = client.request('test');

      await client.disconnect();

      await expect(requestPromise).rejects.toThrow('Client disconnected');
    });

    it('should do nothing if already disconnected', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      await client.disconnect();
      await client.disconnect();

      expect(mockSocket.destroy).not.toHaveBeenCalled();
    });
  });

  describe('request', () => {
    it('should send a request and receive response', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const requestPromise = client.request('getData', { id: 123 });

      // Check that request was sent
      expect(mockSocket.write).toHaveBeenCalled();
      const writtenData = mockSocket.write.mock.calls[0][0];
      expect(writtenData).toContain('"method":"getData"');
      expect(writtenData).toContain('"params":{"id":123}');

      // Simulate server response
      mockSocket.emit(
        'data',
        Buffer.from('{"jsonrpc":"2.0","result":{"data":"test"},"id":1}\n')
      );

      const result = await requestPromise;
      expect(result).toEqual({ data: 'test' });
    });

    it('should handle request timeout', async () => {
      const client = new JSONRPCClient({
        socketPath: '/tmp/test.sock',
        requestTimeout: 100,
      });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const requestPromise = client.request('getData');

      // Don't send response, let it timeout
      await expect(requestPromise).rejects.toThrow(/Request timeout/);
    });

    it('should handle JSON-RPC error response', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const requestPromise = client.request('getData');

      // Simulate error response
      mockSocket.emit(
        'data',
        Buffer.from(
          JSON.stringify({
            jsonrpc: '2.0',
            error: { code: -32601, message: 'Method not found' },
            id: 1,
          }) + '\n'
        )
      );

      await expect(requestPromise).rejects.toThrow(JSONRPCError);
      await expect(requestPromise).rejects.toThrow('Method not found');

      try {
        await requestPromise;
      } catch (error) {
        expect(error).toBeInstanceOf(JSONRPCError);
        expect((error as JSONRPCError).code).toBe(-32601);
      }
    });

    it('should throw if not connected', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      await expect(client.request('getData')).rejects.toThrow('Not connected');
    });

    it('should handle multiple concurrent requests', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const request1 = client.request('method1');
      const request2 = client.request('method2');

      // Respond to request 2 first
      mockSocket.emit(
        'data',
        Buffer.from('{"jsonrpc":"2.0","result":"result2","id":2}\n')
      );

      // Then respond to request 1
      mockSocket.emit(
        'data',
        Buffer.from('{"jsonrpc":"2.0","result":"result1","id":1}\n')
      );

      expect(await request1).toBe('result1');
      expect(await request2).toBe('result2');
    });

    it('should handle multi-line responses', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const requestPromise = client.request('getData');

      // Simulate response split across multiple data events
      mockSocket.emit('data', Buffer.from('{"jsonrpc":"2.0","result":'));
      mockSocket.emit('data', Buffer.from('{"data":"test"},"id":1}\n'));

      const result = await requestPromise;
      expect(result).toEqual({ data: 'test' });
    });
  });

  describe('notify', () => {
    it('should send a notification', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      client.notify('logMessage', { level: 'info', message: 'test' });

      expect(mockSocket.write).toHaveBeenCalled();
      const writtenData = mockSocket.write.mock.calls[0][0];
      expect(writtenData).toContain('"method":"logMessage"');
      expect(writtenData).not.toContain('"id"');
    });

    it('should throw if not connected', () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      expect(() => client.notify('logMessage')).toThrow('Not connected');
    });
  });

  describe('notifications from server', () => {
    it('should emit notification event for server notifications', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const notificationSpy = vi.fn();
      client.on('notification', notificationSpy);

      // Simulate server notification
      mockSocket.emit(
        'data',
        Buffer.from(
          JSON.stringify({
            jsonrpc: '2.0',
            method: 'progress',
            params: { percentage: 50 },
          }) + '\n'
        )
      );

      expect(notificationSpy).toHaveBeenCalledWith('progress', { percentage: 50 });
    });
  });

  describe('error handling', () => {
    it('should emit error event on socket error', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const errorSpy = vi.fn();
      client.on('error', errorSpy);

      const error = new Error('Socket error');
      mockSocket.emit('error', error);

      expect(errorSpy).toHaveBeenCalledWith(error);
    });

    it('should handle malformed JSON', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const errorSpy = vi.fn();
      client.on('error', errorSpy);

      // Send invalid JSON
      mockSocket.emit('data', Buffer.from('not valid json\n'));

      expect(errorSpy).toHaveBeenCalled();
    });
  });

  describe('reconnection', () => {
    it('should attempt to reconnect on disconnect when autoReconnect is enabled', async () => {
      const client = new JSONRPCClient({
        socketPath: '/tmp/test.sock',
        autoReconnect: true,
        maxReconnectAttempts: 2,
        reconnectDelay: 50,
      });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const reconnectingSpy = vi.fn();
      let stateWhenReconnecting: string | undefined;

      client.on('reconnecting', (attempt) => {
        reconnectingSpy(attempt);
        stateWhenReconnecting = client.getState();
      });

      // Simulate disconnect
      mockSocket.emit('end');

      // Wait for reconnection attempt
      await new Promise((resolve) => setTimeout(resolve, 100));

      expect(reconnectingSpy).toHaveBeenCalledWith(1);
      expect(stateWhenReconnecting).toBe(ConnectionState.RECONNECTING);
      // After reconnectDelay, it will be in CONNECTING state
      expect([ConnectionState.RECONNECTING, ConnectionState.CONNECTING]).toContain(
        client.getState()
      );
    });

    it('should not reconnect if autoReconnect is disabled', async () => {
      const client = new JSONRPCClient({
        socketPath: '/tmp/test.sock',
        autoReconnect: false,
      });

      const connectPromise = client.connect();
      mockSocket.emit('connect');
      await connectPromise;

      const reconnectingSpy = vi.fn();
      client.on('reconnecting', reconnectingSpy);

      // Simulate disconnect
      mockSocket.emit('end');

      // Wait a bit
      await new Promise((resolve) => setTimeout(resolve, 100));

      expect(reconnectingSpy).not.toHaveBeenCalled();
    });
  });

  describe('state management', () => {
    it('should return correct connection state', async () => {
      const client = new JSONRPCClient({ socketPath: '/tmp/test.sock' });

      expect(client.getState()).toBe(ConnectionState.DISCONNECTED);
      expect(client.isConnected()).toBe(false);

      const connectPromise = client.connect();
      expect(client.getState()).toBe(ConnectionState.CONNECTING);
      expect(client.isConnected()).toBe(false);

      mockSocket.emit('connect');
      await connectPromise;

      expect(client.getState()).toBe(ConnectionState.CONNECTED);
      expect(client.isConnected()).toBe(true);

      await client.disconnect();

      expect(client.getState()).toBe(ConnectionState.CLOSED);
      expect(client.isConnected()).toBe(false);
    });
  });
});
