import { Home, Medal, PackageOpen, Puzzle, RefreshCw, ShoppingBag, UserRound, Users } from 'lucide-react';
import type { ComponentType } from 'react';
import type { CEndFeatureToggles, TabKey } from '@/types/api';

export const ANONYMOUS_DRAW_TOKEN_KEY = 'campaign-lottery-anonymous-draw-token';
export const USER_TOKEN_KEY = 'campaign-lottery-user-token';
export const USER_NICKNAME_KEY = 'campaign-lottery-user-nickname';
export const DEVICE_ID_KEY = 'campaign-lottery-device-id';
export const INVITE_FROM_KEY = 'campaign-lottery-invite-from';

export const MONTHLY_CARD_CASH_CENTS = 2800;
export const MONTHLY_CARD_POINTS = 2800;
export const BATTLE_PASS_CASH_CENTS = 6800;
export const BATTLE_PASS_POINTS = 680;
export const POINTS_PER_YUAN = 100;
export const POINTS_RECHARGE_FIXED_YUAN_SKUS = [1, 6, 12, 30, 68] as const;

export interface TabItem {
  readonly key: TabKey;
  readonly label: string;
  readonly icon: ComponentType<{ readonly size?: number; readonly className?: string }>;
}

export const LOTTERY_TABS: readonly TabItem[] = [
  { key: 'series', label: '系列', icon: Home },
  { key: 'inventory', label: '我的', icon: PackageOpen },
  { key: 'exchange', label: '交换', icon: RefreshCw },
  { key: 'rank', label: '排行', icon: Medal },
  { key: 'member', label: '会员', icon: UserRound },
  { key: 'shop', label: '商店', icon: ShoppingBag },
  { key: 'social', label: '社交', icon: Users },
  { key: 'puzzle', label: '拼图', icon: Puzzle },
];

export const DEFAULT_C_END_FEATURE_TOGGLES: CEndFeatureToggles = {
  series: true,
  inventory: true,
  exchange: true,
  rank: true,
  member: true,
  shop: true,
  social: true,
  puzzle: true,
};

export const MEMBER_LEVEL_BENEFITS: readonly {
  readonly level: string;
  readonly label: string;
  readonly threshold: number;
  readonly perks: string;
}[] = [
  { level: 'normal', label: '青铜', threshold: 0, perks: '基础抽盒、每日签到' },
  { level: 'silver', label: '白银', threshold: 500, perks: '合成成功率+5%、抢购资格' },
  { level: 'gold', label: '黄金', threshold: 2000, perks: '九折券/月、交换手续费8折' },
  { level: 'diamond', label: '钻石', threshold: 15000, perks: '限定直购、专属标识' },
];
