import { bearerToken } from '@/server/api-response';
import { getPaymentFulfillmentStatus } from '@/server/payment-fulfillment';
import { paymentOptions, withPaymentApi } from '@/server/payment-http';

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

/** User-facing fulfill endpoint is read-only; fulfillment runs via payment notify / internal jobs. */
export async function GET(request: Request, context: RouteContext): Promise<Response> {
  return withPaymentApi(async () => {
    const token = bearerToken(request);
    const no = await orderNo(context);
    const result = await getPaymentFulfillmentStatus(token, no);
    return { message: 'payment order status', data: result };
  });
}

export async function POST(request: Request, context: RouteContext): Promise<Response> {
  return GET(request, context);
}
