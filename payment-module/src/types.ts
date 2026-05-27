/** 支付渠道 */
export type PaymentChannel = 'wechat' | 'alipay';

/** 客户端平台（由 User-Agent 推断） */
export type ClientPlatform = 'mobile' | 'desktop';

/**
 * 支付呈现方式：
 * - qrcode：桌面端扫码（微信 Native / 支付宝 precreate）
 * - redirect_h5：移动端浏览器拉起（微信 H5 / 支付宝 WAP）
 * - wechat_jsapi：微信内置浏览器 JSAPI 调起支付
 */
export type PaymentPresentation =
  | 'qrcode'
  | 'redirect_h5'
  | 'wechat_jsapi';

export type PaymentOrderStatus =
  | 'created'
  | 'pending'
  | 'paid'
  | 'fulfilling'
  | 'fulfilled'
  | 'compensate_required'
  | 'refund_requested'
  | 'refunded'
  | 'refund_failed'
  | 'expired'
  | 'closed';

export interface PaymentOrderRecord {
  readonly orderNo: string;
  readonly userId: string;
  readonly clientRequestId: string;
  readonly channel: PaymentChannel;
  readonly presentation: PaymentPresentation;
  readonly subject: string;
  readonly body: string;
  readonly businessType: string;
  readonly businessId: string;
  readonly productSnapshot: Record<string, unknown>;
  readonly amountCents: number;
  readonly currency: string;
  readonly status: PaymentOrderStatus;
  readonly channelTradeNo: string;
  readonly paidAt: string | null;
  readonly fulfilledAt: string | null;
  readonly expireAt: string;
  readonly createdAt: string;
  readonly updatedAt: string;
}

/** 创建收银台入参 */
export interface CreateCheckoutInput {
  readonly userId: string;
  readonly clientRequestId: string;
  readonly channel: PaymentChannel;
  readonly amountCents: number;
  readonly subject: string;
  readonly body?: string;
  readonly businessType: string;
  readonly businessId?: string;
  readonly productSnapshot?: Record<string, unknown>;
  /** 用于判断手机端 / 电脑端及微信内置浏览器 */
  readonly userAgent: string;
  readonly clientIp: string;
  /** 微信 JSAPI 必填：用户 openid */
  readonly wechatOpenid?: string;
  /** 强制指定呈现方式，不传则根据 userAgent 自动选择 */
  readonly presentationOverride?: PaymentPresentation;
}

/** 桌面端：二维码内容（业务侧自行渲染为图片） */
export interface QrcodeCheckoutResult {
  readonly presentation: 'qrcode';
  readonly channel: PaymentChannel;
  readonly orderNo: string;
  readonly amountCents: number;
  readonly currency: string;
  readonly expireAt: string;
  /** 微信 code_url 或支付宝 qr_code 原文 */
  readonly qrCodeContent: string;
  readonly platform: ClientPlatform;
}

/** 移动端：跳转 URL 拉起支付 App / 收银台 */
export interface RedirectCheckoutResult {
  readonly presentation: 'redirect_h5';
  readonly channel: PaymentChannel;
  readonly orderNo: string;
  readonly amountCents: number;
  readonly currency: string;
  readonly expireAt: string;
  readonly redirectUrl: string;
  readonly platform: ClientPlatform;
}

/** 微信内置浏览器 JSAPI 参数 */
export interface WechatJsapiCheckoutResult {
  readonly presentation: 'wechat_jsapi';
  readonly channel: 'wechat';
  readonly orderNo: string;
  readonly amountCents: number;
  readonly currency: string;
  readonly expireAt: string;
  readonly platform: ClientPlatform;
  readonly jsapiParams: {
    readonly appId: string;
    readonly timeStamp: string;
    readonly nonceStr: string;
    readonly package: string;
    readonly signType: 'RSA';
    readonly paySign: string;
  };
}

export type CheckoutResult =
  | QrcodeCheckoutResult
  | RedirectCheckoutResult
  | WechatJsapiCheckoutResult;

export interface VerifiedPaymentNotify {
  readonly channel: PaymentChannel;
  readonly orderNo: string;
  readonly channelTradeNo: string;
  readonly amountCents: number;
  readonly currency: string;
  readonly paidAt: string;
  readonly payerId: string;
  readonly rawNotifyId: string;
}

export interface NotifyHandleResult {
  readonly verified: boolean;
  readonly alreadyProcessed: boolean;
  readonly order?: PaymentOrderRecord;
  readonly notify?: VerifiedPaymentNotify;
  readonly channelResponseBody: string;
  readonly channelResponseContentType: string;
}

export interface QueryOrderResult {
  readonly order: PaymentOrderRecord;
  readonly channelPaid: boolean;
}

export interface RefundInput {
  readonly orderNo: string;
  readonly refundNo: string;
  readonly amountCents: number;
  readonly reason?: string;
}

export interface RefundResult {
  readonly refundNo: string;
  readonly orderNo: string;
  readonly status: 'success' | 'processing' | 'failed';
  readonly channelRefundNo: string;
  readonly message: string;
}

export interface PlatformDetectResult {
  readonly platform: ClientPlatform;
  readonly isWechatBrowser: boolean;
  readonly recommendedPresentation: PaymentPresentation;
  readonly channel: PaymentChannel | null;
}
