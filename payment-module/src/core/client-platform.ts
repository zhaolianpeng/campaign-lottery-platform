import type {
  ClientPlatform,
  PaymentChannel,
  PaymentPresentation,
  PlatformDetectResult,
} from '../types.js';

const MOBILE_UA_PATTERN =
  /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini|Mobile|mobile/i;

const WECHAT_UA_PATTERN = /MicroMessenger/i;

export function isMobileUserAgent(userAgent: string): boolean {
  return MOBILE_UA_PATTERN.test(userAgent);
}

export function isWechatBrowser(userAgent: string): boolean {
  return WECHAT_UA_PATTERN.test(userAgent);
}

export function detectClientPlatform(userAgent: string): ClientPlatform {
  return isMobileUserAgent(userAgent) ? 'mobile' : 'desktop';
}

/**
 * 根据访问端与渠道，选择支付呈现方式。
 * 手机端：拉起本地支付 App（微信 H5 / 支付宝 WAP；微信内置浏览器用 JSAPI）。
 * 电脑端：返回扫码支付内容（微信 Native / 支付宝 precreate）。
 */
export function resolvePaymentPresentation(
  userAgent: string,
  channel: PaymentChannel,
): PaymentPresentation {
  const mobile = isMobileUserAgent(userAgent);
  const inWechat = isWechatBrowser(userAgent);

  if (mobile) {
    if (channel === 'wechat' && inWechat) {
      return 'wechat_jsapi';
    }
    return 'redirect_h5';
  }

  return 'qrcode';
}

export function detectPlatformForChannel(
  userAgent: string,
  channel: PaymentChannel,
): PlatformDetectResult {
  const platform = detectClientPlatform(userAgent);
  const inWechatBrowser = isWechatBrowser(userAgent);
  const recommendedPresentation = resolvePaymentPresentation(userAgent, channel);

  return {
    platform,
    isWechatBrowser: inWechatBrowser,
    recommendedPresentation,
    channel,
  };
}
