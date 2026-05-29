import { getAppConfig } from '../config';
import type { MemoryStore } from '../memory-store';
import { createMemoryRepositories } from './memory';
import { createMysqlRepositories } from './mysql';
import type { RepositoryBundle } from './types';

const globalForRepos = globalThis as typeof globalThis & {
  __campaignLotteryRepos?: RepositoryBundle;
};

export function getRepositories(store?: MemoryStore): RepositoryBundle {
  if (globalForRepos.__campaignLotteryRepos) {
    return globalForRepos.__campaignLotteryRepos;
  }

  const mode = getAppConfig().server.storageMode;
  const bundle = mode === 'mysql' ? createMysqlRepositories() : createMemoryRepositories(store!);
  globalForRepos.__campaignLotteryRepos = bundle;
  return bundle;
}

export function resetRepositoriesForTests(): void {
  globalForRepos.__campaignLotteryRepos = undefined;
}

export type { RepositoryBundle } from './types';
