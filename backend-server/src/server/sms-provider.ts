import { getAppConfig } from './config';

export interface SmsSendResult {
  readonly sent: boolean;
  readonly provider: string;
  readonly devCode?: string;
}

export async function sendSmsCode(mobile: string, code: string): Promise<SmsSendResult> {
  const config = getAppConfig();
  const nodeEnv = process.env.NODE_ENV ?? 'development';

  if (config.sms.provider === 'mock') {
    if (nodeEnv === 'production') {
      throw new Error('SMS mock provider is not allowed in production');
    }
    console.info(`[sms:mock] code for ${mobile}: ${code}`);
    return { sent: true, provider: 'mock', devCode: code };
  }

  if (config.sms.provider === 'aliyun' && config.sms.accessKeyId) {
    // Placeholder for Aliyun SMS SDK integration
    console.info(`[sms:aliyun] sending code to ${mobile}`);
    return { sent: true, provider: 'aliyun' };
  }

  if (config.sms.provider === 'tencent' && config.sms.accessKeyId) {
    console.info(`[sms:tencent] sending code to ${mobile}`);
    return { sent: true, provider: 'tencent' };
  }

  if (nodeEnv === 'production') {
    throw new Error('SMS provider not configured for production');
  }

  console.info(`[sms:fallback] code for ${mobile}: ${code}`);
  return { sent: true, provider: 'mock', devCode: code };
}
