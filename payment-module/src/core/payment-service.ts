import { createChannelRegistry, getChannelAdapter } from '../channels/channel-registry.js';
import type { PaymentChannelAdapter } from '../channels/types.js';
import type { ResolvedPaymentConfig } from '../config/schema.js';
import { loadPaymentConfig } from '../config/load-config.js';
import { MemoryOrderStore } from '../store/memory-order-store.js';
import {
  detectClientPlatform,
  detectPlatformForChannel,
  resolvePaymentPresentation,
} from './client-platform.js';
import type {
  CheckoutResult,
  CreateCheckoutInput,
  NotifyHandleResult,
  PaymentChannel,
  PaymentPresentation,
  QueryOrderResult,
  RefundInput,
  RefundResult,
} from '../types.js';
import { channelDisabled, presentationNotSupported } from '../utils/errors.js';
import { generateOrderNo, generateRefundNo } from '../utils/order-no.js';

export interface PaymentModuleOptions {
  /** 配置文件路径，默认 `./config/payment.config.json` */
  readonly configPath?: string;
}

export interface PaymentModule {
  readonly config: ResolvedPaymentConfig;
  readonly configPath: string;

  /** 根据 User-Agent 判断手机端 / 电脑端 */
  detectClientPlatform(userAgent: string): ReturnType<typeof detectClientPlatform>;

  /** 针对指定渠道给出推荐呈现方式 */
  detectPlatformForChannel(
    userAgent: string,
    channel: PaymentChannel,
  ): ReturnType<typeof detectPlatformForChannel>;

  /**
   * 创建收银台：手机端返回跳转 URL 或微信 JSAPI；电脑端返回二维码内容。
   */
  createCheckout(input: CreateCheckoutInput): Promise<CheckoutResult>;

  /** 验签并处理支付回调，推进订单为已支付 */
  handlePaymentNotify(
    channel: PaymentChannel,
    headers: Record<string, string>,
    rawBody: string,
  ): Promise<NotifyHandleResult>;

  /** 查询本地订单，并可选向渠道查单 */
  queryOrder(orderNo: string, options?: { syncChannel?: boolean }): Promise<QueryOrderResult>;

  /** 发起退款 */
  requestRefund(input: RefundInput): Promise<RefundResult>;

  /** 获取内存中的订单快照（模块内置存储，接入方可替换为自有仓储） */
  getOrder(orderNo: string): ReturnType<MemoryOrderStore['get']>;

  /** 开始业务履约（paid -> fulfilling） */
  beginFulfillment(orderNo: string): ReturnType<MemoryOrderStore['get']>;

  /** 完成业务履约（fulfilling -> fulfilled） */
  completeFulfillment(orderNo: string): ReturnType<MemoryOrderStore['get']>;
}

