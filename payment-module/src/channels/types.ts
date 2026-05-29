import type { PaymentPresentation } from '../types.js';

export interface ChannelCreatePaymentInput {
  readonly orderNo: string;
  readonly amountCents: number;
  readonly subject: string;
  readonly body: string;
  readonly clientIp: string;
  readonly presentation: PaymentPresentation;
  readonly expireAt: string;
  readonly wechatOpenid?: string;
}

export interface ChannelCreatePaymentOutput {
  readonly channelTradeNo?: string;
  readonly qrCodeContent?: string;
  readonly redirectUrl?: string;
  readonly jsapiParams?: {
    readonly appId: string;
    readonly timeStamp: string;
    readonly nonceStr: string;
    readonly package: string;
    readonly signType: 'RSA';
    readonly paySign: string;
  };
}

export interface ChannelQueryPaymentOutput {
  readonly paid: boolean;
  readonly channelTradeNo: string;
  readonly paidAt: string | null;
}

export interface ChannelRefundInput {
  readonly orderNo: string;
  readonly refundNo: string;
  readonly totalAmountCents: number;
  readonly refundAmountCents: number;
  readonly reason: string;
  readonly channelTradeNo: string;
}

export interface ChannelRefundOutput {
  readonly status: 'success' | 'processing' | 'failed';
  readonly channelRefundNo: string;
  readonly message: string;
}

export interface PaymentChannelAdapter {
  readonly channel: 'wechat' | 'alipay';

  createPayment(input: ChannelCreatePaymentInput): Promise<ChannelCreatePaymentOutput>;

  verifyNotify(headers: Record<string, string>, body: string): Promise<{
    verified: boolean;
    orderNo: string;
    channelTradeNo: string;
    amountCents: number;
    paidAt: string;
    payerId: string;
    notifyId: string;
  }>;

  queryPayment(orderNo: string): Promise<ChannelQueryPaymentOutput>;

  createRefund(input: ChannelRefundInput): Promise<ChannelRefundOutput>;

  successNotifyResponse(): { body: string; contentType: string };
  failureNotifyResponse(): { body: string; contentType: string };
}
