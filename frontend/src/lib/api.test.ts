import { describe, expect, it } from 'vitest';
import { readableBytes, withToken } from './api';

describe('readableBytes', () => {
  it('formats byte counts for display', () => {
    expect(readableBytes(1024 * 1024)).toBe('1.0 MB');
    expect(readableBytes(undefined)).toBe('Unknown size');
  });
});

describe('withToken', () => {
  it('returns the url unchanged when PUBLIC_API_TOKEN is unset', () => {
    expect(withToken('/api/jobs/abc/events')).toBe('/api/jobs/abc/events');
  });
});
