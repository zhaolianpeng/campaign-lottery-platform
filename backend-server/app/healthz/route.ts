import { NextResponse } from 'next/server';
import { corsHeaders } from '@/server/api-response';
import { getAppConfig } from '@/server/config';
import { checkInfrastructure, isInfrastructureHealthy } from '@/server/infrastructure';
import { getLatestSchemaVersion } from '@/server/schema-version';

export const runtime = 'nodejs';

export async function GET(): Promise<Response> {
  const dependencies = await checkInfrastructure();
  const healthy = isInfrastructureHealthy(dependencies);
  const config = getAppConfig();
  const schemaVersion = await getLatestSchemaVersion();

  return NextResponse.json(
    {
      service: 'campaign-lottery-backend-server',
      status: healthy ? 'ok' : 'degraded',
      storage_mode: config.mysql.enabled ? 'mysql' : 'memory',
      schema_version: schemaVersion,
      dependencies,
      timestamp: new Date().toISOString(),
    },
    { status: healthy ? 200 : 503, headers: corsHeaders() },
  );
}
