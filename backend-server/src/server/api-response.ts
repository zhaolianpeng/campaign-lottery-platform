import { randomUUID } from 'node:crypto';
import { NextResponse } from 'next/server';
import { getAppConfig } from './config';
import { AppError } from './errors';

export interface ApiEnvelope<T> {
  readonly code: string;
  readonly message: string;
  readonly data: T;
  readonly error_id?: string;
}

export function corsHeaders(): HeadersInit {
  return {
    'Access-Control-Allow-Origin': getAppConfig().server.corsAllowOrigin,
    'Access-Control-Allow-Methods': 'GET,POST,PUT,PATCH,DELETE,OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization, X-Anonymous-Draw-Token, X-Request-Id',
  };
}

export function ok<T>(message: string, data: T, status = 200): NextResponse<ApiEnvelope<T>> {
  return NextResponse.json(
    { code: 'ok', message, data },
    { status, headers: corsHeaders() },
  );
}

export function fail(error: unknown): NextResponse<ApiEnvelope<null>> {
  if (error instanceof AppError) {
    return NextResponse.json(
      { code: error.code, message: error.message, data: null },
      { status: error.status, headers: corsHeaders() },
    );
  }

  const errorId = randomUUID();
  console.error(`[internal_error:${errorId}]`, error);
  return NextResponse.json(
    {
      code: 'internal_error',
      message: '服务器内部错误，请稍后重试',
      data: null,
      error_id: errorId,
    },
    { status: 500, headers: corsHeaders() },
  );
}

export function bearerToken(request: Request): string {
  const auth = request.headers.get('authorization') ?? '';
  if (!auth.toLowerCase().startsWith('bearer ')) {
    return '';
  }
  return auth.slice(7).trim();
}

export function anonymousDrawToken(request: Request): string {
  return request.headers.get('x-anonymous-draw-token')?.trim() ?? '';
}

export function requestIdFromRequest(request: Request): string {
  return request.headers.get('x-request-id')?.trim() || randomUUID();
}
