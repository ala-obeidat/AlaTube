import { describe, expect, it } from 'vitest';
import { readableBytes } from './api';

describe('readableBytes', () => {
  it('formats byte counts for display', () => {
    expect(readableBytes(1024 * 1024)).toBe('1.0 MB');
    expect(readableBytes(undefined)).toBe('Unknown size');
  });
});
