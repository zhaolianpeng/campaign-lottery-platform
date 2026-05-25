import { LotteryService } from './lottery-service';
import { createMemoryStore, type MemoryStore } from './memory-store';

interface GlobalServices {
  readonly store: MemoryStore;
  readonly service: LotteryService;
}

const globalForServices = globalThis as typeof globalThis & {
  __campaignLotteryServices?: GlobalServices;
};

export function getService(): LotteryService {
  if (!globalForServices.__campaignLotteryServices) {
    const store = createMemoryStore();
    globalForServices.__campaignLotteryServices = {
      store,
      service: new LotteryService(store),
    };
  }
  return globalForServices.__campaignLotteryServices.service;
}
