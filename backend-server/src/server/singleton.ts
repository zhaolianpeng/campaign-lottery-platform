import { LotteryService } from './lottery-service';
import { createMemoryStore, type MemoryStore } from './memory-store';

interface GlobalServices {
  readonly store: MemoryStore;
  readonly service: LotteryService;
}

const globalForServices = globalThis as typeof globalThis & {
  __campaignLotteryServices?: GlobalServices;
  __campaignLotteryServicesPromise?: Promise<GlobalServices>;
};

export async function getService(): Promise<LotteryService> {
  if (globalForServices.__campaignLotteryServices) {
    return globalForServices.__campaignLotteryServices.service;
  }

  if (!globalForServices.__campaignLotteryServicesPromise) {
    globalForServices.__campaignLotteryServicesPromise = createMemoryStore().then((store) => {
      const services = {
        store,
        service: new LotteryService(store),
      };
      globalForServices.__campaignLotteryServices = services;
      return services;
    });
  }

  const services = await globalForServices.__campaignLotteryServicesPromise;
  return services.service;
}
