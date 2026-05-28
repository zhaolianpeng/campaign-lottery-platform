import { ApiRequestError } from '@/client/api';
import type { CEndFeatureToggles } from '@/types/api';
import { DEFAULT_C_END_FEATURE_TOGGLES } from './constants';

export const phonePattern = /^1[3-9]\d{9}$/;

export function normalizeCEndFeatureToggles(toggles?: Partial<CEndFeatureToggles>): CEndFeatureToggles {
  return {
    series: toggles?.series ?? DEFAULT_C_END_FEATURE_TOGGLES.series,
    inventory: toggles?.inventory ?? DEFAULT_C_END_FEATURE_TOGGLES.inventory,
    exchange: toggles?.exchange ?? DEFAULT_C_END_FEATURE_TOGGLES.exchange,
    rank: toggles?.rank ?? DEFAULT_C_END_FEATURE_TOGGLES.rank,
    member: toggles?.member ?? DEFAULT_C_END_FEATURE_TOGGLES.member,
    shop: toggles?.shop ?? DEFAULT_C_END_FEATURE_TOGGLES.shop,
    social: toggles?.social ?? DEFAULT_C_END_FEATURE_TOGGLES.social,
    puzzle: toggles?.puzzle ?? DEFAULT_C_END_FEATURE_TOGGLES.puzzle,
  };
}

export function mapDrawErrorMessage(error: unknown): Error {
  if (error instanceof ApiRequestError && error.code === 'no_draw_chances') {
    return new Error('当前活动今日抽奖次数已用完，请明天再试。');
  }
  if (error instanceof ApiRequestError && error.code === 'draw_phone_binding_required') {
    return new Error('当前盲盒要求先绑定手机号后才能抽取，请先完成手机号登录或绑定。');
  }
  return error instanceof Error ? error : new Error('抽奖失败，请稍后重试');
}

export function mapPurchaseErrorMessage(error: unknown): Error {
  if (error instanceof ApiRequestError && error.code === 'insufficient_points') {
    return new Error(error.message === 'insufficient points' ? '积分不足，无法完成购买' : error.message);
  }
  return error instanceof Error ? error : new Error('购买失败，请稍后重试');
}

export function anonymousDrawHeaders(token: string): HeadersInit | undefined {
  return token ? { 'X-Anonymous-Draw-Token': token } : undefined;
}

export function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return `${date.getMonth() + 1}/${date.getDate()} ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
}

export function getDeviceId(): string {
  if (typeof window === 'undefined') {
    return 'server';
  }
  let id = window.localStorage.getItem('campaign-lottery-device-id');
  if (!id) {
    id = `dev_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`;
    window.localStorage.setItem('campaign-lottery-device-id', id);
  }
  return id;
}

export function pointsByYuan(yuan: number): number {
  return Math.max(0, Math.round(yuan * 100));
}

export function levelScore(level: string): number {
  return { limited: 5, S: 5, secret: 4, A: 3, rare: 3, B: 2, common: 1 }[level] ?? 0;
}

export function drawGlowClass(level?: string): string {
  switch (level) {
    case 'limited':
    case 'S':
      return 'shadow-[0_0_48px_rgba(251,191,36,0.55)]';
    case 'secret':
      return 'shadow-[0_0_40px_rgba(167,139,250,0.5)]';
    case 'rare':
    case 'A':
      return 'shadow-[0_0_32px_rgba(56,189,248,0.45)]';
    default:
      return 'shadow-[0_0_24px_rgba(255,255,255,0.15)]';
  }
}
