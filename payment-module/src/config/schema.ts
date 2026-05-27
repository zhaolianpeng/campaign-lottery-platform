import { z } from 'zod';

const channelBaseSchema = z.object({
  enabled: z.boolean().default(true),
  notifyPath: z.string().min(1),
});

export const paymentConfigSchema = z.object({
  notifyBaseUrl: z.string().url(),
  orderExpireMinutes: z.number().int().positive().default(30),
  mock: z.boolean().default(false),
  wechat: channelBaseSchema
    .extend({
      appId: z.string().min(1),
      mchId: z.string().min(1),
      apiV3Key: z.string().min(16),
      serialNo: z.string().min(1),
      privateKeyPath: z.string().min(1),
      platformCertPath: z.string().min(1),
      h5AppName: z.string().default('Payment'),
      h5AppUrl: z.string().url(),
    })
    .optional(),
  alipay: channelBaseSchema
    .extend({
      appId: z.string().min(1),
      privateKeyPath: z.string().min(1),
      alipayPublicKeyPath: z.string().min(1),
      gateway: z.string().url(),
      signType: z.enum(['RSA2']).default('RSA2'),
      returnUrl: z.string().url().optional(),
    })
    .optional(),
});

export type PaymentConfig = z.infer<typeof paymentConfigSchema>;

export interface ResolvedPaymentConfig extends PaymentConfig {
  readonly wechatNotifyUrl: string;
  readonly alipayNotifyUrl: string;
}
