import { bearerToken } from '@/server/api-response';
import { paymentOptions, withPaymentApi } from '@/server/payment-http';
import { fulfillPaymentOrder } from '@/server/payment-fulfillment';

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

export async function POST(request: Request, context: RouteContext): Promise<Response> {
  return withPaymentApi(async () => {
    const token = bearerToken(request);
    const no = await orderNo(context);
    const result = await fulfillPaymentOrder(token, no);
    return { message: 'payment fulfilled', data: result };
  });
}
