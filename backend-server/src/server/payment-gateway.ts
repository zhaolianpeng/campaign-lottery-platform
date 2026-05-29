import { readFileSync } from 'node:fs';
import { isAbsolute, resolve } from 'node:path';
import { AppError } from './errors';
import { getAppConfig } from './config';

let cachedModule: PaymentModule | null = null;
let cachedRuntime: Promise<PaymentRuntimeModule> | null = null;

type PaymentChannel = 'wechat' | 'alipay';
type ClientPlatform = 'mobile' | 'desktop';
type PaymentPresentation = 'qrcode' | 'redirect_h5' | 'wechat_jsapi';

type PlatformDetectResult = {
  readonly channel: PaymentChannel;
  readonly isWechatBrowser: boolean;
  readonly platform: ClientPlatform;
  readonly recommendedPresentation: PaymentPresentation;
};

type PaymentRuntimeModule = {
  readonly createPaymentModule: (options?: { readonly configPath?: string }) => PaymentModule;
  readonly loadPaymentConfig: (configPath: string) => ResolvedPaymentConfig;
};

type ResolvedPaymentConfig = {
  readonly mock?: boolean;
  readonly alipay?: { readonly enabled?: boolean } | null;
  readonly wechat?: { readonly enabled?: boolean } | null;
};

type PaymentModule = {
  readonly beginFulfillment: (orderNo: string) => unknown;
  readonly completeFulfillment: (orderNo: string) => unknown;
  readonly createCheckout: (input: Record<string, unknown>) => Promise<unknown>;
  readonly getOrder: (orderNo: string) => any;
  readonly handlePaymentNotify: (
    channel: PaymentChannel,
    headers: Record<string, string>,
    rawBody: string,
  ) => Promise<{
    readonly channelResponseBody: string;
    readonly channelResponseContentType: string;
    readonly verified: boolean;
  }>;
  readonly queryOrder: (orderNo: string, options?: { readonly syncChannel?: boolean }) => Promise<{ readonly order: any; readonly channelPaid: boolean }>;
  readonly requestRefund: (input: Record<string, unknown>) => Promise<unknown>;
};

const MOBILE_UA_PATTERN = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini|Mobile|mobile/i;
const WECHAT_UA_PATTERN = /MicroMessenger/i;

async function loadPaymentRuntime(): Promise<PaymentRuntimeModule> {
  if (!cachedRuntime) {
    cachedRuntime = import(
      /* webpackIgnore: true */
      /* turbopackIgnore: true */
      '@campaign-lottery/payment-module',
    ).then((module) => module as unknown as PaymentRuntimeModule);
  }
  return cachedRuntime;
}

export function isPaymentEnabled(): boolean {
  return getAppConfig().payment.enabled;
}

export function assertPaymentEnabled(): void {
  if (!isPaymentEnabled()) {
    throw new AppError('payment_disabled', '支付功能未启用', 503);
  }
}

export function getPaymentConfigPath(): string {
  const fromConfig = getAppConfig().payment.configPath.trim();
  if (!fromConfig) {
    return resolve(process.cwd(), 'config/payment.config.json');
  }
  return isAbsolute(fromConfig) ? fromConfig : resolve(process.cwd(), fromConfig);
}

export async function getPaymentModule(): Promise<PaymentModule> {
  assertPaymentEnabled();
  if (!cachedModule) {
    const runtime = await loadPaymentRuntime();
    cachedModule = runtime.createPaymentModule({ configPath: getPaymentConfigPath() });
  }
  return cachedModule;
}

export async function loadPaymentConfigFromRuntime(): Promise<ResolvedPaymentConfig> {
  const runtime = await loadPaymentRuntime();
  return runtime.loadPaymentConfig(getPaymentConfigPath());
}

/** 公开探测接口用：不依赖 payment-module 动态加载，直接读 JSON */
export function readPaymentConfigFlagsFromFile(): {
  readonly channels: { readonly wechat: boolean; readonly alipay: boolean };
  readonly mock: boolean;
} | null {
  try {
    const raw = JSON.parse(readFileSync(getPaymentConfigPath(), 'utf8')) as {
      readonly mock?: boolean;
      readonly wechat?: { readonly enabled?: boolean };
      readonly alipay?: { readonly enabled?: boolean };
    };
    return {
      channels: {
        wechat: Boolean(raw.wechat?.enabled),
        alipay: Boolean(raw.alipay?.enabled),
      },
      mock: Boolean(raw.mock),
    };
  } catch {
    return null;
  }
}

export function resetPaymentModuleForTests(): void {
  cachedModule = null;
  cachedRuntime = null;
}

export function toAppError(error: unknown): AppError {
  if (error instanceof AppError) {
    return error;
  }
  if (
    error &&
    typeof error === 'object' &&
    'code' in error &&
    'httpStatus' in error &&
    'message' in error &&
    typeof error.code === 'string' &&
    typeof error.httpStatus === 'number' &&
    typeof error.message === 'string'
  ) {
    return new AppError(error.code, error.message, error.httpStatus);
  }
  if (error instanceof Error) {
    return new AppError('internal_error', error.message, 500);
  }
  return new AppError('internal_error', 'internal server error', 500);
}

export function isWechatBrowser(userAgent: string): boolean {
  return WECHAT_UA_PATTERN.test(userAgent);
}

export function detectClientPlatform(userAgent: string): ClientPlatform {
  return MOBILE_UA_PATTERN.test(userAgent) ? 'mobile' : 'desktop';
}

export function detectPlatformForChannel(userAgent: string, channel: PaymentChannel): PlatformDetectResult {
  const platform = detectClientPlatform(userAgent);
  const inWechatBrowser = isWechatBrowser(userAgent);
  const recommendedPresentation: PaymentPresentation =
    platform === 'mobile'
      ? (channel === 'wechat' && inWechatBrowser ? 'wechat_jsapi' : 'redirect_h5')
      : 'qrcode';

  return {
    channel,
    isWechatBrowser: inWechatBrowser,
    platform,
    recommendedPresentation,
  };
}

export function clientIpFromRequest(request: Request): string {
  const forwarded = request.headers.get('x-forwarded-for');
  if (forwarded) {
    return forwarded.split(',')[0]?.trim() ?? '127.0.0.1';
  }
  return request.headers.get('x-real-ip')?.trim() ?? '127.0.0.1';
}
