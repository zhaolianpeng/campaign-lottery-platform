import type { RowDataPacket } from 'mysql2/promise';
import { getMysqlPool } from './database';

const FALLBACK_SCHEMA_VERSION = '000_legacy_schema';

interface MigrationRow extends RowDataPacket {
  name: string;
}

let cachedVersion: string | null = null;

export async function getLatestSchemaVersion(): Promise<string> {
  if (cachedVersion) {
    return cachedVersion;
  }

  const pool = getMysqlPool();
  if (!pool) {
    cachedVersion = FALLBACK_SCHEMA_VERSION;
    return cachedVersion;
  }

  try {
    const [rows] = await pool.query<MigrationRow[]>(
      'SELECT name FROM _schema_migrations ORDER BY name DESC LIMIT 1',
    );
    cachedVersion = rows[0]?.name ?? FALLBACK_SCHEMA_VERSION;
  } catch {
    cachedVersion = FALLBACK_SCHEMA_VERSION;
  }

  return cachedVersion;
}

export function resetSchemaVersionCacheForTests(): void {
  cachedVersion = null;
}
