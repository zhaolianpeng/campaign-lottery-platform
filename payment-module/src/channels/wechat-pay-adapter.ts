import { randomBytes } from 'node:crypto';
import { dirname, resolve } from 'node:path';
import type { ResolvedPaymentConfig } from '../config/schema.js';
import { readPemFile } from '../config/load-config.js';
import {
  buildJsapiPaySign,
  buildWechatAuthorization,
  decryptWechatResource,
  verifyWechatNotifySignature,
} from '../crypto/wechat-sign.js';
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

const WECHAT_API = 'https://api.mch.weixin.qq.com';

export function createWechatPayAdapter(
  config: ResolvedPaymentConfig,
  configFilePath: string,
): PaymentChannelAdapter {
  if (!config.wechat?.enabled) {
    throw channelApiError('微信支付未启用');
  }

  const wechat = config.wechat;
  const mock = config.mock;
  const configDir = dirname(resolve(configFilePath));
  const mockKeys = mock ? getMockRsaKeyPair() : null;
  const privateKeyPem = mockKeys
    ? mockKeys.privateKeyPem
    : readPemFile(wechat.privateKeyPath, configDir);
  const platformPublicKeyPem = mockKeys
    ? mockKeys.publicKeyPem
    : readPemFile(wechat.platformCertPath, configDir);
  const notifyUrl = config.wechatNotifyUrl;

  return {
    channel: 'wechat',

    async createPayment(input: ChannelCreatePaymentInput): Promise<ChannelCreatePaymentOutput> {
      if (mock) {
        return mockCreate(input, wechat.appId, privateKeyPem);
      }

      const amount = input.amountCents;
      const description = input.subject;
      const expireTime = toWechatExpire(input.expireAt);

      if (input.presentation === 'qrcode') {
        const path = '/v3/pay/transactions/native';
        const body = JSON.stringify({
          appid: wechat.appId,
          mchid: wechat.mchId,
          description,
          out_trade_no: input.orderNo,
          notify_url: notifyUrl,
          amount: { total: amount, currency: 'CNY' },
          time_expire: expireTime,
        });
        const data = await wechatRequest('POST', path, body, wechat, privateKeyPem);
        return { qrCodeContent: String(data.code_url ?? '') };
      }

      if (input.presentation === 'redirect_h5') {
        const path = '/v3/pay/transactions/h5';
        const body = JSON.stringify({
          appid: wechat.appId,
          mchid: wechat.mchId,
          description,
          out_trade_no: input.orderNo,
          notify_url: notifyUrl,
          amount: { total: amount, currency: 'CNY' },
          time_expire: expireTime,
          scene_info: {
            payer_client_ip: input.clientIp,
            h5_info: {
              type: 'Wap',
              app_name: wechat.h5AppName,
              app_url: wechat.h5AppUrl,
            },
          },
        });
        const data = await wechatRequest('POST', path, body, wechat, privateKeyPem);
        return { redirectUrl: String(data.h5_url ?? '') };
      }

      if (input.presentation === 'wechat_jsapi') {
        if (!input.wechatOpenid) {
          throw channelApiError('微信 JSAPI 支付需要 openid');
        }
        const path = '/v3/pay/transactions/jsapi';
        const body = JSON.stringify({
          appid: wechat.appId,
          mchid: wechat.mchId,
          description,
          out_trade_no: input.orderNo,
          notify_url: notifyUrl,
          amount: { total: amount, currency: 'CNY' },
          time_expire: expireTime,
          payer: { openid: input.wechatOpenid },
        });
        const data = await wechatRequest('POST', path, body, wechat, privateKeyPem);
        const prepayId = String(data.prepay_id ?? '');
        const timeStamp = Math.floor(Date.now() / 1000).toString();
        const nonceStr = randomBytes(16).toString('hex');
        const packageValue = `prepay_id=${prepayId}`;
        const paySign = buildJsapiPaySign(
          wechat.appId,
          timeStamp,
          nonceStr,
          packageValue,
          privateKeyPem,
        );
        return {
          jsapiParams: {
            appId: wechat.appId,
            timeStamp,
            nonceStr,
            package: packageValue,
            signType: 'RSA',
            paySign,
          },
        };
      }

      throw channelApiError(`不支持的微信支付呈现方式: ${input.presentation}`);
    },

    async verifyNotify(headers, body) {
      if (mock) {
        const payload = JSON.parse(body) as { out_trade_no: string; transaction_id: string; amount: number };
        return {
          verified: true,
          orderNo: payload.out_trade_no,
          channelTradeNo: payload.transaction_id,
          amountCents: payload.amount,
          paidAt: new Date().toISOString(),
          payerId: '',
          notifyId: `mock_${Date.now()}`,
        };
      }

      const timestamp = headers['wechatpay-timestamp'] ?? headers['Wechatpay-Timestamp'] ?? '';
      const nonce = headers['wechatpay-nonce'] ?? headers['Wechatpay-Nonce'] ?? '';
      const signature = headers['wechatpay-signature'] ?? headers['Wechatpay-Signature'] ?? '';

      const verified = verifyWechatNotifySignature(
        timestamp,
        nonce,
        body,
        signature,
        platformPublicKeyPem,
      );

      if (!verified) {
        return {
          verified: false,
          orderNo: '',
          channelTradeNo: '',
          amountCents: 0,
          paidAt: '',
          payerId: '',
          notifyId: '',
        };
      }

      const envelope = JSON.parse(body) as {
        id: string;
        resource: {
          associated_data: string;
          nonce: string;
          ciphertext: string;
        };
      };

      const plain = decryptWechatResource(
        wechat.apiV3Key,
        envelope.resource.associated_data,
        envelope.resource.nonce,
        envelope.resource.ciphertext,
      );

      const trade = JSON.parse(plain) as {
        out_trade_no: string;
        transaction_id: string;
        trade_state: string;
        success_time?: string;
        payer?: { openid?: string };
        amount?: { total?: number };
      };

      if (trade.trade_state !== 'SUCCESS') {
        return {
          verified: false,
          orderNo: trade.out_trade_no ?? '',
          channelTradeNo: '',
          amountCents: 0,
          paidAt: '',
          payerId: '',
          notifyId: envelope.id,
        };
      }

      return {
        verified: true,
        orderNo: trade.out_trade_no,
        channelTradeNo: trade.transaction_id,
        amountCents: trade.amount?.total ?? 0,
        paidAt: trade.success_time ?? new Date().toISOString(),
        payerId: trade.payer?.openid ?? '',
        notifyId: envelope.id,
      };
    },

    async queryPayment(orderNo: string): Promise<ChannelQueryPaymentOutput> {
      if (mock) {
        return { paid: false, channelTradeNo: '', paidAt: null };
      }

      const path = `/v3/pay/transactions/out-trade-no/${encodeURIComponent(orderNo)}?mchid=${wechat.mchId}`;
      const data = await wechatRequest('GET', path, '', wechat, privateKeyPem);
      const paid = data.trade_state === 'SUCCESS';
      return {
        paid,
        channelTradeNo: String(data.transaction_id ?? ''),
        paidAt: paid ? String(data.success_time ?? new Date().toISOString()) : null,
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

      const path = '/v3/refund/domestic/refunds';
      const body = JSON.stringify({
        out_trade_no: input.orderNo,
        out_refund_no: input.refundNo,
        reason: input.reason,
        amount: {
          refund: input.refundAmountCents,
          total: input.totalAmountCents,
          currency: 'CNY',
        },
      });
      const data = await wechatRequest('POST', path, body, wechat, privateKeyPem);
      const status = data.status === 'SUCCESS' ? 'success' : 'processing';
      return {
        status,
        channelRefundNo: String(data.refund_id ?? ''),
        message: String(data.status ?? 'processing'),
      };
    },

    successNotifyResponse() {
      return { body: JSON.stringify({ code: 'SUCCESS', message: '成功' }), contentType: 'application/json' };
    },

    failureNotifyResponse() {
      return { body: JSON.stringify({ code: 'FAIL', message: '失败' }), contentType: 'application/json' };
    },
  };
}

async function wechatRequest(
  method: string,
  path: string,
  body: string,
  wechat: NonNullable<ResolvedPaymentConfig['wechat']>,
  privateKeyPem: string,
): Promise<Record<string, unknown>> {
  const { authorization } = buildWechatAuthorization(
    method,
    path,
    body,
    wechat.mchId,
    wechat.serialNo,
    privateKeyPem,
  );

  const response = await fetch(`${WECHAT_API}${path}`, {
    method,
    headers: {
      Authorization: authorization,
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: method === 'GET' ? undefined : body,
  });

  const text = await response.text();
  const data = text ? (JSON.parse(text) as Record<string, unknown>) : {};

  if (!response.ok) {
    throw channelApiError(
      `微信 API 错误: ${response.status} ${String(data.message ?? data.code ?? text)}`,
    );
  }

  return data;
}

function toWechatExpire(iso: string): string {
  const date = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}+08:00`;
}

function mockCreate(
  input: ChannelCreatePaymentInput,
  appId: string,
  privateKeyPem: string,
): ChannelCreatePaymentOutput {
  if (input.presentation === 'qrcode') {
    return { qrCodeContent: `weixin://wxpay/bizpayurl?pr=mock_${input.orderNo}` };
  }
  if (input.presentation === 'redirect_h5') {
    return { redirectUrl: `https://mock.wechat.pay/h5/${input.orderNo}` };
  }
  const timeStamp = Math.floor(Date.now() / 1000).toString();
  const nonceStr = randomBytes(8).toString('hex');
  const packageValue = `prepay_id=mock_${input.orderNo}`;
  return {
    jsapiParams: {
      appId,
      timeStamp,
      nonceStr,
      package: packageValue,
      signType: 'RSA',
      paySign: buildJsapiPaySign(appId, timeStamp, nonceStr, packageValue, privateKeyPem),
    },
  };
}
