import { describe, expect, it } from 'vitest';
import { clearUserToken, getUserToken, setUserToken } from './auth-token';

describe('auth-token', () => {
  it('reads and writes user token from localStorage', () => {
    const storage = new Map<string, string>();
    const localStorageMock = {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => {
        storage.set(key, value);
      },
      removeItem: (key: string) => {
        storage.delete(key);
      },
    };
    Object.defineProperty(globalThis, 'window', { value: { localStorage: localStorageMock }, configurable: true });

    setUserToken('abc');
    expect(getUserToken()).toBe('abc');
    clearUserToken();
    expect(getUserToken()).toBe('');
  });
});
