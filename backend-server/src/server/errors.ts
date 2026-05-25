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
