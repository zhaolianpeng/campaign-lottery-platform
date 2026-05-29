import type { DrawRecord, User, UserMember } from '../types';

export interface UserRepo {
  findById(userId: string): Promise<User | null>;
  findByMobile(mobile: string): Promise<User | null>;
  create(user: User): Promise<void>;
  updateStatus(userId: string, status: User['status']): Promise<void>;
}

export interface SessionRepo {
  create(token: string, userId: string, expiresAt: Date): Promise<void>;
  findUserIdByToken(token: string): Promise<string | null>;
  deleteToken(token: string): Promise<void>;
}

export interface DrawRepo {
  findByRequestId(requestId: string): Promise<DrawRecord | null>;
  insertDraw(record: DrawRecord, requestId: string): Promise<void>;
}

export interface LedgerRepo {
  addPoints(userId: string, delta: number, reason: string, remark: string): Promise<number>;
  getPoints(userId: string): Promise<number>;
}

export interface PityRepo {
  getState(userId: string, campaignId: string): Promise<{ soft: number; hard: number; upGuarantee: boolean }>;
  saveState(userId: string, campaignId: string, state: { soft: number; hard: number; upGuarantee: boolean }): Promise<void>;
}

export interface PaymentOrderRow {
  readonly orderNo: string;
  readonly userId: string;
  readonly clientRequestId: string;
  readonly channel: string;
  readonly businessType: string;
  readonly businessId: string;
  readonly amountCents: number;
  readonly status: string;
}

export interface PaymentOrderRepo {
  upsert(order: PaymentOrderRow & { readonly subject: string; readonly presentation: string }): Promise<void>;
  findByOrderNo(orderNo: string): Promise<PaymentOrderRow | null>;
  updateStatus(orderNo: string, status: string, patch?: { readonly channelTradeNo?: string; readonly paidAt?: Date }): Promise<void>;
  listByUser(userId: string, limit?: number): Promise<readonly PaymentOrderRow[]>;
}

export interface AdminAuditEntry {
  readonly adminUserId?: number;
  readonly adminUsername: string;
  readonly action: string;
  readonly resourceType: string;
  readonly resourceId: string;
  readonly requestId: string;
  readonly payload?: Record<string, unknown>;
  readonly ipAddress: string;
}

export interface AdminAuditRepo {
  insert(entry: AdminAuditEntry): Promise<void>;
  listRecent(limit?: number): Promise<readonly AdminAuditEntry[]>;
}

export interface RepositoryBundle {
  readonly users: UserRepo;
  readonly sessions: SessionRepo;
  readonly draws: DrawRepo;
  readonly ledger: LedgerRepo;
  readonly pity: PityRepo;
  readonly paymentOrders: PaymentOrderRepo;
  readonly adminAudit: AdminAuditRepo;
}
