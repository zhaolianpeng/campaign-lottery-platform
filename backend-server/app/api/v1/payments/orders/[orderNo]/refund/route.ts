import { z } from 'zod';
import { bearerToken } from '@/server/api-response';
import { paymentOptions, requireUserId, withPaymentApi } from '@/server/payment-http';
import { getPaymentModule } from '@/server/payment-gateway';

export const dynamic = 'force-dynamic';

type RouteContext = {
  readonly params: { readonly orderNo: string } | Promise<{ readonly orderNo: string }>;
};

const refundSchema = z.object({
  reason: z.string().max(255).optional(),
  amount_cents: z.number().int().positive().optional(),
});

async function orderNo(context: RouteContext): Promise<string> {
  const params = await context.params;
  return params.orderNo;
}

export function OPTIONS(): Response {
  return paymentOptions();
}

export async function POST(request: Request, context: RouteContext): Promise<Response> {
  return withPaymentApi(async () => {
    const userId = await requireUserId(request);
    const no = await orderNo(context);
    const input = refundSchema.parse(await request.json().catch(() => ({})));
    const payment = await getPaymentModule();
    const order = payment.getOrder(no);

    if (order.userId !== userId) {
      throw new Error('forbidden');
    }

    const refund = await payment.requestRefund({
      orderNo: no,
      reason: input.reason ?? 'user requested refund',
      amountCents: input.amount_cents,
    });

    return { message: 'refund requested', data: refund };
  });
}

export async function GET(_request: Request, context: RouteContext): Promise<Response> {
  return withPaymentApi(async () => {
    const token = bearerToken(_request);
    const service = await import('@/server/singleton').then((m) => m.getService());
    const user = (await service).currentUser(token);
    const { getRepositories } = await import('@/server/repositories');
    const orders = await getRepositories().paymentOrders.listByUser(user.id);
    return { message: 'payment orders', data: orders };
  });
}
