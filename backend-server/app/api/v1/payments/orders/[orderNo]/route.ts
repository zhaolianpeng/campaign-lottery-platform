import { paymentOptions, requireUserId, withPaymentApi } from '@/server/payment-http';
import { getPaymentModule } from '@/server/payment-gateway';
import { AppError } from '@/server/errors';
import { tryFulfillPaidOrder } from '@/server/payment-fulfillment';

export const dynamic = 'force-dynamic';

type RouteContext = {
  readonly params: { readonly orderNo: string } | Promise<{ readonly orderNo: string }>;
};

async function orderNo(context: RouteContext): Promise<string> {
  const params = await context.params;
  return params.orderNo;
}

export function OPTIONS(): Response {
  return paymentOptions();
}

export async function GET(request: Request, context: RouteContext): Promise<Response> {
  return withPaymentApi(async () => {
    const userId = await requireUserId(request);
    const no = await orderNo(context);
    const syncChannel = new URL(request.url).searchParams.get('sync_channel') === 'true';

    const payment = await getPaymentModule();
    const { order, channelPaid } = await payment.queryOrder(no, { syncChannel });

    if (order.userId !== userId) {
      throw new AppError('forbidden', '无权查看该订单', 403);
    }

    if (syncChannel && (order.status === 'paid' || order.status === 'fulfilling')) {
      await tryFulfillPaidOrder(no);
    }

    const latest = payment.getOrder(no);

    return {
      message: 'payment order',
      data: { order: latest, channel_paid: channelPaid },
    };
  });
}
