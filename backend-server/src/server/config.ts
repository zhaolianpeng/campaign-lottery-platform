import { z } from 'zod';

const envSchema = z.object({
  ADMIN_USER: z.string().default('admin'),
  ADMIN_PASSWORD: z.string().default(''),
  CORS_ALLOW_ORIGIN: z.string().default('*'),

  MYSQL_ENABLED: z.coerce.boolean().default(false),
  MYSQL_DSN: z.string().optional(),
  MYSQL_HOST: z.string().default('127.0.0.1'),
  MYSQL_PORT: z.coerce.number().int().positive().default(3306),
  MYSQL_DATABASE: z.string().default('campaign_lottery_platform'),
  MYSQL_USER: z.string().default('campaign_lottery_app'),
  MYSQL_PASSWORD: z.string().default(''),
  MYSQL_CHARSET: z.string().default('utf8mb4'),
  MYSQL_CONNECTION_LIMIT: z.coerce.number().int().positive().default(10),

  REDIS_ENABLED: z.coerce.boolean().default(false),
  REDIS_URL: z.string().optional(),
  REDIS_HOST: z.string().default('127.0.0.1'),
  REDIS_PORT: z.coerce.number().int().positive().default(6379),
  REDIS_PASSWORD: z.string().optional(),
  REDIS_DATABASE: z.coerce.number().int().nonnegative().default(0),
  REDIS_KEY_PREFIX: z.string().default(''),

  WECHAT_APP_ID: z.string().default(''),
  WECHAT_APP_SECRET: z.string().default(''),
  WECHAT_TOKEN: z.string().default(''),
  WECHAT_REDIRECT_URI: z.string().default(''),
});

export interface AppConfig {
  readonly admin: {
    readonly user: string;
    readonly password: string;
  };
  readonly server: {
    readonly corsAllowOrigin: string;
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
    readonly appId: string;
    readonly appSecret: string;
    readonly token: string;
    readonly redirectUri: string;
  };
}

let cachedConfig: AppConfig | null = null;

export function getAppConfig(): AppConfig {
  if (cachedConfig) {
    return cachedConfig;
  }

  const env = envSchema.parse(process.env);
  cachedConfig = {
    admin: {
      user: env.ADMIN_USER,
      password: env.ADMIN_PASSWORD,
    },
    server: {
      corsAllowOrigin: env.CORS_ALLOW_ORIGIN,
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
      appId: env.WECHAT_APP_ID,
      appSecret: env.WECHAT_APP_SECRET,
      token: env.WECHAT_TOKEN,
      redirectUri: env.WECHAT_REDIRECT_URI,
    },
  };

  return cachedConfig;
}
