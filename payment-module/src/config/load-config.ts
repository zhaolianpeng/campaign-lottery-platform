import { readFileSync } from 'node:fs';
import { isAbsolute, resolve } from 'node:path';
import { paymentConfigSchema, type PaymentConfig, type ResolvedPaymentConfig } from './schema.js';

export function loadPaymentConfig(configPath: string): ResolvedPaymentConfig {
  const absolutePath = isAbsolute(configPath) ? configPath : resolve(process.cwd(), configPath);
  const raw = JSON.parse(readFileSync(absolutePath, 'utf8')) as unknown;
  const config = paymentConfigSchema.parse(raw) satisfies PaymentConfig;

  const wechatNotifyUrl = config.wechat
    ? joinUrl(config.notifyBaseUrl, config.wechat.notifyPath)
    : '';
  const alipayNotifyUrl = config.alipay
    ? joinUrl(config.notifyBaseUrl, config.alipay.notifyPath)
    : '';

  return {
    ...config,
    wechatNotifyUrl,
    alipayNotifyUrl,
  };
}

export function readPemFile(filePath: string, configDir: string): string {
  const absolutePath = isAbsolute(filePath) ? filePath : resolve(configDir, filePath);
  return readFileSync(absolutePath, 'utf8');
}

function joinUrl(base: string, path: string): string {
  const normalizedBase = base.endsWith('/') ? base.slice(0, -1) : base;
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${normalizedBase}${normalizedPath}`;
}
