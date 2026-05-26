import { pingMysql, type DependencyHealth } from './database';
import { pingRedis } from './redis';

export interface InfrastructureHealth {
  readonly mysql: DependencyHealth;
  readonly redis: DependencyHealth;
}

export async function checkInfrastructure(): Promise<InfrastructureHealth> {
  const [mysql, redis] = await Promise.all([pingMysql(), pingRedis()]);
  return { mysql, redis };
}

export function isInfrastructureHealthy(health: InfrastructureHealth): boolean {
  return Object.values(health).every((item) => item.status !== 'error');
}
