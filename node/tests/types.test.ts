import { describe, expect, it } from 'vitest';
import type {
  ClientConfig,
  JSONRPCError,
  JSONRPCErrorResponse,
  JSONRPCMessage,
  JSONRPCNotification,
  JSONRPCRequest,
  JSONRPCResponse,
} from '../src/types.js';
import { ConnectionState } from '../src/types.js';

describe('Types', () => {
  describe('ConnectionState enum', () => {
    it('should have all required states', () => {
      expect(ConnectionState.DISCONNECTED).toBe('disconnected');
      expect(ConnectionState.CONNECTING).toBe('connecting');
      expect(ConnectionState.CONNECTED).toBe('connected');
      expect(ConnectionState.RECONNECTING).toBe('reconnecting');
      expect(ConnectionState.CLOSED).toBe('closed');
    });
  });

  describe('JSONRPCRequest', () => {
    it('should accept valid request object', () => {
      const request: JSONRPCRequest = {
        jsonrpc: '2.0',
        method: 'getData',
        params: { id: 123 },
        id: 1,
      };

      expect(request.method).toBe('getData');
      expect(request.id).toBe(1);
    });

    it('should allow string or number id', () => {
      const request1: JSONRPCRequest = {
        method: 'test',
        id: 1,
      };

      const request2: JSONRPCRequest = {
        method: 'test',
        id: 'abc-123',
      };

      expect(request1.id).toBe(1);
      expect(request2.id).toBe('abc-123');
    });

    it('should allow omitting jsonrpc field', () => {
      const request: JSONRPCRequest = {
        method: 'test',
        id: 1,
      };

      expect(request.jsonrpc).toBeUndefined();
    });
  });

  describe('JSONRPCResponse', () => {
    it('should accept valid success response', () => {
      const response: JSONRPCResponse = {
        jsonrpc: '2.0',
        result: { data: 'test' },
        id: 1,
      };

      expect(response.result).toEqual({ data: 'test' });
      expect(response.id).toBe(1);
    });

    it('should accept null result', () => {
      const response: JSONRPCResponse = {
        result: null,
        id: 1,
      };

      expect(response.result).toBeNull();
    });
  });

  describe('JSONRPCErrorResponse', () => {
    it('should accept valid error response', () => {
      const response: JSONRPCErrorResponse = {
        jsonrpc: '2.0',
        error: {
          code: -32601,
          message: 'Method not found',
        },
        id: 1,
      };

      expect(response.error.code).toBe(-32601);
      expect(response.error.message).toBe('Method not found');
    });

    it('should accept error with data field', () => {
      const response: JSONRPCErrorResponse = {
        error: {
          code: -32602,
          message: 'Invalid params',
          data: { field: 'email' },
        },
        id: 1,
      };

      expect(response.error.data).toEqual({ field: 'email' });
    });
  });

  describe('JSONRPCNotification', () => {
    it('should accept valid notification', () => {
      const notification: JSONRPCNotification = {
        jsonrpc: '2.0',
        method: 'progress',
        params: { percentage: 50 },
      };

      expect(notification.method).toBe('progress');
      expect(notification.params).toEqual({ percentage: 50 });
    });

    it('should not have id field', () => {
      const notification: JSONRPCNotification = {
        method: 'event',
      };

      expect('id' in notification).toBe(false);
    });
  });

  describe('JSONRPCMessage union', () => {
    it('should accept any valid message type', () => {
      const messages: JSONRPCMessage[] = [
        { method: 'test', id: 1 } as JSONRPCRequest,
        { result: 'ok', id: 1 } as JSONRPCResponse,
        {
          error: { code: -32601, message: 'Not found' },
          id: 1,
        } as JSONRPCErrorResponse,
        { method: 'notification' } as JSONRPCNotification,
      ];

      expect(messages).toHaveLength(4);
    });
  });

  describe('ClientConfig', () => {
    it('should require socketPath', () => {
      const config: ClientConfig = {
        socketPath: '/tmp/test.sock',
      };

      expect(config.socketPath).toBe('/tmp/test.sock');
    });

    it('should accept all optional fields', () => {
      const config: ClientConfig = {
        socketPath: '/tmp/test.sock',
        connectionTimeout: 5000,
        requestTimeout: 10000,
        debug: true,
        autoReconnect: true,
        maxReconnectAttempts: 5,
        reconnectDelay: 2000,
      };

      expect(config.debug).toBe(true);
      expect(config.autoReconnect).toBe(true);
    });

    it('should accept minimal config', () => {
      const config: ClientConfig = {
        socketPath: '/tmp/test.sock',
      };

      expect(config.connectionTimeout).toBeUndefined();
      expect(config.debug).toBeUndefined();
    });
  });

  describe('JSONRPCError interface', () => {
    it('should define required fields', () => {
      const error: JSONRPCError = {
        code: -32601,
        message: 'Method not found',
      };

      expect(error.code).toBe(-32601);
      expect(error.message).toBe('Method not found');
    });

    it('should accept optional data field', () => {
      const error: JSONRPCError = {
        code: -32602,
        message: 'Invalid params',
        data: { details: 'missing field' },
      };

      expect(error.data).toEqual({ details: 'missing field' });
    });
  });

  describe('Type compatibility', () => {
    it('should ensure request and response share id type', () => {
      const request: JSONRPCRequest = {
        method: 'test',
        id: 'request-123',
      };

      const response: JSONRPCResponse = {
        result: 'ok',
        id: request.id, // Should be type compatible
      };

      expect(response.id).toBe('request-123');
    });

    it('should distinguish between response types by discriminator fields', () => {
      const successResponse: JSONRPCResponse = {
        result: 'success',
        id: 1,
      };

      const errorResponse: JSONRPCErrorResponse = {
        error: { code: -1, message: 'error' },
        id: 1,
      };

      expect('result' in successResponse).toBe(true);
      expect('error' in successResponse).toBe(false);
      expect('result' in errorResponse).toBe(false);
      expect('error' in errorResponse).toBe(true);
    });
  });
});
