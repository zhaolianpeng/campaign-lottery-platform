import { NextResponse } from 'next/server';
import { AppError } from './errors';

export interface ApiEnvelope<T> {
  readonly code: string;
  readonly message: string;
  readonly data: T;
}

export function corsHeaders(): HeadersInit {
  return {
    'Access-Control-Allow-Origin': process.env.CORS_ALLOW_ORIGIN ?? '*',
    'Access-Control-Allow-Methods': 'GET,POST,PUT,PATCH,DELETE,OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization',
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

  const message = error instanceof Error ? error.message : 'internal server error';
  return NextResponse.json(
    { code: 'internal_error', message, data: null },
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
