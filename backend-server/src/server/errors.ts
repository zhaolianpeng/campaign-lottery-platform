export class AppError extends Error {
  public readonly status: number;
  public readonly code: string;

  public constructor(code: string, message: string, status = 400) {
    super(message);
    this.name = 'AppError';
    this.code = code;
    this.status = status;
  }
}

export const unauthorized = new AppError('unauthorized', 'unauthorized', 401);
export const adminUnauthorized = new AppError('admin_unauthorized', 'admin unauthorized', 401);
export const campaignNotFound = new AppError('campaign_not_found', 'campaign not found', 404);
export const campaignInactive = new AppError('campaign_inactive', 'campaign inactive', 409);
export const noDrawChances = new AppError('no_draw_chances', 'no draw chances', 409);
export const insufficientPoints = new AppError('insufficient_points', 'insufficient points', 409);
export const badAdminAuth = new AppError('bad_admin_credentials', 'bad admin credentials', 401);
export const notFound = new AppError('not_found', 'not found', 404);
export const wechatAuthFailed = new AppError('wechat_auth_failed', '微信授权失败', 401);
export const wechatPhoneRequired = new AppError('wechat_phone_required', '需要获取手机号', 403);
export const phoneAlreadyBound = new AppError('phone_already_bound', '手机号已绑定其他账号', 409);
export const phoneCodeInvalid = new AppError('phone_code_invalid', '验证码错误或已过期', 400);
export const phoneVerificationRequired = new AppError('phone_verification_required', '请使用短信验证码登录', 403);
export const userStatusForbidden = new AppError('user_status_forbidden', '当前账号状态不允许执行该操作', 403);
