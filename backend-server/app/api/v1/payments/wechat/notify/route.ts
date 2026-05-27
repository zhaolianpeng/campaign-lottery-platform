import { paymentOptions, withPaymentNotify } from '@/server/payment-http';

export const dynamic = 'force-dynamic';

export function OPTIONS(): Response {
  return paymentOptions();
}

export async function POST(request: Request): Promise<Response> {
  return withPaymentNotify('wechat', request);
}
