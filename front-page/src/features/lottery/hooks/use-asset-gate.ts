import type { UserStatus } from '@/types/api';

export interface AssetGateResult {
  readonly canUseAssets: boolean;
  readonly blockedReason: string | null;
}

export function useAssetGate(status?: UserStatus, hasToken = false): AssetGateResult {
  if (!hasToken) {
    return { canUseAssets: false, blockedReason: '请先登录' };
  }
  if (status === 'frozen') {
    return { canUseAssets: false, blockedReason: '账号已冻结，资产操作已禁用' };
  }
  if (status === 'disabled') {
    return { canUseAssets: false, blockedReason: '账号已禁用' };
  }
  if (status === 'cancelled') {
    return { canUseAssets: false, blockedReason: '账号已注销' };
  }
  if (status === 'pending_phone') {
    return { canUseAssets: false, blockedReason: '请先完成手机号验证' };
  }
  return { canUseAssets: true, blockedReason: null };
}
