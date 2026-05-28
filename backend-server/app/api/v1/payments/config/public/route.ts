import { fail, ok } from '@/server/api-response';
import { paymentOptions } from '@/server/payment-http';
import { isPaymentEnabled, loadPaymentConfigFromRuntime, toAppError } from '@/server/payment-gateway';

export const dynamic = 'force-dynamic';

export function OPTIONS(): Response {
  return paymentOptions();
}

export async function GET(): Promise<Response> {
  try {
    let channels: { readonly wechat: boolean; readonly alipay: boolean } = {
      wechat: false,
      alipay: false,
    };

    if (isPaymentEnabled()) {
      try {
        const config = await loadPaymentConfigFromRuntime();
        channels = {
          wechat: Boolean(config.wechat?.enabled),
          alipay: Boolean(config.alipay?.enabled),
        };
      } catch {
        channels = { wechat: false, alipay: false };
      }
    }

    return ok('payment public config', {
      enabled: isPaymentEnabled(),
      channels,
    });
  } catch (error) {
    return fail(toAppError(error));
  }
}
