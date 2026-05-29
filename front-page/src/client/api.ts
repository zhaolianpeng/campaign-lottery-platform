export interface ApiEnvelope<T> {
  readonly code: string;
  readonly message: string;
  readonly data: T;
  readonly error_id?: string;
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

async function parseApiEnvelope<T>(response: Response): Promise<ApiEnvelope<T>> {
  const contentType = response.headers.get('content-type') ?? '';
  const raw = await response.text();
  if (!raw.trim()) {
    throw new ApiRequestError('invalid_response', '服务器返回空响应', response.status);
  }
  if (!contentType.includes('application/json')) {
    throw new ApiRequestError('invalid_response', '服务器返回非 JSON 响应', response.status);
  }
  try {
    return JSON.parse(raw) as ApiEnvelope<T>;
  } catch {
    throw new ApiRequestError('invalid_response', '无法解析服务器响应', response.status);
  }
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

  let response: Response;
  try {
    response = await fetch(apiUrl(path), {
      ...init,
      headers,
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : '网络请求失败';
    throw new ApiRequestError('network_error', message, 0);
  }

  const payload = await parseApiEnvelope<T>(response);
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
