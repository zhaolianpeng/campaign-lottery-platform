import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { AppError } from './errors';

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
    const moduleUrl = pathToFileURL(resolve(process.cwd(), '..', 'payment-module', 'dist', 'index.js')).href;
    cachedRuntime = import(moduleUrl) as Promise<PaymentRuntimeModule>;
  }
  return cachedRuntime;
}

export function isPaymentEnabled(): boolean {
  const flag = process.env.PAYMENT_ENABLED?.trim().toLowerCase() ?? '';
  return ['true', '1', 'yes', 'on'].includes(flag);
}

export function assertPaymentEnabled(): void {
  if (!isPaymentEnabled()) {
    throw new AppError('payment_disabled', '支付功能未启用', 503);
  }
}

export function getPaymentConfigPath(): string {
  const fromEnv = process.env.PAYMENT_CONFIG_PATH?.trim();
  if (fromEnv) {
    return fromEnv;
  }
  return resolve(process.cwd(), 'config/payment.config.json');
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
