import { z } from 'zod';

const envBoolean = z.preprocess((value) => {
  if (typeof value !== 'string') {
    return value;
  }

  const normalized = value.trim().toLowerCase();
  if (['true', '1', 'yes', 'on'].includes(normalized)) {
    return true;
  }
  if (['false', '0', 'no', 'off', ''].includes(normalized)) {
    return false;
  }

  return value;
}, z.boolean());

const envSchema = z
  .object({
    ADMIN_USER: z.string().default('admin'),
    ADMIN_PASSWORD: z.string().default(''),
    CORS_ALLOW_ORIGIN: z.string().default('*'),
    STORAGE_MODE: z.enum(['mysql', 'memory']).optional(),

    MYSQL_ENABLED: envBoolean.default(false),
    MYSQL_DSN: z.string().optional(),
    MYSQL_HOST: z.string().default('127.0.0.1'),
    MYSQL_PORT: z.coerce.number().int().positive().default(3306),
    MYSQL_DATABASE: z.string().default('campaign_lottery_platform'),
    MYSQL_USER: z.string().default('campaign_lottery_app'),
    MYSQL_PASSWORD: z.string().default(''),
    MYSQL_CHARSET: z.string().default('utf8mb4'),
    MYSQL_CONNECTION_LIMIT: z.coerce.number().int().positive().default(10),

    REDIS_ENABLED: envBoolean.default(false),
    REDIS_URL: z.string().optional(),
    REDIS_HOST: z.string().default('127.0.0.1'),
    REDIS_PORT: z.coerce.number().int().positive().default(6379),
    REDIS_PASSWORD: z.string().optional(),
    REDIS_DATABASE: z.coerce.number().int().nonnegative().default(0),
    REDIS_KEY_PREFIX: z.string().default(''),

    WECHAT_QUICK_LOGIN_ENABLED: envBoolean.default(false),
    WECHAT_APP_ID: z.string().default(''),
    WECHAT_APP_SECRET: z.string().default(''),
    WECHAT_TOKEN: z.string().default(''),
    WECHAT_REDIRECT_URI: z.string().default(''),

    SMS_PROVIDER: z.string().default('mock'),
    SMS_ACCESS_KEY_ID: z.string().default(''),
    SMS_ACCESS_KEY_SECRET: z.string().default(''),
    SMS_SIGN_NAME: z.string().default(''),
    SMS_TEMPLATE_CODE: z.string().default(''),
    CARRIER_AUTH_PROVIDER: z.string().default(''),
    CARRIER_AUTH_APP_ID: z.string().default(''),
    CARRIER_AUTH_API_KEY: z.string().default(''),

    PAYMENT_ENABLED: envBoolean.default(false),
    PAYMENT_CONFIG_PATH: z.string().default('config/payment.config.json'),

    GUEST_INITIAL_POINTS: z.coerce.number().int().nonnegative().default(100),
  })
  .superRefine((data, ctx) => {
    const nodeEnv = process.env.NODE_ENV ?? 'development';
    if (nodeEnv === 'production' && data.CORS_ALLOW_ORIGIN.trim() === '*') {
      ctx.addIssue({
        code: 'custom',
        message: 'CORS_ALLOW_ORIGIN cannot be * in production',
        path: ['CORS_ALLOW_ORIGIN'],
      });
    }
  });

export type StorageMode = 'mysql' | 'memory';

export interface AppConfig {
  readonly admin: {
    readonly user: string;
    readonly password: string;
  };
  readonly server: {
    readonly corsAllowOrigin: string;
    readonly storageMode: StorageMode;
  };
  readonly mysql: {
    readonly enabled: boolean;
    readonly dsn?: string;
    readonly host: string;
    readonly port: number;
    readonly database: string;
    readonly user: string;
    readonly password: string;
    readonly charset: string;
    readonly connectionLimit: number;
  };
  readonly redis: {
    readonly enabled: boolean;
    readonly url?: string;
    readonly host: string;
    readonly port: number;
    readonly password?: string;
    readonly database: number;
    readonly keyPrefix: string;
  };
  readonly wechat: {
    readonly quickLoginEnabled: boolean;
    readonly appId: string;
    readonly appSecret: string;
    readonly token: string;
    readonly redirectUri: string;
  };
  readonly sms: {
    readonly provider: string;
    readonly accessKeyId: string;
    readonly accessKeySecret: string;
    readonly signName: string;
    readonly templateCode: string;
  };
  readonly carrierAuth: {
    readonly provider: string;
    readonly appId: string;
    readonly apiKey: string;
  };
  readonly payment: {
    readonly enabled: boolean;
    readonly configPath: string;
  };
  readonly guest: {
    readonly initialPoints: number;
  };
}

let cachedConfig: AppConfig | null = null;

function resolveStorageMode(env: z.infer<typeof envSchema>): StorageMode {
  if (env.STORAGE_MODE) {
    return env.STORAGE_MODE;
  }
  return env.MYSQL_ENABLED ? 'mysql' : 'memory';
}

export function getAppConfig(): AppConfig {
  if (cachedConfig) {
    return cachedConfig;
  }

  const env = envSchema.parse(process.env);
  const storageMode = resolveStorageMode(env);
  if (storageMode === 'mysql' && !env.MYSQL_ENABLED) {
    throw new Error('STORAGE_MODE=mysql requires MYSQL_ENABLED=true');
  }

  cachedConfig = {
    admin: {
      user: env.ADMIN_USER,
      password: env.ADMIN_PASSWORD,
    },
    server: {
      corsAllowOrigin: env.CORS_ALLOW_ORIGIN,
      storageMode,
    },
    mysql: {
      enabled: env.MYSQL_ENABLED,
      dsn: env.MYSQL_DSN,
      host: env.MYSQL_HOST,
      port: env.MYSQL_PORT,
      database: env.MYSQL_DATABASE,
      user: env.MYSQL_USER,
      password: env.MYSQL_PASSWORD,
      charset: env.MYSQL_CHARSET,
      connectionLimit: env.MYSQL_CONNECTION_LIMIT,
    },
    redis: {
      enabled: env.REDIS_ENABLED,
      url: env.REDIS_URL,
      host: env.REDIS_HOST,
      port: env.REDIS_PORT,
      password: env.REDIS_PASSWORD || undefined,
      database: env.REDIS_DATABASE,
      keyPrefix: env.REDIS_KEY_PREFIX,
    },
    wechat: {
      quickLoginEnabled: env.WECHAT_QUICK_LOGIN_ENABLED,
      appId: env.WECHAT_APP_ID,
      appSecret: env.WECHAT_APP_SECRET,
      token: env.WECHAT_TOKEN,
      redirectUri: env.WECHAT_REDIRECT_URI,
    },
    sms: {
      provider: env.SMS_PROVIDER,
      accessKeyId: env.SMS_ACCESS_KEY_ID,
      accessKeySecret: env.SMS_ACCESS_KEY_SECRET,
      signName: env.SMS_SIGN_NAME,
      templateCode: env.SMS_TEMPLATE_CODE,
    },
    carrierAuth: {
      provider: env.CARRIER_AUTH_PROVIDER,
      appId: env.CARRIER_AUTH_APP_ID,
      apiKey: env.CARRIER_AUTH_API_KEY,
    },
    payment: {
      enabled: env.PAYMENT_ENABLED,
      configPath: env.PAYMENT_CONFIG_PATH,
    },
    guest: {
      initialPoints: env.GUEST_INITIAL_POINTS,
    },
  };

  return cachedConfig;
}

export function resetAppConfigForTests(): void {
  cachedConfig = null;
}
