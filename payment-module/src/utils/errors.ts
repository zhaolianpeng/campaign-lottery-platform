export class PaymentModuleError extends Error {
  readonly code: string;
  readonly httpStatus: number;

  constructor(code: string, message: string, httpStatus = 400) {
    super(message);
    this.name = 'PaymentModuleError';
    this.code = code;
    this.httpStatus = httpStatus;
  }
}

export const invalidConfig = (message: string) =>
  new PaymentModuleError('invalid_config', message, 500);

export const channelDisabled = (channel: string) =>
  new PaymentModuleError('channel_disabled', `支付渠道未启用: ${channel}`, 400);

export const orderNotFound = (orderNo: string) =>
  new PaymentModuleError('order_not_found', `订单不存在: ${orderNo}`, 404);

export const invalidOrderState = (message: string) =>
  new PaymentModuleError('invalid_order_state', message, 409);

export const presentationNotSupported = (message: string) =>
  new PaymentModuleError('presentation_not_supported', message, 400);

export const notifyVerifyFailed = (message: string) =>
  new PaymentModuleError('notify_verify_failed', message, 400);

export const channelApiError = (message: string) =>
  new PaymentModuleError('channel_api_error', message, 502);
