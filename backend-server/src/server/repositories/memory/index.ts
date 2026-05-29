import type { MemoryStore } from '../../memory-store';
import type { DrawRecord } from '../../types';
import type { AdminAuditEntry, DrawRepo, LedgerRepo, PaymentOrderRepo, PaymentOrderRow, PityRepo, RepositoryBundle, SessionRepo, UserRepo } from '../types';

function noopUserRepo(): UserRepo {
  return {
    findById: async () => null,
    findByMobile: async () => null,
    create: async () => undefined,
    updateStatus: async () => undefined,
  };
}

function noopSessionRepo(): SessionRepo {
  return {
    create: async () => undefined,
    findUserIdByToken: async () => null,
    deleteToken: async () => undefined,
  };
}

function memoryDrawRepo(store: MemoryStore): DrawRepo {
  const requestIndex = new Map<string, DrawRecord>();
  return {
    async findByRequestId(requestId) {
      return requestIndex.get(requestId) ?? null;
    },
    async insertDraw(record, requestId) {
      if (requestId) {
        requestIndex.set(requestId, record);
      }
    },
  };
}

function memoryLedgerRepo(store: MemoryStore): LedgerRepo {
  return {
    async getPoints(userId) {
      try {
        return store.getUserMember(userId).points;
      } catch {
        return 0;
      }
    },
    async addPoints(userId, delta, reason, remark) {
      return store.grantMemberPoints(userId, delta, reason, remark).new_points;
    },
  };
}

function memoryPityRepo(): PityRepo {
  const state = new Map<string, { soft: number; hard: number; upGuarantee: boolean }>();
  const key = (userId: string, campaignId: string) => `${userId}:${campaignId}`;
  return {
    async getState(userId, campaignId) {
      return state.get(key(userId, campaignId)) ?? { soft: 0, hard: 0, upGuarantee: false };
    },
    async saveState(userId, campaignId, next) {
      state.set(key(userId, campaignId), next);
    },
  };
}

function memoryPaymentOrderRepo(): PaymentOrderRepo {
  const orders = new Map<string, PaymentOrderRow & { subject: string; presentation: string }>();
  return {
    async upsert(order) {
      orders.set(order.orderNo, order);
    },
    async findByOrderNo(orderNo) {
      return orders.get(orderNo) ?? null;
    },
    async updateStatus(orderNo, status) {
      const existing = orders.get(orderNo);
      if (existing) {
        orders.set(orderNo, { ...existing, status });
      }
    },
    async listByUser(userId, limit = 50) {
      return [...orders.values()].filter((o) => o.userId === userId).slice(0, limit);
    },
  };
}

function memoryAdminAuditRepo(): import('../types').AdminAuditRepo {
  const entries: AdminAuditEntry[] = [];
  return {
    async insert(entry) {
      entries.unshift(entry);
    },
    async listRecent(limit = 100) {
      return entries.slice(0, limit);
    },
  };
}

export function createMemoryRepositories(store: MemoryStore): RepositoryBundle {
  return {
    users: noopUserRepo(),
    sessions: noopSessionRepo(),
    draws: memoryDrawRepo(store),
    ledger: memoryLedgerRepo(store),
    pity: memoryPityRepo(),
    paymentOrders: memoryPaymentOrderRepo(),
    adminAudit: memoryAdminAuditRepo(),
  };
}
