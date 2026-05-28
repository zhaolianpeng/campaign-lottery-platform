export type PaymentChannel = 'wechat' | 'alipay';

export type PaymentPresentation = 'qrcode' | 'redirect_h5' | 'wechat_jsapi';

export interface PaymentPublicConfig {
  readonly enabled: boolean;
  readonly channels: {
    readonly wechat: boolean;
    readonly alipay: boolean;
  };
  readonly mock: boolean;
}

export interface CheckoutBase {
  readonly order_no: string;
  readonly amount_cents: number;
  readonly currency: string;
  readonly expire_at: string;
  readonly platform: 'mobile' | 'desktop';
  readonly channel: PaymentChannel;
}

export interface QrcodeCheckout extends CheckoutBase {
  readonly presentation: 'qrcode';
  readonly qr_code_content: string;
}

export interface RedirectCheckout extends CheckoutBase {
  readonly presentation: 'redirect_h5';
  readonly redirect_url: string;
}

export interface WechatJsapiCheckout extends CheckoutBase {
  readonly presentation: 'wechat_jsapi';
  readonly channel: 'wechat';
  readonly jsapi_params: {
    readonly appId: string;
    readonly timeStamp: string;
    readonly nonceStr: string;
    readonly package: string;
    readonly signType: 'RSA';
    readonly paySign: string;
  };
}

export type CheckoutResult = QrcodeCheckout | RedirectCheckout | WechatJsapiCheckout;

export interface PaymentOrderSnapshot {
  readonly orderNo: string;
  readonly userId: string;
  readonly channel: PaymentChannel;
  readonly businessType: string;
  readonly businessId: string;
  readonly amountCents: number;
  readonly status: string;
  readonly presentation?: string;
}

export interface QueryPaymentOrderResult {
  readonly order: PaymentOrderSnapshot;
  readonly channel_paid: boolean;
}

export interface CreateCheckoutInput {
  readonly client_request_id: string;
  readonly channel: PaymentChannel;
  readonly amount_cents: number;
  readonly subject: string;
  readonly body?: string;
  readonly business_type: string;
  readonly business_id?: string;
  readonly product_snapshot?: Record<string, unknown>;
}
