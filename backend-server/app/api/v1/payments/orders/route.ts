import { z } from 'zod';
import {
  clientIpFromRequest,
  paymentOptions,
  requireUserId,
  resolveWechatOpenid,
  userAgentFromRequest,
  withPaymentApi,
} from '@/server/payment-http';
import { getPaymentModule } from '@/server/payment-gateway';

export const dynamic = 'force-dynamic';

const createCheckoutSchema = z.object({
  client_request_id: z.string().min(1).max(64),
  channel: z.enum(['wechat', 'alipay']),
  amount_cents: z.number().int().positive(),
  subject: z.string().min(1).max(128),
  body: z.string().max(255).optional(),
  business_type: z.string().min(1).max(32),
  business_id: z.string().max(64).optional(),
  product_snapshot: z.record(z.string(), z.unknown()).optional(),
  presentation_override: z.enum(['qrcode', 'redirect_h5', 'wechat_jsapi']).optional(),
});

export function OPTIONS(): Response {
  return paymentOptions();
}

export async function POST(request: Request): Promise<Response> {
  return withPaymentApi(async () => {
    const userId = await requireUserId(request);
    const input = createCheckoutSchema.parse(await request.json());
    const payment = getPaymentModule();

    const checkout = await payment.createCheckout({
      userId,
      clientRequestId: input.client_request_id,
      channel: input.channel,
      amountCents: input.amount_cents,
      subject: input.subject,
      body: input.body,
      businessType: input.business_type,
      businessId: input.business_id,
      productSnapshot: input.product_snapshot,
      userAgent: userAgentFromRequest(request),
      clientIp: clientIpFromRequest(request),
      wechatOpenid: await resolveWechatOpenid(request),
      presentationOverride: input.presentation_override,
    });

    return { message: 'checkout created', data: checkout };
  });
}
