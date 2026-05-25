import { NextResponse } from 'next/server';
import { corsHeaders } from '@/server/api-response';
import { checkInfrastructure, isInfrastructureHealthy } from '@/server/infrastructure';

export const runtime = 'nodejs';

export async function GET(): Promise<Response> {
  const dependencies = await checkInfrastructure();
  const healthy = isInfrastructureHealthy(dependencies);

  return NextResponse.json(
    {
      service: 'campaign-lottery-backend-server',
      status: healthy ? 'ok' : 'degraded',
      dependencies,
      timestamp: new Date().toISOString(),
    },
    { status: healthy ? 200 : 503, headers: corsHeaders() },
  );
}
