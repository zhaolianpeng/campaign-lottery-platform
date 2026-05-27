import type { PaymentChannel, PaymentOrderRecord, PaymentOrderStatus } from '../types.js';
import { invalidOrderState, orderNotFound } from '../utils/errors.js';

export interface CreateOrderParams {
  readonly orderNo: string;
  readonly userId: string;
  readonly clientRequestId: string;
  readonly channel: PaymentChannel;
  readonly presentation: PaymentOrderRecord['presentation'];
  readonly subject: string;
  readonly body: string;
  readonly businessType: string;
  readonly businessId: string;
  readonly productSnapshot: Record<string, unknown>;
  readonly amountCents: number;
  readonly expireAt: string;
}

export class MemoryOrderStore {
  private readonly byOrderNo = new Map<string, PaymentOrderRecord>();
  private readonly byUserRequest = new Map<string, string>();

  findByUserRequest(userId: string, clientRequestId: string): PaymentOrderRecord | undefined {
    const orderNo = this.byUserRequest.get(`${userId}:${clientRequestId}`);
    if (!orderNo) {
      return undefined;
    }
    return this.byOrderNo.get(orderNo);
  }

  get(orderNo: string): PaymentOrderRecord {
    const order = this.byOrderNo.get(orderNo);
    if (!order) {
      throw orderNotFound(orderNo);
    }
    return order;
  }

  create(params: CreateOrderParams): PaymentOrderRecord {
    const requestKey = `${params.userId}:${params.clientRequestId}`;
    const existingOrderNo = this.byUserRequest.get(requestKey);
    if (existingOrderNo) {
      const existing = this.byOrderNo.get(existingOrderNo);
      if (existing) {
        return existing;
      }
    }

    const now = new Date().toISOString();
    const record: PaymentOrderRecord = {
      orderNo: params.orderNo,
      userId: params.userId,
      clientRequestId: params.clientRequestId,
      channel: params.channel,
      presentation: params.presentation,
      subject: params.subject,
      body: params.body,
      businessType: params.businessType,
      businessId: params.businessId,
      productSnapshot: params.productSnapshot,
      amountCents: params.amountCents,
      currency: 'CNY',
      status: 'created',
      channelTradeNo: '',
      paidAt: null,
      expireAt: params.expireAt,
      createdAt: now,
      updatedAt: now,
    };

    this.byOrderNo.set(record.orderNo, record);
    this.byUserRequest.set(requestKey, record.orderNo);
    return record;
  }

  updateStatus(
    orderNo: string,
    status: PaymentOrderStatus,
    patch: Partial<Pick<PaymentOrderRecord, 'channelTradeNo' | 'paidAt'>> = {},
  ): PaymentOrderRecord {
    const current = this.get(orderNo);
    assertTransition(current.status, status);

    const updated: PaymentOrderRecord = {
      ...current,
      ...patch,
      status,
      updatedAt: new Date().toISOString(),
    };
    this.byOrderNo.set(orderNo, updated);
    return updated;
  }

  markPending(orderNo: string): PaymentOrderRecord {
    return this.updateStatus(orderNo, 'pending');
  }

  markPaid(orderNo: string, channelTradeNo: string, paidAt: string): PaymentOrderRecord {
    const current = this.get(orderNo);
    if (current.status === 'paid' || current.status === 'fulfilled') {
      return current;
    }
    return this.updateStatus(orderNo, 'paid', { channelTradeNo, paidAt });
  }
}

function assertTransition(from: PaymentOrderStatus, to: PaymentOrderStatus): void {
  const allowed: Record<PaymentOrderStatus, PaymentOrderStatus[]> = {
    created: ['pending', 'closed'],
    pending: ['paid', 'expired', 'closed'],
    paid: ['fulfilling', 'compensate_required'],
    fulfilling: ['fulfilled', 'compensate_required'],
    fulfilled: ['refund_requested'],
    compensate_required: ['fulfilling', 'refund_requested', 'fulfilled'],
    refund_requested: ['refunded', 'refund_failed'],
    refund_failed: ['refund_requested'],
    refunded: [],
    expired: [],
    closed: [],
  };

  if (!allowed[from].includes(to) && from !== to) {
    throw invalidOrderState(`非法状态转移: ${from} -> ${to}`);
  }
}
