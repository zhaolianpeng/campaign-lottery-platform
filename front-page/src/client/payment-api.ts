import { apiPostRequest, apiRequest } from './api';
import type {
  CheckoutResult,
  CreateCheckoutInput,
  PaymentChannel,
  PaymentPublicConfig,
  QueryPaymentOrderResult,
} from '@/types/payment';

export interface PlatformDetectResult {
  readonly platform: 'mobile' | 'desktop';
  readonly isWechatBrowser: boolean;
  readonly recommendedPresentation: string;
  readonly channel: PaymentChannel | null;
}

export function fetchPaymentPublicConfig(): Promise<PaymentPublicConfig> {
  return apiRequest<PaymentPublicConfig>('/api/v1/payments/config/public', '');
}

export function fetchPaymentPlatform(channel?: PaymentChannel): Promise<PlatformDetectResult | { readonly platform: string; readonly wechat: PlatformDetectResult; readonly alipay: PlatformDetectResult }> {
  const query = channel ? `?channel=${channel}` : '';
  return apiRequest(`/api/v1/payments/platform${query}`, '');
}

export function createPaymentCheckout(token: string, input: CreateCheckoutInput): Promise<CheckoutResult> {
  return apiPostRequest<CheckoutResult>('/api/v1/payments/orders', token, {
    body: JSON.stringify({
      client_request_id: input.client_request_id,
      channel: input.channel,
      amount_cents: input.amount_cents,
      subject: input.subject,
      body: input.body,
      business_type: input.business_type,
      business_id: input.business_id,
      product_snapshot: input.product_snapshot,
    }),
  });
}

export function queryPaymentOrder(
  token: string,
  orderNo: string,
  syncChannel = false,
): Promise<QueryPaymentOrderResult> {
  const query = syncChannel ? '?sync_channel=true' : '';
  return apiRequest<QueryPaymentOrderResult>(`/api/v1/payments/orders/${encodeURIComponent(orderNo)}${query}`, token);
}

export function fulfillPaymentOrder(
  token: string,
  orderNo: string,
): Promise<{ readonly order_no: string; readonly status: string; readonly already_fulfilled: boolean }> {
  return apiPostRequest(`/api/v1/payments/orders/${encodeURIComponent(orderNo)}/fulfill`, token, {
    body: JSON.stringify({}),
  });
}

export function pickDefaultChannel(config: PaymentPublicConfig, prefer: PaymentChannel = 'wechat'): PaymentChannel {
  if (prefer === 'wechat' && config.channels.wechat) {
    return 'wechat';
  }
  if (prefer === 'alipay' && config.channels.alipay) {
    return 'alipay';
  }
  if (config.channels.wechat) {
    return 'wechat';
  }
  if (config.channels.alipay) {
    return 'alipay';
  }
  return prefer;
}

export async function pollPaymentUntilPaid(
  token: string,
  orderNo: string,
  options: { readonly maxAttempts?: number; readonly intervalMs?: number } = {},
): Promise<QueryPaymentOrderResult> {
  const maxAttempts = options.maxAttempts ?? 60;
  const intervalMs = options.intervalMs ?? 2000;

  for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
    const result = await queryPaymentOrder(token, orderNo, true);
    if (result.order.status === 'paid' || result.order.status === 'fulfilled' || result.channel_paid) {
      return result;
    }
    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }

  throw new Error('支付结果确认超时，请稍后在订单记录中查看');
}
