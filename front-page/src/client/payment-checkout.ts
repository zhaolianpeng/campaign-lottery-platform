import {
  createPaymentCheckout,
  fetchPaymentPublicConfig,
  fulfillPaymentOrder,
  pickDefaultChannel,
  pollPaymentUntilPaid,
} from './payment-api';
import type { CheckoutResult, CreateCheckoutInput, PaymentChannel } from '@/types/payment';

declare global {
  interface Window {
    WeixinJSBridge?: {
      invoke: (
        api: string,
        params: Record<string, string>,
        callback: (res: { err_msg?: string }) => void,
      ) => void;
    };
  }
}

export interface RunPaymentCheckoutOptions {
  readonly token: string;
  readonly input: CreateCheckoutInput;
  readonly channel?: PaymentChannel;
  readonly onQrcode?: (checkout: CheckoutResult & { presentation: 'qrcode' }) => void | Promise<void>;
}

function qrImageUrl(content: string): string {
  return `https://api.qrserver.com/v1/create-qr-code/?size=240x240&data=${encodeURIComponent(content)}`;
}

function invokeWechatJsapi(params: CheckoutResult & { presentation: 'wechat_jsapi' }): Promise<void> {
  return new Promise((resolve, reject) => {
    const bridge = window.WeixinJSBridge;
    if (!bridge) {
      reject(new Error('请在微信内打开以完成支付'));
      return;
    }
    bridge.invoke(
      'getBrandWCPayRequest',
      {
        appId: params.jsapi_params.appId,
        timeStamp: params.jsapi_params.timeStamp,
        nonceStr: params.jsapi_params.nonceStr,
        package: params.jsapi_params.package,
        signType: params.jsapi_params.signType,
        paySign: params.jsapi_params.paySign,
      },
      (res) => {
        const msg = res.err_msg ?? '';
        if (msg.includes(':ok')) {
          resolve();
          return;
        }
        reject(new Error(msg || '微信支付未完成'));
      },
    );
  });
}

async function presentCheckout(
  checkout: CheckoutResult,
  onQrcode?: RunPaymentCheckoutOptions['onQrcode'],
): Promise<void> {
  if (checkout.presentation === 'redirect_h5') {
    window.location.href = checkout.redirect_url;
    return;
  }

  if (checkout.presentation === 'wechat_jsapi') {
    await invokeWechatJsapi(checkout);
    return;
  }

  if (checkout.presentation === 'qrcode') {
    if (onQrcode) {
      await onQrcode(checkout);
      return;
    }
    const amountYuan = (checkout.amount_cents / 100).toFixed(2);
    const html = [
      '<div style="font-family:sans-serif;text-align:center;padding:8px">',
      `<p style="margin:0 0 12px">请使用${checkout.channel === 'wechat' ? '微信' : '支付宝'}扫码支付 ¥${amountYuan}</p>`,
      `<img src="${qrImageUrl(checkout.qr_code_content)}" width="240" height="240" alt="支付二维码" />`,
      '<p style="margin:12px 0 0;font-size:12px;color:#666">支付完成后请关闭此窗口</p>',
      '</div>',
    ].join('');
    const popup = window.open('', '_blank', 'width=320,height=400');
    if (popup) {
      popup.document.write(html);
      popup.document.close();
      return;
    }
    window.alert(`请扫码支付 ¥${amountYuan}\n\n${checkout.qr_code_content}`);
  }
}

/**
 * 创建收银台 → 拉起支付 → 轮询确认 → 履约
 */
export async function runPaymentCheckout(options: RunPaymentCheckoutOptions): Promise<string> {
  const config = await fetchPaymentPublicConfig();
  if (!config.enabled) {
    throw new Error('支付功能未启用');
  }

  const channel = options.channel ?? pickDefaultChannel(config);
  const checkout = await createPaymentCheckout(options.token, {
    ...options.input,
    channel,
  });

  if (config.mock) {
    await fulfillPaymentOrder(options.token, checkout.order_no);
    return checkout.order_no;
  }

  if (checkout.presentation === 'qrcode' && options.onQrcode) {
    await options.onQrcode(checkout);
    return checkout.order_no;
  }

  if (checkout.presentation !== 'qrcode') {
    await presentCheckout(checkout, options.onQrcode);
  } else {
    await presentCheckout(checkout);
  }

  if (checkout.presentation === 'qrcode') {
    await pollPaymentUntilPaid(options.token, checkout.order_no, { maxAttempts: 90 });
    await fulfillPaymentOrder(options.token, checkout.order_no);
    return checkout.order_no;
  }

  if (checkout.presentation === 'redirect_h5') {
    sessionStorage.setItem('pending_payment_order', checkout.order_no);
    return checkout.order_no;
  }

  await pollPaymentUntilPaid(options.token, checkout.order_no);
  await fulfillPaymentOrder(options.token, checkout.order_no);
  return checkout.order_no;
}

/** 从支付页返回后恢复履约（H5 跳转场景） */
export async function resumePendingPayment(token: string): Promise<boolean> {
  const orderNo = sessionStorage.getItem('pending_payment_order');
  if (!orderNo) {
    return false;
  }
  try {
    await pollPaymentUntilPaid(token, orderNo, { maxAttempts: 3, intervalMs: 1000 });
    await fulfillPaymentOrder(token, orderNo);
    sessionStorage.removeItem('pending_payment_order');
    return true;
  } catch {
    return false;
  }
}

export function formatCentsToYuan(cents: number): string {
  return `¥${(cents / 100).toFixed(2)}`;
}
