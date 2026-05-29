import { randomUUID } from 'node:crypto';

export interface RequestLogContext {
  readonly requestId: string;
  readonly method: string;
  readonly path: string;
  readonly ip: string;
}

export function buildRequestLogContext(request: Request, path: string): RequestLogContext {
  return {
    requestId: request.headers.get('x-request-id')?.trim() || randomUUID(),
    method: request.method,
    path,
    ip: request.headers.get('x-forwarded-for')?.split(',')[0]?.trim() ?? request.headers.get('x-real-ip') ?? '127.0.0.1',
  };
}

export function logStructured(event: string, context: RequestLogContext, extra?: Record<string, unknown>): void {
  console.info(JSON.stringify({ event, ...context, ...extra, timestamp: new Date().toISOString() }));
}
