import {
  detectClientPlatform,
  isMobileUserAgent,
  isWechatBrowser,
  resolvePaymentPresentation,
} from './client-platform.js';

const MOBILE_UA =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1';

const DESKTOP_UA =
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36';

const WECHAT_MOBILE_UA =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.43';

describe('client-platform', () => {
  it('detects mobile vs desktop', () => {
    expect(isMobileUserAgent(MOBILE_UA)).toBe(true);
    expect(isMobileUserAgent(DESKTOP_UA)).toBe(false);
    expect(detectClientPlatform(MOBILE_UA)).toBe('mobile');
    expect(detectClientPlatform(DESKTOP_UA)).toBe('desktop');
  });

  it('detects wechat browser', () => {
    expect(isWechatBrowser(WECHAT_MOBILE_UA)).toBe(true);
    expect(isWechatBrowser(MOBILE_UA)).toBe(false);
  });

  it('mobile wechat in-app uses jsapi', () => {
    expect(resolvePaymentPresentation(WECHAT_MOBILE_UA, 'wechat')).toBe('wechat_jsapi');
  });

  it('mobile non-wechat browser uses h5 redirect', () => {
    expect(resolvePaymentPresentation(MOBILE_UA, 'wechat')).toBe('redirect_h5');
    expect(resolvePaymentPresentation(MOBILE_UA, 'alipay')).toBe('redirect_h5');
  });

  it('desktop uses qrcode', () => {
    expect(resolvePaymentPresentation(DESKTOP_UA, 'wechat')).toBe('qrcode');
    expect(resolvePaymentPresentation(DESKTOP_UA, 'alipay')).toBe('qrcode');
  });
});