export function createPaymentModule(options: PaymentModuleOptions = {}): PaymentModule {
  const configPath = options.configPath ?? './config/payment.config.json';
  const config = loadPaymentConfig(configPath);
  const store = new MemoryOrderStore();
  const channels = createChannelRegistry(config, configPath);

  return {
    config,
    configPath,

    detectClientPlatform(userAgent: string) {
      return detectClientPlatform(userAgent);
    },

    detectPlatformForChannel(userAgent, channel) {
      return detectPlatformForChannel(userAgent, channel);
    },

    async createCheckout(input: CreateCheckoutInput): Promise<CheckoutResult> {
      assertChannelEnabled(config, input.channel);
      const adapter = getChannelAdapter(channels, input.channel);

      const presentation =
        input.presentationOverride ?? resolvePaymentPresentation(input.userAgent, input.channel);

      validatePresentation(input.channel, presentation, input.wechatOpenid);

      const existing = store.findByUserRequest(input.userId, input.clientRequestId);
      if (existing && (existing.status === 'pending' || existing.status === 'created')) {
        return rebuildCheckoutFromOrder(existing, adapter);
      }

      const orderNo = generateOrderNo();
      const expireAt = new Date(
        Date.now() + config.orderExpireMinutes * 60 * 1000,
      ).toISOString();

      const order = store.create({
        orderNo,
        userId: input.userId,
        clientRequestId: input.clientRequestId,
        channel: input.channel,
        presentation,
        subject: input.subject,
        body: input.body ?? input.subject,
        businessType: input.businessType,
        businessId: input.businessId ?? '',
        productSnapshot: input.productSnapshot ?? {},
        amountCents: input.amountCents,
        expireAt,
      });

      const channelResult = await adapter.createPayment({
        orderNo: order.orderNo,
        amountCents: order.amountCents,
        subject: order.subject,
        body: order.body,
        clientIp: input.clientIp,
        presentation,
        expireAt: order.expireAt,
        wechatOpenid: input.wechatOpenid,
      });

      store.markPending(order.orderNo);

      const platform = detectClientPlatform(input.userAgent);
      const base = {
        channel: input.channel,
        orderNo: order.orderNo,
        amountCents: order.amountCents,
        currency: order.currency,
        expireAt: order.expireAt,
        platform,
      };

      if (presentation === 'qrcode') {
        if (!channelResult.qrCodeContent) {
          throw presentationNotSupported('渠道未返回二维码内容');
        }
        return {
          presentation: 'qrcode',
          ...base,
          qrCodeContent: channelResult.qrCodeContent,
        };
      }

      if (presentation === 'redirect_h5') {
        if (!channelResult.redirectUrl) {
          throw presentationNotSupported('渠道未返回跳转地址');
        }
        return {
          presentation: 'redirect_h5',
          ...base,
          redirectUrl: channelResult.redirectUrl,
        };
      }

      if (!channelResult.jsapiParams) {
        throw presentationNotSupported('渠道未返回 JSAPI 参数');
      }

      return {
        presentation: 'wechat_jsapi',
        channel: 'wechat',
        orderNo: order.orderNo,
        amountCents: order.amountCents,
        currency: order.currency,
        expireAt: order.expireAt,
        platform,
        jsapiParams: channelResult.jsapiParams,
      };
    },

    async handlePaymentNotify(channel, headers, rawBody) {
      const adapter = getChannelAdapter(channels, channel);
      const verified = await adapter.verifyNotify(headers, rawBody);

      if (!verified.verified) {
        const fail = adapter.failureNotifyResponse();
        return {
          verified: false,
          alreadyProcessed: false,
          channelResponseBody: fail.body,
          channelResponseContentType: fail.contentType,
        };
      }

      const order = store.get(verified.orderNo);
      if (order.amountCents !== verified.amountCents) {
        const fail = adapter.failureNotifyResponse();
        return {
          verified: false,
          alreadyProcessed: false,
          channelResponseBody: fail.body,
          channelResponseContentType: fail.contentType,
        };
      }

      const alreadyProcessed = order.status === 'paid' || order.status === 'fulfilled';
      const updated = alreadyProcessed
        ? order
        : store.markPaid(order.orderNo, verified.channelTradeNo, verified.paidAt);

      const success = adapter.successNotifyResponse();
      return {
        verified: true,
        alreadyProcessed,
        order: updated,
        notify: {
          channel,
          orderNo: verified.orderNo,
          channelTradeNo: verified.channelTradeNo,
          amountCents: verified.amountCents,
          currency: 'CNY',
          paidAt: verified.paidAt,
          payerId: verified.payerId,
          rawNotifyId: verified.notifyId,
        },
        channelResponseBody: success.body,
        channelResponseContentType: success.contentType,
      };
    },

    async queryOrder(orderNo, options = {}) {
      const order = store.get(orderNo);
      if (!options.syncChannel) {
        return { order, channelPaid: order.status === 'paid' || order.status === 'fulfilled' };
      }

      const adapter = getChannelAdapter(channels, order.channel);
      const remote = await adapter.queryPayment(orderNo);
      if (remote.paid && order.status !== 'paid' && order.status !== 'fulfilled') {
        const updated = store.markPaid(
          orderNo,
          remote.channelTradeNo,
          remote.paidAt ?? new Date().toISOString(),
        );
        return { order: updated, channelPaid: true };
      }

      return { order, channelPaid: remote.paid };
    },

    async requestRefund(input: RefundInput) {
      const order = store.get(input.orderNo);
      if (order.status !== 'paid' && order.status !== 'fulfilled' && order.status !== 'compensate_required') {
        throw presentationNotSupported(`当前订单状态不可退款: ${order.status}`);
      }

      const adapter = getChannelAdapter(channels, order.channel);
      const refundNo = input.refundNo || generateRefundNo();

      const result = await adapter.createRefund({
        orderNo: order.orderNo,
        refundNo,
        totalAmountCents: order.amountCents,
        refundAmountCents: input.amountCents,
        reason: input.reason ?? '业务退款',
        channelTradeNo: order.channelTradeNo,
      });

      if (result.status === 'success') {
        store.updateStatus(order.orderNo, 'refunded');
      } else if (result.status === 'processing') {
        store.updateStatus(order.orderNo, 'refund_requested');
      } else {
        store.updateStatus(order.orderNo, 'refund_failed');
      }

      return {
        refundNo,
        orderNo: order.orderNo,
        status: result.status,
        channelRefundNo: result.channelRefundNo,
        message: result.message,
      };
    },

    getOrder(orderNo: string) {
      return store.get(orderNo);
    },

    beginFulfillment(orderNo: string) {
      return store.beginFulfillment(orderNo);
    },

    completeFulfillment(orderNo: string) {
      return store.completeFulfillment(orderNo);
    },
  };
}

