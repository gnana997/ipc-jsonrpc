import { describe, expect, it } from 'vitest';
import { JSONRPCError } from '../src/client.js';

describe('JSONRPCError', () => {
  it('should create an error with code and message', () => {
    const error = new JSONRPCError({
      code: -32601,
      message: 'Method not found',
    });

    expect(error).toBeInstanceOf(Error);
    expect(error).toBeInstanceOf(JSONRPCError);
    expect(error.code).toBe(-32601);
    expect(error.message).toBe('Method not found');
    expect(error.name).toBe('JSONRPCError');
  });

  it('should include optional data field', () => {
    const error = new JSONRPCError({
      code: -32602,
      message: 'Invalid params',
      data: { expected: 'string', received: 'number' },
    });

    expect(error.code).toBe(-32602);
    expect(error.message).toBe('Invalid params');
    expect(error.data).toEqual({ expected: 'string', received: 'number' });
  });

  it('should maintain proper stack trace', () => {
    const error = new JSONRPCError({
      code: -32603,
      message: 'Internal error',
    });

    expect(error.stack).toBeDefined();
    expect(error.stack).toContain('JSONRPCError');
  });

  it('should handle standard JSON-RPC error codes', () => {
    const testCases = [
      { code: -32700, message: 'Parse error' },
      { code: -32600, message: 'Invalid Request' },
      { code: -32601, message: 'Method not found' },
      { code: -32602, message: 'Invalid params' },
      { code: -32603, message: 'Internal error' },
    ];

    for (const testCase of testCases) {
      const error = new JSONRPCError(testCase);
      expect(error.code).toBe(testCase.code);
      expect(error.message).toBe(testCase.message);
    }
  });

  it('should handle custom error codes', () => {
    const error = new JSONRPCError({
      code: 1001,
      message: 'Custom application error',
      data: { details: 'Some specific error details' },
    });

    expect(error.code).toBe(1001);
    expect(error.message).toBe('Custom application error');
    expect(error.data).toEqual({ details: 'Some specific error details' });
  });

  it('should be catchable and type-checkable', () => {
    try {
      throw new JSONRPCError({
        code: -32601,
        message: 'Method not found',
      });
    } catch (error) {
      expect(error).toBeInstanceOf(JSONRPCError);
      if (error instanceof JSONRPCError) {
        expect(error.code).toBe(-32601);
        expect(error.message).toBe('Method not found');
      }
    }
  });

  it('should serialize to JSON properly', () => {
    const error = new JSONRPCError({
      code: -32602,
      message: 'Invalid params',
      data: { field: 'email', reason: 'invalid format' },
    });

    const serialized = JSON.stringify({
      code: error.code,
      message: error.message,
      data: error.data,
    });

    const parsed = JSON.parse(serialized);
    expect(parsed.code).toBe(-32602);
    expect(parsed.message).toBe('Invalid params');
    expect(parsed.data).toEqual({ field: 'email', reason: 'invalid format' });
  });
});
