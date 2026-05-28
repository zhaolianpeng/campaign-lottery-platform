export interface ApiEnvelope<T> {
  readonly code: string;
  readonly message: string;
  readonly data: T;
}

export const AUTH_EXPIRED_EVENT = 'lottery:auth-expired';

export class ApiRequestError extends Error {
  public readonly code: string;
  public readonly status: number;

  public constructor(code: string, message: string, status: number) {
    super(message);
    this.name = 'ApiRequestError';
    this.code = code;
    this.status = status;
  }
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? '';

function apiUrl(path: string): string {
  if (path.startsWith('http://') || path.startsWith('https://')) {
    return path;
  }
  return `${API_BASE_URL}${path.startsWith('/') ? path : `/${path}`}`;
}

export function apiAssetUrl(path: string): string {
  return apiUrl(path);
}

function normalizeAuthExpiredMessage(message: string | undefined): string {
  const trimmed = message?.trim();
  if (!trimmed || trimmed.toLowerCase() === 'unauthorized') {
    return '登录状态已失效，请重新登录后继续。';
  }
  return trimmed;
}

export async function apiRequest<T>(
  path: string,
  token: string,
  init: RequestInit = {},
): Promise<T> {
  const headers = new Headers(init.headers);
  const isFormData = typeof FormData !== 'undefined' && init.body instanceof FormData;
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  if (!isFormData && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(apiUrl(path), {
    ...init,
    headers,
  });
  const payload = (await response.json()) as ApiEnvelope<T>;
  if (!response.ok || payload.code !== 'ok') {
    if (response.status === 401 && token && typeof window !== 'undefined') {
      window.dispatchEvent(
        new CustomEvent(AUTH_EXPIRED_EVENT, {
          detail: {
            path,
            code: payload.code || 'request_failed',
            message: normalizeAuthExpiredMessage(payload.message),
          },
        }),
      );
    }
    throw new ApiRequestError(payload.code || 'request_failed', payload.message || '请求失败', response.status);
  }
  return payload.data;
}

export function apiPostRequest<T>(path: string, token: string, init: RequestInit = {}): Promise<T> {
  return apiRequest<T>(path, token, {
    ...init,
    method: init.method ?? 'POST',
  });
}
