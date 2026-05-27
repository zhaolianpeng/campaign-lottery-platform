export { createPaymentModule } from './core/payment-service.js';
export type { PaymentModule, PaymentModuleOptions } from './core/payment-service.js';

export { loadPaymentConfig, readPemFile } from './config/load-config.js';
export { paymentConfigSchema } from './config/schema.js';
export type { PaymentConfig, ResolvedPaymentConfig } from './config/schema.js';

export {
  detectClientPlatform,
  detectPlatformForChannel,
  isMobileUserAgent,
  isWechatBrowser,
  resolvePaymentPresentation,
} from './core/client-platform.js';

export { MemoryOrderStore } from './store/memory-order-store.js';
export type { CreateOrderParams } from './store/memory-order-store.js';

export { PaymentModuleError } from './utils/errors.js';

export type {
  CheckoutResult,
  ClientPlatform,
  CreateCheckoutInput,
  NotifyHandleResult,
  PaymentChannel,
  PaymentOrderRecord,
  PaymentOrderStatus,
  PaymentPresentation,
  PlatformDetectResult,
  QrcodeCheckoutResult,
  RedirectCheckoutResult,
  RefundInput,
  RefundResult,
  QueryOrderResult,
  VerifiedPaymentNotify,
  WechatJsapiCheckoutResult,
} from './types.js';
