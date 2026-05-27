import { bearerToken, corsHeaders, fail, ok } from './api-response';
import { unauthorized } from './errors';
import { getService } from './singleton';
import { assertPaymentEnabled, clientIpFromRequest, getPaymentModule, toAppError } from './payment-gateway';

export { corsHeaders };

export function paymentOptions(): Response {
  return new Response(null, { status: 204, headers: corsHeaders() });
}

export async function requireUserId(request: Request): Promise<string> {
  const token = bearerToken(request);
  if (!token) {
    throw unauthorized;
  }
  const service = await getService();
  const user = service.currentUser(token);
  return user.id;
}

export async function resolveWechatOpenid(request: Request): Promise<string | undefined> {
  const token = bearerToken(request);
  if (!token) {
    return undefined;
  }
  const service = await getService();
  const account = service.currentUserAccount(token);
  return account.wechat_openid;
}

export async function withPaymentApi<T>(
  handler: () => Promise<{ readonly message: string; readonly data: T; readonly status?: number }>,
): Promise<Response> {
  try {
    assertPaymentEnabled();
    const result = await handler();
    return ok(result.message, result.data, result.status ?? 200);
  } catch (error) {
    return fail(toAppError(error));
  }
}

export async function withPaymentNotify(
  channel: 'wechat' | 'alipay',
  request: Request,
): Promise<Response> {
  try {
    assertPaymentEnabled();
    const rawBody = await request.text();
    const headers: Record<string, string> = {};
    request.headers.forEach((value, key) => {
      headers[key] = value;
    });

    const payment = getPaymentModule();
    const result = await payment.handlePaymentNotify(channel, headers, rawBody);

    return new Response(result.channelResponseBody, {
      status: result.verified ? 200 : 400,
      headers: {
        ...corsHeaders(),
        'Content-Type': result.channelResponseContentType,
      },
    });
  } catch {
    const contentType = channel === 'wechat' ? 'application/json' : 'text/plain';
    const body =
      channel === 'wechat'
        ? JSON.stringify({ code: 'FAIL', message: '失败' })
        : 'failure';
    return new Response(body, {
      status: 400,
      headers: { ...corsHeaders(), 'Content-Type': contentType },
    });
  }
}

export function userAgentFromRequest(request: Request): string {
  return request.headers.get('user-agent') ?? '';
}

export { clientIpFromRequest };
