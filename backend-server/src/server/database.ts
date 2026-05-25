import mysql, { type Pool, type PoolOptions } from 'mysql2/promise';
import { getAppConfig, type AppConfig } from './config';

export interface DependencyHealth {
  readonly status: 'ok' | 'disabled' | 'error';
  readonly latency_ms?: number;
  readonly message?: string;
}

const globalForMysql = globalThis as typeof globalThis & {
  __campaignLotteryMysqlPool?: Pool;
};

export function getMysqlPool(): Pool | null {
  const config = getAppConfig().mysql;
  if (!config.enabled) {
    return null;
  }

  if (!globalForMysql.__campaignLotteryMysqlPool) {
    globalForMysql.__campaignLotteryMysqlPool = mysql.createPool(mysqlPoolOptions(config));
  }

  return globalForMysql.__campaignLotteryMysqlPool;
}

export async function pingMysql(): Promise<DependencyHealth> {
  const startedAt = Date.now();
  const pool = getMysqlPool();
  if (!pool) {
    return { status: 'disabled' };
  }

  try {
    await pool.query('SELECT 1 AS ok');
    return { status: 'ok', latency_ms: Date.now() - startedAt };
  } catch (error) {
    return {
      status: 'error',
      latency_ms: Date.now() - startedAt,
      message: error instanceof Error ? error.message : 'MySQL connection failed',
    };
  }
}

function mysqlPoolOptions(config: AppConfig['mysql']): PoolOptions {
  return {
    ...mysqlDsnOptions(config),
    waitForConnections: true,
    connectionLimit: config.connectionLimit,
  };
}

function mysqlDsnOptions(config: AppConfig['mysql']): PoolOptions {
  if (!config.dsn) {
    return {
      host: config.host,
      port: config.port,
      database: config.database,
      user: config.user,
      password: config.password,
      charset: config.charset,
    };
  }

  if (config.dsn.startsWith('mysql://') || config.dsn.startsWith('mysql2://')) {
    const url = new URL(config.dsn.replace(/^mysql2:\/\//, 'mysql://'));
    return {
      host: url.hostname,
      port: Number(url.port || 3306),
      database: decodeURIComponent(url.pathname.replace(/^\//, '')),
      user: decodeURIComponent(url.username),
      password: decodeURIComponent(url.password),
      charset: url.searchParams.get('charset') ?? config.charset,
    };
  }

  const match = config.dsn.match(/^([^:]+):(.*)@tcp\(([^):]+)(?::(\d+))?\)\/([^?]+)(?:\?(.*))?$/);
  if (!match) {
    throw new Error('Unsupported MYSQL_DSN format');
  }

  const [, user, password, host, port, database, query = ''] = match;
  const params = new URLSearchParams(query);
  return {
    host,
    port: Number(port || 3306),
    database,
    user,
    password,
    charset: params.get('charset') ?? config.charset,
  };
}
