import { resolve } from 'node:path';
import {
  createPaymentModule,
  PaymentModuleError,
  type PaymentModule,
} from '@campaign-lottery/payment-module';
import { AppError } from './errors';

let cachedModule: PaymentModule | null = null;

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

export function getPaymentModule(): PaymentModule {
  assertPaymentEnabled();
  if (!cachedModule) {
    cachedModule = createPaymentModule({ configPath: getPaymentConfigPath() });
  }
  return cachedModule;
}

export function resetPaymentModuleForTests(): void {
  cachedModule = null;
}

export function toAppError(error: unknown): AppError {
  if (error instanceof PaymentModuleError) {
    return new AppError(error.code, error.message, error.httpStatus);
  }
  if (error instanceof AppError) {
    return error;
  }
  if (error instanceof Error) {
    return new AppError('internal_error', error.message, 500);
  }
  return new AppError('internal_error', 'internal server error', 500);
}

export function clientIpFromRequest(request: Request): string {
  const forwarded = request.headers.get('x-forwarded-for');
  if (forwarded) {
    return forwarded.split(',')[0]?.trim() ?? '127.0.0.1';
  }
  return request.headers.get('x-real-ip')?.trim() ?? '127.0.0.1';
}
