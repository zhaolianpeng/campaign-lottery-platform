import { dirname, resolve } from 'node:path';
import type { ResolvedPaymentConfig } from '../config/schema.js';
import { readPemFile } from '../config/load-config.js';
import {
  buildAlipayGetUrl,
  randomAlipayNonce,
  signAlipayParams,
  verifyAlipayParams,
} from '../crypto/alipay-sign.js';
import { getMockRsaKeyPair } from '../dev/mock-keys.js';
import { channelApiError } from '../utils/errors.js';
import type {
  ChannelCreatePaymentInput,
  ChannelCreatePaymentOutput,
  ChannelRefundInput,
  ChannelRefundOutput,
  ChannelQueryPaymentOutput,
  PaymentChannelAdapter,
} from './types.js';

export function createAlipayAdapter(
  config: ResolvedPaymentConfig,
  configFilePath: string,
): PaymentChannelAdapter {
  if (!config.alipay?.enabled) {
    throw channelApiError('支付宝未启用');
  }

  const alipay = config.alipay;
  const mock = config.mock;
  const configDir = dirname(resolve(configFilePath));
  const mockKeys = mock ? getMockRsaKeyPair() : null;
  const privateKeyPem = mockKeys
    ? mockKeys.privateKeyPem
    : readPemFile(alipay.privateKeyPath, configDir);
  const alipayPublicKeyPem = mockKeys
    ? mockKeys.publicKeyPem
    : readPemFile(alipay.alipayPublicKeyPath, configDir);
  const notifyUrl = config.alipayNotifyUrl;

  return {
    channel: 'alipay',

    async createPayment(input: ChannelCreatePaymentInput): Promise<ChannelCreatePaymentOutput> {
      if (mock) {
        if (input.presentation === 'qrcode') {
          return { qrCodeContent: `https://mock.alipay/qrcode/${input.orderNo}` };
        }
        return { redirectUrl: `https://mock.alipay/wap/${input.orderNo}` };
      }

      if (input.presentation === 'qrcode') {
        const bizContent = JSON.stringify({
          out_trade_no: input.orderNo,
          total_amount: (input.amountCents / 100).toFixed(2),
          subject: input.subject,
          timeout_express: `${Math.max(1, Math.floor((new Date(input.expireAt).getTime() - Date.now()) / 60000))}m`,
        });
        const data = await alipayRequest(alipay, privateKeyPem, 'alipay.trade.precreate', {
          notify_url: notifyUrl,
          biz_content: bizContent,
        });
        const response = data.alipay_trade_precreate_response as Record<string, string> | undefined;
        if (!response || response.code !== '10000') {
          throw channelApiError(`支付宝 precreate 失败: ${response?.sub_msg ?? response?.msg ?? 'unknown'}`);
        }
        return { qrCodeContent: response.qr_code };
      }

      if (input.presentation === 'redirect_h5') {
        const bizContent = JSON.stringify({
          out_trade_no: input.orderNo,
          total_amount: (input.amountCents / 100).toFixed(2),
          subject: input.subject,
          product_code: 'QUICK_WAP_WAY',
          quit_url: alipay.returnUrl ?? alipay.gateway,
        });
        const params = buildCommonParams(alipay, 'alipay.trade.wap.pay', {
          notify_url: notifyUrl,
          return_url: alipay.returnUrl ?? notifyUrl,
          biz_content: bizContent,
        });
        params.sign = signAlipayParams(params, privateKeyPem);
        return { redirectUrl: buildAlipayGetUrl(alipay.gateway, params) };
      }

      throw channelApiError(`不支持的支付宝呈现方式: ${input.presentation}`);
    },

    async verifyNotify(_headers, body) {
      const params = parseFormBody(body);
      if (mock) {
        return {
          verified: true,
          orderNo: params.out_trade_no ?? '',
          channelTradeNo: params.trade_no ?? '',
          amountCents: Math.round(Number(params.total_amount ?? '0') * 100),
          paidAt: params.gmt_payment ?? new Date().toISOString(),
          payerId: params.buyer_id ?? '',
          notifyId: params.notify_id ?? `mock_${Date.now()}`,
        };
      }

      const verified =
        verifyAlipayParams(params, alipayPublicKeyPem) &&
        (params.trade_status === 'TRADE_SUCCESS' || params.trade_status === 'TRADE_FINISHED');

      if (!verified) {
        return {
          verified: false,
          orderNo: params.out_trade_no ?? '',
          channelTradeNo: '',
          amountCents: 0,
          paidAt: '',
          payerId: '',
          notifyId: params.notify_id ?? '',
        };
      }

      return {
        verified: true,
        orderNo: params.out_trade_no ?? '',
        channelTradeNo: params.trade_no ?? '',
        amountCents: Math.round(Number(params.total_amount ?? '0') * 100),
        paidAt: params.gmt_payment ?? new Date().toISOString(),
        payerId: params.buyer_id ?? '',
        notifyId: params.notify_id ?? '',
      };
    },

    async queryPayment(orderNo: string): Promise<ChannelQueryPaymentOutput> {
      if (mock) {
        return { paid: false, channelTradeNo: '', paidAt: null };
      }

      const bizContent = JSON.stringify({ out_trade_no: orderNo });
      const data = await alipayRequest(alipay, privateKeyPem, 'alipay.trade.query', {
        biz_content: bizContent,
      });
      const response = data.alipay_trade_query_response as Record<string, string> | undefined;
      const paid =
        response?.trade_status === 'TRADE_SUCCESS' ||
        response?.trade_status === 'TRADE_FINISHED';
      return {
        paid: Boolean(paid),
        channelTradeNo: response?.trade_no ?? '',
        paidAt: paid ? (response?.send_pay_date ?? new Date().toISOString()) : null,
      };
    },

    async createRefund(input: ChannelRefundInput): Promise<ChannelRefundOutput> {
      if (mock) {
        return {
          status: 'success',
          channelRefundNo: `mock_refund_${input.refundNo}`,
          message: 'mock refund ok',
        };
      }

      const bizContent = JSON.stringify({
        out_trade_no: input.orderNo,
        refund_amount: (input.refundAmountCents / 100).toFixed(2),
        out_request_no: input.refundNo,
        refund_reason: input.reason,
      });
      const data = await alipayRequest(alipay, privateKeyPem, 'alipay.trade.refund', {
        biz_content: bizContent,
      });
      const response = data.alipay_trade_refund_response as Record<string, string> | undefined;
      if (!response || response.code !== '10000') {
        return {
          status: 'failed',
          channelRefundNo: '',
          message: response?.sub_msg ?? response?.msg ?? 'refund failed',
        };
      }
      return {
        status: 'success',
        channelRefundNo: response.trade_no ?? input.refundNo,
        message: 'success',
      };
    },

    successNotifyResponse() {
      return { body: 'success', contentType: 'text/plain' };
    },

    failureNotifyResponse() {
      return { body: 'failure', contentType: 'text/plain' };
    },
  };
}

