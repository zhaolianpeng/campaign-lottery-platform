import { NextResponse } from 'next/server';
import { corsHeaders } from '@/server/api-response';

export function GET(): Response {
  return NextResponse.json(
    {
      service: 'campaign-lottery-backend-server',
      status: 'ok',
      timestamp: new Date().toISOString(),
    },
    { headers: corsHeaders() },
  );
}
