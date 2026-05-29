import { USER_TOKEN_KEY } from '@/features/lottery/constants';

export { USER_TOKEN_KEY };

export function getUserToken(): string {
  if (typeof window === 'undefined') {
    return '';
  }
  return window.localStorage.getItem(USER_TOKEN_KEY) ?? '';
}

/** @deprecated Use getUserToken */
export const readUserAuthToken = getUserToken;

export function setUserToken(token: string): void {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(USER_TOKEN_KEY, token);
}

export function clearUserToken(): void {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.removeItem(USER_TOKEN_KEY);
}