function buildCommonParams(
  alipay: NonNullable<ResolvedPaymentConfig['alipay']>,
  method: string,
  extra: Record<string, string>,
): Record<string, string> {
  return {
    app_id: alipay.appId,
    method,
    format: 'JSON',
    charset: 'utf-8',
    sign_type: alipay.signType,
    timestamp: formatAlipayTimestamp(new Date()),
    version: '1.0',
    ...extra,
  };
}

async function alipayRequest(
  alipay: NonNullable<ResolvedPaymentConfig['alipay']>,
  privateKeyPem: string,
  method: string,
  extra: Record<string, string>,
): Promise<Record<string, unknown>> {
  const params = buildCommonParams(alipay, method, extra);
  params.sign = signAlipayParams(params, privateKeyPem);

  const response = await fetch(alipay.gateway, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=utf-8' },
    body: new URLSearchParams(params).toString(),
  });

  const text = await response.text();
  let data: Record<string, unknown>;
  try {
    data = JSON.parse(text) as Record<string, unknown>;
  } catch {
    throw channelApiError(`支付宝响应非 JSON: ${text.slice(0, 200)}`);
  }

  if (!response.ok) {
    throw channelApiError(`支付宝 HTTP 错误: ${response.status}`);
  }

  return data;
}

function formatAlipayTimestamp(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

function parseFormBody(body: string): Record<string, string> {
  const params: Record<string, string> = {};
  for (const part of body.split('&')) {
    const [rawKey, rawValue = ''] = part.split('=');
    if (!rawKey) {
      continue;
    }
    const key = decodeURIComponent(rawKey);
    const value = decodeURIComponent(rawValue.replace(/\+/g, ' '));
    params[key] = value;
  }
  return params;
}
