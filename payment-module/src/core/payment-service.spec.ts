import { writeFileSync, mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { createPaymentModule } from './payment-service.js';

const MOBILE_UA =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1';

const DESKTOP_UA =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36';

const WECHAT_MOBILE_UA =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) MicroMessenger/8.0.43 Mobile/15E148';

describe('payment-service (mock)', () => {
  let configPath: string;
  let payment: ReturnType<typeof createPaymentModule>;

  beforeAll(() => {
    const dir = mkdtempSync(join(tmpdir(), 'payment-module-'));
    configPath = join(dir, 'payment.config.json');
    writeFileSync(
      configPath,
      JSON.stringify({
        notifyBaseUrl: 'https://localhost:18100',
        orderExpireMinutes: 30,
        mock: true,
        wechat: {
          enabled: true,
          appId: 'wx_mock',
          mchId: '1900000001',
          apiV3Key: '01234567890123456789012345678901',
          serialNo: 'MOCK',
          privateKeyPath: './unused.pem',
          platformCertPath: './unused.pem',
          notifyPath: '/payments/wechat/notify',
          h5AppName: 'Mock',
          h5AppUrl: 'https://localhost:3000',
        },
        alipay: {
          enabled: true,
          appId: '2021000000000000',
          privateKeyPath: './unused.pem',
          alipayPublicKeyPath: './unused.pem',
          gateway: 'https://openapi.alipay.com/gateway.do',
          notifyPath: '/payments/alipay/notify',
          signType: 'RSA2',
        },
      }),
    );
    payment = createPaymentModule({ configPath });
  });

  it('creates desktop wechat qrcode checkout', async () => {
    const checkout = await payment.createCheckout({
      userId: 'u1',
      clientRequestId: 'req_desktop_wx_1',
      channel: 'wechat',
      amountCents: 100,
      subject: '测试商品',
      businessType: 'points_pack',
      userAgent: DESKTOP_UA,
      clientIp: '127.0.0.1',
    });

    expect(checkout.presentation).toBe('qrcode');
    expect(checkout.platform).toBe('desktop');
    expect('qrCodeContent' in checkout && checkout.qrCodeContent.length > 0).toBe(true);
  });

  it('creates mobile alipay redirect checkout', async () => {
    const checkout = await payment.createCheckout({
      userId: 'u1',
      clientRequestId: 'req_mobile_ali_1',
      channel: 'alipay',
      amountCents: 200,
      subject: '测试商品',
      businessType: 'points_pack',
      userAgent: MOBILE_UA,
      clientIp: '127.0.0.1',
    });

    expect(checkout.presentation).toBe('redirect_h5');
    expect('redirectUrl' in checkout && checkout.redirectUrl.length > 0).toBe(true);
  });

  it('creates wechat jsapi in wechat browser', async () => {
    const checkout = await payment.createCheckout({
      userId: 'u1',
      clientRequestId: 'req_jsapi_1',
      channel: 'wechat',
      amountCents: 300,
      subject: '测试商品',
      businessType: 'points_pack',
      userAgent: WECHAT_MOBILE_UA,
      clientIp: '127.0.0.1',
      wechatOpenid: 'oMockOpenId',
    });

    expect(checkout.presentation).toBe('wechat_jsapi');
    expect('jsapiParams' in checkout && checkout.jsapiParams.paySign.length > 0).toBe(true);
  });

  it('is idempotent for same client_request_id', async () => {
    const first = await payment.createCheckout({
      userId: 'u2',
      clientRequestId: 'req_idem_1',
      channel: 'wechat',
      amountCents: 50,
      subject: '幂等',
      businessType: 'test',
      userAgent: DESKTOP_UA,
      clientIp: '127.0.0.1',
    });
    const second = await payment.createCheckout({
      userId: 'u2',
      clientRequestId: 'req_idem_1',
      channel: 'wechat',
      amountCents: 50,
      subject: '幂等',
      businessType: 'test',
      userAgent: DESKTOP_UA,
      clientIp: '127.0.0.1',
    });

    expect(first.orderNo).toBe(second.orderNo);
  });

  it('handles mock wechat notify', async () => {
    const checkout = await payment.createCheckout({
      userId: 'u3',
      clientRequestId: 'req_notify_1',
      channel: 'wechat',
      amountCents: 99,
      subject: '回调',
      businessType: 'test',
      userAgent: DESKTOP_UA,
      clientIp: '127.0.0.1',
    });

    const body = JSON.stringify({
      out_trade_no: checkout.orderNo,
      transaction_id: 'wx_mock_tx_1',
      amount: 99,
    });

    const result = await payment.handlePaymentNotify('wechat', {}, body);
    expect(result.verified).toBe(true);
    expect(result.order?.status).toBe('paid');
  });
});
