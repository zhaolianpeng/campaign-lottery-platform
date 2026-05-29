import type { ResolvedPaymentConfig } from '../config/schema.js';
import { channelDisabled } from '../utils/errors.js';
import type { PaymentChannel } from '../types.js';
import { createAlipayAdapter } from './alipay-adapter.js';
import { createWechatPayAdapter } from './wechat-pay-adapter.js';
import type { PaymentChannelAdapter } from './types.js';

export function createChannelRegistry(
  config: ResolvedPaymentConfig,
  configFilePath: string,
): Map<PaymentChannel, PaymentChannelAdapter> {
  const registry = new Map<PaymentChannel, PaymentChannelAdapter>();

  if (config.wechat?.enabled) {
    registry.set('wechat', createWechatPayAdapter(config, configFilePath));
  }

  if (config.alipay?.enabled) {
    registry.set('alipay', createAlipayAdapter(config, configFilePath));
  }

  return registry;
}

export function getChannelAdapter(
  registry: Map<PaymentChannel, PaymentChannelAdapter>,
  channel: PaymentChannel,
): PaymentChannelAdapter {
  const adapter = registry.get(channel);
  if (!adapter) {
    throw channelDisabled(channel);
  }
  return adapter;
}
