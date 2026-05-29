'use client';

import { getUserToken, setUserToken, clearUserToken } from '@/client/auth-token';

export function useAuth() {
  return {
    getToken: getUserToken,
    setToken: setUserToken,
    clearToken: clearUserToken,
  };
}
