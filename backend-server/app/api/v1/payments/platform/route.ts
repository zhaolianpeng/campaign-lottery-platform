import { z } from 'zod';
import {
  detectClientPlatform,
  detectPlatformForChannel,
} from '@campaign-lottery/payment-module';
import { fail, ok } from '@/server/api-response';
import { paymentOptions, userAgentFromRequest } from '@/server/payment-http';
import { toAppError } from '@/server/payment-gateway';

export const dynamic = 'force-dynamic';

const querySchema = z.object({
  channel: z.enum(['wechat', 'alipay']).optional(),
});

export function OPTIONS(): Response {
  return paymentOptions();
}

/** 根据 User-Agent 返回推荐支付呈现方式（无需登录，不依赖支付是否启用） */
export async function GET(request: Request): Promise<Response> {
  try {
    const userAgent = userAgentFromRequest(request);
    const channel = querySchema.parse({
      channel: new URL(request.url).searchParams.get('channel') ?? undefined,
    }).channel;

    const platform = detectClientPlatform(userAgent);

    if (!channel) {
      return ok('client platform', {
        platform,
        wechat: detectPlatformForChannel(userAgent, 'wechat'),
        alipay: detectPlatformForChannel(userAgent, 'alipay'),
      });
    }

    return ok('client platform', detectPlatformForChannel(userAgent, channel));
  } catch (error) {
    return fail(toAppError(error));
  }
}