async function rebuildCheckoutFromOrder(
  order: ReturnType<MemoryOrderStore['get']>,
  adapter: PaymentChannelAdapter,
): Promise<CheckoutResult> {
  const channelResult = await adapter.createPayment({
    orderNo: order.orderNo,
    amountCents: order.amountCents,
    subject: order.subject,
    body: order.body,
    clientIp: '127.0.0.1',
    presentation: order.presentation,
    expireAt: order.expireAt,
  });

  const platform = order.presentation === 'qrcode' ? 'desktop' : 'mobile';

  const base = {
    channel: order.channel,
    orderNo: order.orderNo,
    amountCents: order.amountCents,
    currency: order.currency,
    expireAt: order.expireAt,
    platform: platform as 'mobile' | 'desktop',
  };

  if (order.presentation === 'qrcode' && channelResult.qrCodeContent) {
    return { presentation: 'qrcode', ...base, qrCodeContent: channelResult.qrCodeContent };
  }
  if (order.presentation === 'redirect_h5' && channelResult.redirectUrl) {
    return { presentation: 'redirect_h5', ...base, redirectUrl: channelResult.redirectUrl };
  }
  if (order.presentation === 'wechat_jsapi' && channelResult.jsapiParams) {
    return {
      presentation: 'wechat_jsapi',
      channel: 'wechat',
      orderNo: order.orderNo,
      amountCents: order.amountCents,
      currency: order.currency,
      expireAt: order.expireAt,
      platform: 'mobile',
      jsapiParams: channelResult.jsapiParams,
    };
  }

  throw presentationNotSupported('无法重建收银台');
}

function assertChannelEnabled(config: ResolvedPaymentConfig, channel: PaymentChannel): void {
  if (channel === 'wechat' && !config.wechat?.enabled) {
    throw channelDisabled('wechat');
  }
  if (channel === 'alipay' && !config.alipay?.enabled) {
    throw channelDisabled('alipay');
  }
}

function validatePresentation(
  channel: PaymentChannel,
  presentation: PaymentPresentation,
  openid?: string,
): void {
  if (presentation === 'wechat_jsapi') {
    if (channel !== 'wechat') {
      throw presentationNotSupported('仅微信支付支持 JSAPI');
    }
    if (!openid) {
      throw presentationNotSupported('微信 JSAPI 需要 wechatOpenid');
    }
  }
  if (channel === 'alipay' && presentation === 'wechat_jsapi') {
    throw presentationNotSupported('支付宝不支持 JSAPI');
  }
}
