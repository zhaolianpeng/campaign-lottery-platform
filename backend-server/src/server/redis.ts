import { createClient } from 'redis';
import { getAppConfig, type AppConfig } from './config';
import type { DependencyHealth } from './database';

type RedisClient = ReturnType<typeof createClient>;

const globalForRedis = globalThis as typeof globalThis & {
  __campaignLotteryRedisClient?: RedisClient;
};

export function getRedisClient(): RedisClient | null {
  const config = getAppConfig().redis;
  if (!config.enabled) {
    return null;
  }

  if (!globalForRedis.__campaignLotteryRedisClient) {
    const client = createClient(redisClientOptions(config));
    client.on('error', () => undefined);
    globalForRedis.__campaignLotteryRedisClient = client;
  }

  return globalForRedis.__campaignLotteryRedisClient;
}

export async function pingRedis(): Promise<DependencyHealth> {
  const startedAt = Date.now();
  const client = getRedisClient();
  if (!client) {
    return { status: 'disabled' };
  }

  try {
    if (!client.isOpen) {
      await client.connect();
    }
    await client.ping();
    return { status: 'ok', latency_ms: Date.now() - startedAt };
  } catch (error) {
    return {
      status: 'error',
      latency_ms: Date.now() - startedAt,
      message: error instanceof Error ? error.message : 'Redis connection failed',
    };
  }
}

export function redisKey(key: string): string {
  return `${getAppConfig().redis.keyPrefix}${key}`;
}

function redisClientOptions(config: AppConfig['redis']): Parameters<typeof createClient>[0] {
  if (config.url) {
    return { url: config.url };
  }

  return {
    socket: {
      host: config.host,
      port: config.port,
    },
    password: config.password,
    database: config.database,
  };
}
