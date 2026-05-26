'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Gift,
  Home,
  Loader2,
  Medal,
  PackageOpen,
  Puzzle,
  RefreshCw,
  Share2,
  ShoppingBag,
  Sparkles,
  Trophy,
  UserRound,
  Users,
  X,
} from 'lucide-react';
import { useMemo, useState, type ComponentType } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { apiRequest } from '@/client/api';
import type {
  BlindBoxDrawResult,
  ActivityListInfo,
  AssistProgress,
  BattlePassInfo,
  CampaignListItem,
  ExchangeOffer,
  FirstRechargePack,
  FlashListInfo,
  GiftRecord,
  InviteStats,
  LeaderboardEntry,
  MonthCardStatus,
  PuzzleInfo,
  Prize,
  SeriesProgress,
  ShopItem,
  TeamInfo,
  UserInventory,
  UserItem,
  UserFirstRecharge,
  UserAccount,
  UserMember,
  UserPointsLog,
} from '@/types/api';

const loginSchema = z.object({
  nickname: z.string().max(32).optional(),
});

const exchangeSchema = z.object({
  have_prize_id: z.string().min(1),
  want_prize_id: z.string().min(1),
});

type LoginFormValues = z.infer<typeof loginSchema>;
type ExchangeFormValues = z.infer<typeof exchangeSchema>;
type TabKey = 'series' | 'inventory' | 'exchange' | 'rank' | 'member' | 'shop' | 'social' | 'puzzle';

interface LoginPayload {
  readonly user: {
    readonly id: string;
    readonly nickname: string;
  };
  readonly session: {
    readonly token: string;
  };
}

interface PublicConfig {
  readonly wechat?: {
    readonly quick_login_enabled?: boolean;
  };
  readonly sms?: {
    readonly provider: string;
    readonly mock_enabled: boolean;
  };
}

interface PhoneCodePayload {
  readonly sent: boolean;
  readonly provider: string;
  readonly expires_in: number;
  readonly dev_code?: string;
  readonly message: string;
}

interface PhoneLoginFormValues {
  readonly phone: string;
  readonly code: string;
}

interface CampaignProbabilities {
  readonly campaign: CampaignListItem['campaign'];
  readonly prizes: readonly (Prize & { readonly base_prob?: string })[];
  readonly pity_config?: {
    readonly enabled: boolean;
    readonly soft_pity_n: number;
    readonly hard_pity_n: number;
  } | null;
}

interface TabItem {
  readonly key: TabKey;
  readonly label: string;
  readonly icon: ComponentType<{ readonly size?: number; readonly className?: string }>;
}

const phonePattern = /^1[3-9]\d{9}$/;

const tabs: readonly TabItem[] = [
  { key: 'series', label: '系列', icon: Home },
  { key: 'inventory', label: '我的', icon: PackageOpen },
  { key: 'exchange', label: '交换', icon: RefreshCw },
  { key: 'rank', label: '排行', icon: Medal },
  { key: 'member', label: '会员', icon: UserRound },
  { key: 'shop', label: '商店', icon: ShoppingBag },
  { key: 'social', label: '社交', icon: Users },
  { key: 'puzzle', label: '拼图', icon: Puzzle },
];

const rarityStyles: Record<string, { readonly icon: string; readonly label: string; readonly className: string }> = {
  common: { icon: '□', label: '普通', className: 'text-slate-300' },
  rare: { icon: '◆', label: '稀有', className: 'text-sky-300' },
  secret: { icon: '✦', label: '隐藏', className: 'text-violet-300' },
  limited: { icon: '★', label: '限定', className: 'text-amber-300' },
  S: { icon: '★', label: 'S', className: 'text-amber-300' },
  A: { icon: '◆', label: 'A', className: 'text-sky-300' },
  B: { icon: '□', label: 'B', className: 'text-slate-300' },
};

function levelMeta(level?: string): { readonly icon: string; readonly label: string; readonly className: string } {
  return rarityStyles[level ?? ''] ?? { icon: '□', label: level ?? '未知', className: 'text-slate-300' };
}

function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return `${date.getMonth() + 1}/${date.getDate()} ${date.getHours().toString().padStart(2, '0')}:${date
    .getMinutes()
    .toString()
    .padStart(2, '0')}`;
}

function EmptyState({
  icon,
  title,
  description,
}: {
  readonly icon: string;
  readonly title: string;
  readonly description: string;
}): React.ReactNode {
  return (
    <div className="rounded-3xl border border-white/10 bg-white/[0.06] px-6 py-12 text-center text-sm text-violet-100/70">
      <div className="mb-3 text-5xl">{icon}</div>
      <div className="text-base font-semibold text-white">{title}</div>
      <p className="mt-2">{description}</p>
    </div>
  );
}

function SkeletonCards(): React.ReactNode {
  return (
    <div className="grid grid-cols-2 gap-3">
      {Array.from({ length: 4 }).map((_, index) => (
        <div className="h-28 animate-pulse rounded-2xl bg-white/[0.06]" key={index} />
      ))}
    </div>
  );
}

export function LotteryApp(): React.ReactNode {
  const [token, setToken] = useState('');
  const [nickname, setNickname] = useState('');
  const [tab, setTab] = useState<TabKey>('series');
  const [selectedCampaignId, setSelectedCampaignId] = useState<string | null>(null);
  const [showBoxModal, setShowBoxModal] = useState(false);
  const [boxAnimating, setBoxAnimating] = useState(false);
  const [lastDraw, setLastDraw] = useState<BlindBoxDrawResult | null>(null);
  const [showExchangeModal, setShowExchangeModal] = useState(false);
  const queryClient = useQueryClient();

  // 微信 OAuth 回调处理：检查 URL 中是否有 code 参数
  const [wechatLoggingIn, setWechatLoggingIn] = useState(false);
  const [wechatError, setWechatError] = useState('');
  const [showPhoneLogin, setShowPhoneLogin] = useState(false);
  const [phoneLoginForm, setPhoneLoginForm] = useState<PhoneLoginFormValues>({ phone: '', code: '' });
  const [phoneCodeMessage, setPhoneCodeMessage] = useState('');

  // 在组件挂载时检查微信登录回调
  if (typeof window !== 'undefined' && !token && !wechatLoggingIn) {
    const params = new URLSearchParams(window.location.search);
    const code = params.get('code');
    if (code) {
      setWechatLoggingIn(true);
      // 清除 URL 中的 code 参数
      window.history.replaceState({}, '', window.location.pathname);
      apiRequest<{ token: string; user_id: string; nickname: string }>(
        '/api/v1/auth/wechat/login', '', { method: 'POST', body: JSON.stringify({ code }) }
      )
        .then((data) => {
          setToken(data.token);
          setNickname(data.nickname);
        })
        .catch((err) => {
          setWechatError(err instanceof Error ? err.message : '微信登录失败');
        })
        .finally(() => {
          setWechatLoggingIn(false);
        });
    }
  }

  const loginForm = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { nickname: '' },
  });

  const exchangeForm = useForm<ExchangeFormValues>({
    resolver: zodResolver(exchangeSchema),
    defaultValues: { have_prize_id: '', want_prize_id: '' },
  });

  const publicConfigQuery = useQuery({
    queryKey: ['public-config'],
    queryFn: () => apiRequest<PublicConfig>('/api/v1/config/public', ''),
    enabled: !token,
  });

  const smsProvider = publicConfigQuery.data?.sms?.provider?.toLowerCase();
  const isMockSmsProvider = publicConfigQuery.data?.sms?.mock_enabled ?? (smsProvider === 'mock');
  const isWechatQuickLoginEnabled = publicConfigQuery.data?.wechat?.quick_login_enabled === true;

  const loginMutation = useMutation({
    mutationFn: (values: LoginFormValues) =>
      apiRequest<LoginPayload>('/api/v1/auth/guest-login', '', {
        method: 'POST',
        body: JSON.stringify(values),
      }),
    onSuccess: (payload) => {
      setToken(payload.session.token);
      setNickname(payload.user.nickname);
    },
  });

  const sendPhoneCodeMutation = useMutation({
    mutationFn: (phone: string) =>
      apiRequest<PhoneCodePayload>('/api/v1/auth/phone/code', '', {
        method: 'POST',
        body: JSON.stringify({ phone, scene: 'login' }),
      }),
    onSuccess: (payload) => {
      setPhoneCodeMessage(payload.dev_code ? `${payload.message} 开发验证码：${payload.dev_code}` : payload.message);
      setWechatError('');
    },
    onError: (error) => {
      setPhoneCodeMessage('');
      setWechatError(error instanceof Error ? error.message : '验证码发送失败');
    },
  });

  const phoneLoginMutation = useMutation({
    mutationFn: (values: PhoneLoginFormValues) =>
      isMockSmsProvider
        ? apiRequest<LoginPayload>('/api/v1/auth/phone-login', '', {
            method: 'POST',
            body: JSON.stringify({ phone: values.phone }),
          })
        : apiRequest<LoginPayload>('/api/v1/auth/phone/verify', '', {
            method: 'POST',
            body: JSON.stringify({ phone: values.phone, code: values.code, scene: 'login' }),
          }),
    onSuccess: (payload) => {
      setToken(payload.session.token);
      setNickname(payload.user.nickname);
      setShowPhoneLogin(false);
      setPhoneLoginForm({ phone: '', code: '' });
      setPhoneCodeMessage('');
      setWechatError('');
    },
    onError: (error) => {
      setWechatError(error instanceof Error ? error.message : '手机号登录失败');
    },
  });

  const campaignsQuery = useQuery({
    queryKey: ['campaigns', token],
    queryFn: () => apiRequest<CampaignListItem[]>('/api/v1/blindbox/campaigns', token),
    enabled: Boolean(token),
  });

  const memberQuery = useQuery({
    queryKey: ['member', token],
    queryFn: () => apiRequest<UserMember>('/api/v1/blindbox/member', token),
    enabled: Boolean(token),
  });

  const accountQuery = useQuery({
    queryKey: ['me-account', token],
    queryFn: () => apiRequest<UserAccount>('/api/v1/me/account', token),
    enabled: Boolean(token),
  });

  const inventoryQuery = useQuery({
    queryKey: ['inventory', token],
    queryFn: () => apiRequest<UserInventory[]>('/api/v1/blindbox/inventory', token),
    enabled: Boolean(token) && (tab === 'inventory' || tab === 'social' || showExchangeModal),
  });

  const exchangeQuery = useQuery({
    queryKey: ['exchange-offers', token],
    queryFn: () => apiRequest<ExchangeOffer[]>('/api/v1/blindbox/exchange-offers', token),
    enabled: Boolean(token) && tab === 'exchange',
  });

  const leaderboardQuery = useQuery({
    queryKey: ['leaderboard', token],
    queryFn: () => apiRequest<LeaderboardEntry[]>('/api/v1/blindbox/leaderboard?limit=20', token),
    enabled: Boolean(token) && tab === 'rank',
  });

  const pointsLogQuery = useQuery({
    queryKey: ['points-log', token],
    queryFn: () => apiRequest<UserPointsLog[]>('/api/v1/blindbox/points-log', token),
    enabled: Boolean(token) && tab === 'member',
  });

  const shopQuery = useQuery({
    queryKey: ['shop-items', token],
    queryFn: () => apiRequest<ShopItem[]>('/api/v1/shop/items', token),
    enabled: Boolean(token) && tab === 'shop',
  });

  const userItemsQuery = useQuery({
    queryKey: ['user-items', token],
    queryFn: () => apiRequest<UserItem[]>('/api/v1/shop/items/inventory', token),
    enabled: Boolean(token) && tab === 'shop',
  });

  const firstRechargePacksQuery = useQuery({
    queryKey: ['first-recharge-packs', token],
    queryFn: () => apiRequest<FirstRechargePack[]>('/api/v1/first-recharge/packs', token),
    enabled: Boolean(token) && tab === 'shop',
  });

  const firstRechargeStatusQuery = useQuery({
    queryKey: ['first-recharge-status', token],
    queryFn: () => apiRequest<UserFirstRecharge>('/api/v1/first-recharge/status', token),
    enabled: Boolean(token) && tab === 'shop',
  });

  const monthCardQuery = useQuery({
    queryKey: ['month-card', token],
    queryFn: () => apiRequest<MonthCardStatus>('/api/v1/month-card/status', token),
    enabled: Boolean(token) && tab === 'member',
  });

  const battlePassQuery = useQuery({
    queryKey: ['battle-pass', token],
    queryFn: () => apiRequest<BattlePassInfo>('/api/v1/battle-pass/info', token),
    enabled: Boolean(token) && tab === 'member',
  });

  const inviteStatsQuery = useQuery({
    queryKey: ['invite-stats', token],
    queryFn: () => apiRequest<InviteStats>('/api/v1/share/invite-stats', token),
    enabled: Boolean(token) && tab === 'social',
  });

  const assistProgressQuery = useQuery({
    queryKey: ['assist-progress', token],
    queryFn: () => apiRequest<Record<string, AssistProgress>>('/api/v1/share/assist-progress', token),
    enabled: Boolean(token) && tab === 'social',
  });

  const teamQuery = useQuery({
    queryKey: ['team', token],
    queryFn: () => apiRequest<TeamInfo>('/api/v1/team/my', token),
    enabled: Boolean(token) && tab === 'social',
  });

  const giftsQuery = useQuery({
    queryKey: ['incoming-gifts', token],
    queryFn: () => apiRequest<GiftRecord[]>('/api/v1/share/gifts/incoming', token),
    enabled: Boolean(token) && tab === 'social',
  });

  const puzzleQuery = useQuery({
    queryKey: ['puzzles', token],
    queryFn: () => apiRequest<PuzzleInfo[]>('/api/v1/puzzle/my', token),
    enabled: Boolean(token) && tab === 'puzzle',
  });

  const flashQuery = useQuery({
    queryKey: ['flash-list', token],
    queryFn: () => apiRequest<FlashListInfo[]>('/api/v1/flash/list', token),
    enabled: Boolean(token) && tab === 'puzzle',
  });

  const activitiesQuery = useQuery({
    queryKey: ['activities', token],
    queryFn: () => apiRequest<ActivityListInfo[]>('/api/v1/activities', token),
    enabled: Boolean(token) && tab === 'series',
  });

  const selectedCampaign = campaignsQuery.data?.find((item) => item.campaign.id === selectedCampaignId);

  const progressQuery = useQuery({
    queryKey: ['series-progress', token, selectedCampaignId],
    queryFn: () =>
      apiRequest<SeriesProgress>(
        `/api/v1/blindbox/series-progress?campaign_id=${selectedCampaignId ?? ''}`,
        token,
      ),
    enabled: Boolean(token && selectedCampaignId),
  });

  const probabilityQuery = useQuery({
    queryKey: ['campaign-probabilities', token, selectedCampaignId],
    queryFn: () =>
      apiRequest<CampaignProbabilities>(
        `/api/v1/blindbox/campaigns/${selectedCampaignId ?? ''}/probabilities`,
        token,
      ),
    enabled: Boolean(token && selectedCampaignId),
  });

  const allPrizes = useMemo(
    () =>
      (campaignsQuery.data ?? []).flatMap((item) =>
        item.prizes.map((prize) => ({
          ...prize,
          campaign_name: item.campaign.name,
        })),
      ),
    [campaignsQuery.data],
  );

  const duplicateInventory = useMemo(() => {
    const counts = new Map<string, number>();
    for (const item of inventoryQuery.data ?? []) {
      counts.set(item.prize_id, (counts.get(item.prize_id) ?? 0) + 1);
    }
    const seen = new Set<string>();
    return (inventoryQuery.data ?? []).filter((item) => {
      if ((counts.get(item.prize_id) ?? 0) < 2 || seen.has(item.prize_id)) {
        return false;
      }
      seen.add(item.prize_id);
      return true;
    });
  }, [inventoryQuery.data]);

  const drawMutation = useMutation({
    mutationFn: (drawCount: number) =>
      apiRequest<BlindBoxDrawResult>('/api/v1/blindbox/draw', token, {
        method: 'POST',
        body: JSON.stringify({ campaign_id: selectedCampaignId, draw_count: drawCount }),
      }),
    onSuccess: (payload) => {
      window.setTimeout(() => {
        setLastDraw(payload);
        setBoxAnimating(false);
      }, 900);
      void queryClient.invalidateQueries({ queryKey: ['member', token] });
      void queryClient.invalidateQueries({ queryKey: ['campaigns', token] });
      void queryClient.invalidateQueries({ queryKey: ['inventory', token] });
      void queryClient.invalidateQueries({ queryKey: ['series-progress', token, selectedCampaignId] });
    },
    onError: () => {
      setBoxAnimating(false);
    },
  });

  const checkInMutation = useMutation({
    mutationFn: () => apiRequest('/api/v1/blindbox/checkin', token, { method: 'POST' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['member', token] });
      void queryClient.invalidateQueries({ queryKey: ['points-log', token] });
    },
  });

  const shareMutation = useMutation({
    mutationFn: () => apiRequest('/api/v1/blindbox/share-reward', token, { method: 'POST' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['member', token] });
      void queryClient.invalidateQueries({ queryKey: ['points-log', token] });
    },
  });

  const buyShopMutation = useMutation({
    mutationFn: (shopItemId: string) =>
      apiRequest('/api/v1/shop/buy', token, {
        method: 'POST',
        body: JSON.stringify({ shop_item_id: shopItemId, quantity: 1 }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['member', token] });
      void queryClient.invalidateQueries({ queryKey: ['user-items', token] });
    },
  });

  const firstRechargeMutation = useMutation({
    mutationFn: (packId: string) =>
      apiRequest('/api/v1/first-recharge/claim', token, {
        method: 'POST',
        body: JSON.stringify({ pack_id: packId }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['member', token] });
      void queryClient.invalidateQueries({ queryKey: ['user-items', token] });
      void queryClient.invalidateQueries({ queryKey: ['first-recharge-status', token] });
    },
  });

  const buyMonthCardMutation = useMutation({
    mutationFn: (cardType: 'weekly' | 'monthly' | 'season') =>
      apiRequest('/api/v1/month-card/buy', token, {
        method: 'POST',
        body: JSON.stringify({ card_type: cardType }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['member', token] });
      void queryClient.invalidateQueries({ queryKey: ['month-card', token] });
    },
  });

  const inviteMutation = useMutation({
    mutationFn: () => apiRequest('/api/v1/share/invite', token, { method: 'POST' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['invite-stats', token] });
    },
  });

  const assistMutation = useMutation({
    mutationFn: () =>
      apiRequest('/api/v1/share/assist', token, {
        method: 'POST',
        body: JSON.stringify({ assist_type: 'free_draw', helper_id: `helper_${Date.now()}` }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['assist-progress', token] });
      void queryClient.invalidateQueries({ queryKey: ['invite-stats', token] });
    },
  });

  const createTeamMutation = useMutation({
    mutationFn: () =>
      apiRequest('/api/v1/team/create', token, {
        method: 'POST',
        body: JSON.stringify({ name: `${nickname || '我的'}的队伍`, max_members: 3, goal_draws: 20 }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['team', token] });
    },
  });

  const sendGiftMutation = useMutation({
    mutationFn: (prizeId: string) =>
      apiRequest('/api/v1/share/gift', token, {
        method: 'POST',
        body: JSON.stringify({ receiver_id: 'demo_friend', prize_id: prizeId }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['incoming-gifts', token] });
    },
  });

  const composePuzzleMutation = useMutation({
    mutationFn: (templateId: string) =>
      apiRequest('/api/v1/puzzle/compose', token, {
        method: 'POST',
        body: JSON.stringify({ template_id: templateId }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['puzzles', token] });
      void queryClient.invalidateQueries({ queryKey: ['member', token] });
    },
  });

  const flashPurchaseMutation = useMutation({
    mutationFn: (flashId: string) => apiRequest(`/api/v1/flash/${flashId}/purchase`, token, { method: 'POST' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['flash-list', token] });
      void queryClient.invalidateQueries({ queryKey: ['member', token] });
    },
  });

  const publishExchangeMutation = useMutation({
    mutationFn: (values: ExchangeFormValues) =>
      apiRequest<ExchangeOffer>('/api/v1/blindbox/exchange-offers', token, {
        method: 'POST',
        body: JSON.stringify(values),
      }),
    onSuccess: () => {
      setShowExchangeModal(false);
      exchangeForm.reset();
      void queryClient.invalidateQueries({ queryKey: ['exchange-offers', token] });
    },
  });

  const acceptExchangeMutation = useMutation({
    mutationFn: (offerId: string) =>
      apiRequest<ExchangeOffer>(`/api/v1/blindbox/exchange-offers/${offerId}/accept`, token, { method: 'POST' }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['exchange-offers', token] });
      void queryClient.invalidateQueries({ queryKey: ['inventory', token] });
    },
  });

  function openCampaign(campaignId: string): void {
    setSelectedCampaignId(campaignId);
    setLastDraw(null);
  }

  function handleDraw(drawCount: number): void {
    setShowBoxModal(true);
    setBoxAnimating(true);
    setLastDraw(null);
    drawMutation.mutate(drawCount);
  }

  function updatePhoneLoginForm(field: keyof PhoneLoginFormValues, value: string): void {
    setPhoneLoginForm((current) => ({ ...current, [field]: value }));
  }

  function phoneFromForm(): string {
    return phoneLoginForm.phone.replace(/\s+/g, '');
  }

  function validatePhoneLoginPhone(): string | null {
    const phone = phoneFromForm();
    if (!phonePattern.test(phone)) {
      setWechatError('请输入正确的 11 位手机号码');
      return null;
    }
    return phone;
  }

  function handleSendPhoneCode(): void {
    const phone = validatePhoneLoginPhone();
    if (!phone) {
      return;
    }
    setPhoneLoginForm((current) => ({ ...current, phone }));
    setWechatError('');
    if (isMockSmsProvider) {
      setPhoneCodeMessage('当前为 mock 短信配置，无需获取验证码，点击确认即可进入主页面。');
      return;
    }
    sendPhoneCodeMutation.mutate(phone);
  }

  function handleConfirmPhoneLogin(): void {
    const phone = validatePhoneLoginPhone();
    if (!phone) {
      return;
    }
    if (publicConfigQuery.isLoading) {
      setWechatError('短信配置加载中，请稍后再试');
      return;
    }
    const code = phoneLoginForm.code.trim();
    if (!isMockSmsProvider && !code) {
      setWechatError('请输入手机验证码');
      return;
    }
    setWechatError('');
    phoneLoginMutation.mutate({ phone, code });
  }

  if (!token) {
    return (
      <main className="min-h-screen bg-[linear-gradient(160deg,#0d0f1a_0%,#1a1040_38%,#0f2027_100%)] px-4 py-8 text-violet-50">
        <section className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-[480px] items-center">
          <div className="w-full rounded-[28px] border border-white/15 bg-[linear-gradient(160deg,rgba(26,16,64,0.96),rgba(15,32,39,0.96))] p-7 text-center shadow-[0_24px_80px_rgba(0,0,0,0.45)]">
            <div className="mx-auto mb-5 flex size-20 items-center justify-center rounded-3xl bg-white/10 text-pink-300 ring-1 ring-white/15">
              <Gift size={42} />
            </div>
            <h1 className="text-4xl font-black tracking-tight text-white">BOX·MAGIC</h1>
            <p className="mt-3 text-sm text-violet-100/70">盲盒抽奖平台 · 开启你的收藏之旅</p>
            <form
              className="mt-7 space-y-3 text-left"
              onSubmit={loginForm.handleSubmit((values) => loginMutation.mutate(values))}
            >
              <input
                className="w-full rounded-2xl border-0 bg-white px-4 py-3 text-[15px] text-slate-950 outline-none ring-2 ring-transparent placeholder:text-slate-400 focus:ring-violet-300"
                placeholder="输入昵称（留空随机）"
                {...loginForm.register('nickname')}
              />
              <button
                className="flex w-full items-center justify-center gap-2 rounded-2xl bg-[linear-gradient(135deg,#f472b6,#a78bfa)] px-4 py-3 font-bold text-white disabled:opacity-60"
                disabled={loginMutation.isPending}
                type="submit"
              >
                {loginMutation.isPending ? <Loader2 className="animate-spin" size={18} /> : <Sparkles size={18} />}
                开始抽盒
              </button>
              {loginMutation.error ? (
                <p className="rounded-xl border border-red-300/20 bg-red-500/15 px-4 py-3 text-sm text-red-100">
                  {loginMutation.error.message}
                </p>
              ) : null}
            </form>

            {/* 微信登录分隔线 */}
            <div className="mt-5 flex items-center gap-3 text-xs text-violet-100/40">
              <span className="h-px flex-1 bg-white/10" />
              <span>其他登录方式</span>
              <span className="h-px flex-1 bg-white/10" />
            </div>

            {isWechatQuickLoginEnabled ? (
              <button
                type="button"
                onClick={() => {
                  apiRequest<{ url: string }>('/api/v1/auth/wechat/oauth-url', '')
                    .then((data) => {
                      window.location.href = data.url;
                    })
                    .catch(() => {
                      window.location.href = `/api/v1/auth/wechat/oauth-url`;
                    });
                }}
                className="mt-4 flex w-full items-center justify-center gap-2 rounded-2xl border border-white/15 bg-white/[0.06] px-4 py-3 font-medium text-white hover:bg-white/[0.10]"
              >
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 01.213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 00.167-.054l1.903-1.114a.864.864 0 01.717-.098 10.16 10.16 0 002.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348zM5.785 5.991c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178A1.17 1.17 0 014.623 7.17c0-.651.52-1.18 1.162-1.18zm5.813 0c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178 1.17 1.17 0 01-1.162-1.178c0-.651.52-1.18 1.162-1.18zm5.34 2.867c-1.797-.052-3.746.512-5.28 1.786-1.72 1.428-2.687 3.72-1.78 6.22.942 2.453 3.666 4.229 6.884 4.229.826 0 1.622-.12 2.361-.336a.722.722 0 01.598.082l1.584.926a.272.272 0 00.14.045c.134 0 .24-.11.24-.245 0-.06-.024-.12-.04-.178l-.324-1.233a.492.492 0 01.177-.553C23.028 18.48 24 16.82 24 14.98c0-3.21-2.931-5.837-7.062-6.122zm-2.18 2.956c.535 0 .969.44.969.982a.976.976 0 01-.969.983.976.976 0 01-.969-.983c0-.542.434-.982.97-.982zm4.36 0c.535 0 .969.44.969.982a.976.976 0 01-.969.983.976.976 0 01-.969-.983c0-.542.434-.982.97-.982z" />
                </svg>
                微信一键登录
              </button>
            ) : null}

            {/* 手机号码登录按钮 */}
            <button
              type="button"
              onClick={() => {
                setShowPhoneLogin((current) => !current);
                setWechatError('');
                setPhoneCodeMessage('');
              }}
              className="mt-3 flex w-full items-center justify-center gap-2 rounded-2xl border border-white/15 bg-white/[0.06] px-4 py-3 font-medium text-white hover:bg-white/[0.10]"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <rect x="5" y="2" width="14" height="20" rx="2" ry="2" />
                <line x1="12" y1="18" x2="12.01" y2="18" />
              </svg>
              使用手机号码登录
            </button>

            {showPhoneLogin ? (
              <div className="mt-3 rounded-2xl border border-violet-200/15 bg-white/[0.08] p-4 text-left">
                <div className="mb-3 text-center text-sm font-semibold text-white">使用手机号码登录</div>
                <div className="space-y-3">
                  <input
                    className="w-full rounded-2xl border-0 bg-white px-4 py-3 text-[15px] text-slate-950 outline-none ring-2 ring-transparent placeholder:text-slate-400 focus:ring-violet-300"
                    inputMode="tel"
                    maxLength={11}
                    onChange={(event) => updatePhoneLoginForm('phone', event.target.value)}
                    placeholder="填写手机号码"
                    value={phoneLoginForm.phone}
                  />
                  <div className="grid grid-cols-[1fr_auto] gap-2">
                    <input
                      className="min-w-0 rounded-2xl border-0 bg-white px-4 py-3 text-[15px] text-slate-950 outline-none ring-2 ring-transparent placeholder:text-slate-400 focus:ring-violet-300 disabled:bg-white/70"
                      disabled={isMockSmsProvider}
                      inputMode="numeric"
                      maxLength={8}
                      onChange={(event) => updatePhoneLoginForm('code', event.target.value)}
                      placeholder={isMockSmsProvider ? 'mock 模式无需验证码' : '填写验证码'}
                      value={phoneLoginForm.code}
                    />
                    <button
                      className="rounded-2xl border border-white/15 bg-white/10 px-3 py-3 text-sm font-semibold text-white disabled:opacity-60"
                      disabled={sendPhoneCodeMutation.isPending || phoneLoginMutation.isPending || publicConfigQuery.isLoading}
                      onClick={handleSendPhoneCode}
                      type="button"
                    >
                      {sendPhoneCodeMutation.isPending ? <Loader2 className="animate-spin" size={16} /> : '获取验证码'}
                    </button>
                  </div>
                  <p className="text-xs text-violet-100/55">
                    {publicConfigQuery.isLoading
                      ? '正在读取短信配置...'
                      : isMockSmsProvider
                        ? '当前 SMS_PROVIDER=mock，确认后将直接进入主页面。'
                        : '请先获取验证码，再输入短信验证码完成登录。'}
                  </p>
                  {phoneCodeMessage ? <p className="text-xs text-violet-100/70">{phoneCodeMessage}</p> : null}
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      className="rounded-2xl border border-white/15 bg-white/[0.06] px-4 py-3 font-semibold text-white"
                      onClick={() => {
                        setShowPhoneLogin(false);
                        setPhoneCodeMessage('');
                      }}
                      type="button"
                    >
                      取消
                    </button>
                    <button
                      className="flex items-center justify-center gap-2 rounded-2xl bg-[linear-gradient(135deg,#f472b6,#a78bfa)] px-4 py-3 font-bold text-white disabled:opacity-60"
                      disabled={phoneLoginMutation.isPending || publicConfigQuery.isLoading}
                      onClick={handleConfirmPhoneLogin}
                      type="button"
                    >
                      {phoneLoginMutation.isPending ? <Loader2 className="animate-spin" size={18} /> : null}
                      确认登录
                    </button>
                  </div>
                </div>
              </div>
            ) : null}

            {/* 微信登录中 */}
            {wechatLoggingIn ? (
              <div className="mt-3 flex items-center justify-center gap-2 text-sm text-violet-100/70">
                <Loader2 className="animate-spin" size={16} />
                微信授权中...
              </div>
            ) : null}

            {/* 微信登录错误 */}
            {wechatError ? (
              <p className="mt-3 rounded-xl border border-red-300/20 bg-red-500/15 px-4 py-3 text-sm text-red-100">
                {wechatError}
              </p>
            ) : null}
          </div>
        </section>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-[linear-gradient(160deg,#0d0f1a_0%,#1a1040_38%,#0f2027_100%)] pb-28 text-violet-50">
      <div className="mx-auto max-w-[480px] px-3">
        <header className="sticky top-0 z-40 flex items-center gap-2 bg-[linear-gradient(180deg,#0d0f1a_68%,rgba(13,15,26,0))] py-3">
          <div className="min-w-0 flex-1">
            <div className="text-lg font-black tracking-tight text-white">BOX·MAGIC</div>
            <div className="truncate text-xs text-violet-100/60">
              {accountQuery.data?.user.nickname || nickname ? `用户 ${accountQuery.data?.user.nickname ?? nickname}` : '盲盒收藏家'}
              {accountQuery.data?.user.mobile ? ` · ${accountQuery.data.user.mobile.replace(/^(\d{3})\d{4}(\d{4})$/, '$1****$2')}` : ''}
            </div>
          </div>
          <span className="rounded-full bg-[linear-gradient(135deg,#fbbf24,#f59e0b)] px-3 py-1 text-xs font-black text-slate-950">
            {memberQuery.data?.points ?? 0}
          </span>
          <button
            className="rounded-full border border-white/15 bg-white/10 px-3 py-1.5 text-xs font-semibold text-white disabled:opacity-50"
            disabled={checkInMutation.isPending}
            onClick={() => checkInMutation.mutate()}
            type="button"
          >
            签到
          </button>
        </header>

        {accountQuery.data && accountQuery.data.status !== 'active' ? (
          <div className="mb-3 rounded-2xl border border-amber-300/20 bg-amber-500/15 px-4 py-3 text-sm text-amber-50">
            当前账号状态：{accountQuery.data.status}。
            {accountQuery.data.status === 'pending_phone'
              ? '请完成手机号验证后再进行抽盒、兑换、购买和领奖。'
              : '部分资产相关操作可能受限。'}
          </div>
        ) : null}

        <section className="mb-3 rounded-3xl border border-violet-300/20 bg-[linear-gradient(135deg,rgba(167,139,250,0.16),rgba(244,114,182,0.16))] p-3">
          <div className="flex gap-3 overflow-x-auto [scrollbar-width:none]">
            {(campaignsQuery.data ?? []).slice(0, 5).map((item) => (
              <button
                className="min-w-[220px] rounded-2xl border border-white/10 bg-white/10 p-3 text-left active:scale-[0.98]"
                key={item.campaign.id}
                onClick={() => openCampaign(item.campaign.id)}
                type="button"
              >
                <span className="rounded-full bg-[linear-gradient(135deg,#f472b6,#a78bfa)] px-2 py-0.5 text-[10px] font-bold">
                  {item.campaign.status}
                </span>
                <div className="mt-2 font-bold text-white">{item.campaign.name}</div>
                <p className="mt-1 line-clamp-2 text-xs text-violet-100/65">{item.campaign.campaign_summary}</p>
              </button>
            ))}
          </div>
        </section>

        {selectedCampaign ? (
          <section className="space-y-4">
            <button
              className="text-sm text-violet-100/70"
              onClick={() => {
                setSelectedCampaignId(null);
                setLastDraw(null);
              }}
              type="button"
            >
              ← 返回系列
            </button>
            <div className="rounded-3xl border border-white/10 bg-white/[0.06] p-4 text-center">
              <h2 className="text-2xl font-black text-white">{selectedCampaign.campaign.name}</h2>
              <p className="mt-2 text-sm text-violet-100/65">{selectedCampaign.campaign.campaign_summary}</p>
              {progressQuery.data ? (
                <div className="mt-4 text-xs text-violet-100/65">
                  已收集 {progressQuery.data.collected_items}/{progressQuery.data.total_items} 款
                  {progressQuery.data.duplicates > 0 ? ` · 重复 ${progressQuery.data.duplicates}` : ''}
                  <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-white/10">
                    <div
                      className="h-full rounded-full bg-[linear-gradient(90deg,#f472b6,#a78bfa)]"
                      style={{ width: `${progressQuery.data.progress_percent}%` }}
                    />
                  </div>
                </div>
              ) : null}
            </div>

            <div className="grid grid-cols-3 gap-2">
              {(probabilityQuery.data?.prizes ?? selectedCampaign.prizes).map((prize) => {
                const owned = progressQuery.data?.collected_prizes.find((item) => item.id === prize.id);
                const meta = levelMeta(prize.level);
                const probabilityPrize = prize as Prize & { readonly base_prob?: string };
                return (
                  <div
                    className={`relative rounded-2xl border bg-white/[0.06] px-2 py-3 text-center ${
                      owned ? 'border-violet-300/60 shadow-[0_0_18px_rgba(167,139,250,0.22)]' : 'border-white/10'
                    }`}
                    key={prize.id}
                  >
                    {owned ? (
                      <span className="absolute -right-1 -top-1 rounded-full bg-pink-400 px-1.5 text-[10px] font-bold text-white">
                        x{owned.count}
                      </span>
                    ) : null}
                    <div className={`text-3xl ${meta.className}`}>{meta.icon}</div>
                    <div className={`mt-1 line-clamp-1 text-xs font-semibold ${meta.className}`}>{prize.name}</div>
                    <div className="mt-1 text-[11px] text-violet-100/55">{meta.label}</div>
                    {probabilityPrize.base_prob ? (
                      <div className="text-[11px] text-violet-100/45">{probabilityPrize.base_prob}</div>
                    ) : null}
                  </div>
                );
              })}
            </div>

            {probabilityQuery.data?.pity_config?.enabled ? (
              <div className="rounded-2xl border border-white/10 bg-white/[0.05] p-3 text-center text-xs text-violet-100/65">
                保底规则：软保底 {probabilityQuery.data.pity_config.soft_pity_n} 次 · 硬保底{' '}
                {probabilityQuery.data.pity_config.hard_pity_n} 次
              </div>
            ) : null}

            <div className="grid grid-cols-2 gap-3">
              <button
                className="rounded-2xl bg-[linear-gradient(135deg,#f472b6,#db2777)] px-4 py-3 font-black text-white disabled:opacity-50"
                disabled={drawMutation.isPending}
                onClick={() => handleDraw(1)}
                type="button"
              >
                单抽 100分
              </button>
              <button
                className="rounded-2xl bg-[linear-gradient(135deg,#a78bfa,#7c3aed)] px-4 py-3 font-black text-white disabled:opacity-50"
                disabled={drawMutation.isPending}
                onClick={() => handleDraw(10)}
                type="button"
              >
                十连 950分
              </button>
            </div>
            {drawMutation.error ? (
              <p className="rounded-xl border border-red-300/20 bg-red-500/15 px-4 py-3 text-sm text-red-100">
                {drawMutation.error.message}
              </p>
            ) : null}
          </section>
        ) : (
          <>
            {tab === 'series' ? (
              <section className="space-y-3">
                {(activitiesQuery.data ?? []).map((item) => (
                  <article className="rounded-3xl border border-pink-300/30 bg-pink-400/10 p-4" key={item.activity.id}>
                    <div className="text-xs font-bold text-pink-200">{item.activity.type}</div>
                    <h3 className="mt-1 font-black text-white">{item.activity.name}</h3>
                    <p className="mt-1 text-sm text-violet-100/65">{item.activity.description}</p>
                    {item.rewards?.[0] ? (
                      <div className="mt-2 text-xs text-pink-100/80">奖励：{item.rewards[0].reward_name} x{item.rewards[0].reward_qty}</div>
                    ) : null}
                  </article>
                ))}
                {campaignsQuery.isLoading ? <SkeletonCards /> : null}
                {(campaignsQuery.data ?? []).map((item) => (
                  <button
                    className="w-full rounded-3xl border border-white/10 bg-white/[0.06] p-4 text-left shadow-[0_8px_32px_rgba(0,0,0,0.32)] transition active:scale-[0.98]"
                    key={item.campaign.id}
                    onClick={() => openCampaign(item.campaign.id)}
                    type="button"
                  >
                    <h3 className="text-lg font-black text-white">{item.campaign.name}</h3>
                    <p className="mt-1 line-clamp-2 text-sm text-violet-100/65">{item.campaign.campaign_summary}</p>
                    <div className="mt-3 flex gap-3 text-xs text-violet-100/55">
                      <span>目标 {item.prizes.length} 款</span>
                      <span>每日 {item.campaign.daily_draw_limit} 次</span>
                      {item.progress ? (
                        <span>
                          收集 {item.progress.collected_items}/{item.progress.total_items}
                        </span>
                      ) : null}
                    </div>
                    {item.progress ? (
                      <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-white/10">
                        <div
                          className="h-full rounded-full bg-[linear-gradient(90deg,#f472b6,#a78bfa)]"
                          style={{ width: `${item.progress.progress_percent}%` }}
                        />
                      </div>
                    ) : null}
                  </button>
                ))}
              </section>
            ) : null}

            {tab === 'inventory' ? (
              <section className="space-y-4">
                <div className="flex items-center justify-between">
                  <h2 className="text-lg font-black text-white">我的收藏</h2>
                  <button className="rounded-full border border-white/15 bg-white/10 px-3 py-1.5 text-xs" type="button">
                    共 {inventoryQuery.data?.length ?? 0} 件
                  </button>
                </div>
                {inventoryQuery.isLoading ? <SkeletonCards /> : null}
                {inventoryQuery.data?.length ? (
                  <div className="grid grid-cols-3 gap-2">
                    {inventoryQuery.data.map((item) => {
                      const meta = levelMeta(item.prize_level);
                      return (
                        <div className="relative rounded-2xl border border-violet-300/40 bg-white/[0.06] p-3 text-center" key={item.id}>
                          <div className={`text-3xl ${meta.className}`}>{meta.icon}</div>
                          <div className={`mt-1 line-clamp-1 text-xs font-semibold ${meta.className}`}>
                            {item.prize_name}
                          </div>
                          <div className="mt-1 text-[11px] text-violet-100/55">{meta.label}</div>
                        </div>
                      );
                    })}
                  </div>
                ) : !inventoryQuery.isLoading ? (
                  <EmptyState icon="□" title="暂无收藏" description="还没有收集到任何盲盒款式，去抽盒吧。" />
                ) : null}
              </section>
            ) : null}

            {tab === 'exchange' ? (
              <section className="space-y-3">
                <div className="flex items-center justify-between">
                  <h2 className="text-lg font-black text-white">交换市场</h2>
                  <button
                    className="rounded-full border border-white/15 bg-white/10 px-3 py-1.5 text-xs font-semibold"
                    onClick={() => setShowExchangeModal(true)}
                    type="button"
                  >
                    + 发布
                  </button>
                </div>
                {(exchangeQuery.data ?? []).map((offer) => (
                  <article
                    className="flex items-center justify-between gap-3 rounded-3xl border border-white/10 bg-white/[0.06] p-4"
                    key={offer.id}
                  >
                    <div className="min-w-0 text-sm">
                      <div className="font-bold text-white">{offer.user_nickname || offer.user_id}</div>
                      <div className="mt-1 text-violet-100/65">
                        {offer.have_prize_name} <span className="mx-1 text-violet-100/35">→</span> {offer.want_prize_name}
                      </div>
                    </div>
                    <button
                      className="shrink-0 rounded-xl bg-violet-400 px-3 py-2 text-xs font-bold text-white disabled:opacity-50"
                      disabled={acceptExchangeMutation.isPending}
                      onClick={() => acceptExchangeMutation.mutate(offer.id)}
                      type="button"
                    >
                      接受
                    </button>
                  </article>
                ))}
                {exchangeQuery.data?.length === 0 ? (
                  <EmptyState icon="↔" title="暂无交换挂单" description="发布重复款，换回缺少的系列款式。" />
                ) : null}
              </section>
            ) : null}

            {tab === 'rank' ? (
              <section className="space-y-3">
                <h2 className="text-lg font-black text-white">收集排行榜</h2>
                {(leaderboardQuery.data ?? []).map((entry, index) => (
                  <article
                    className={`flex items-center gap-3 rounded-2xl border bg-white/[0.06] p-3 ${
                      index === 0 ? 'border-amber-300/50' : index === 1 ? 'border-slate-300/40' : index === 2 ? 'border-orange-300/40' : 'border-white/10'
                    }`}
                    key={entry.user_id}
                  >
                    <div className="w-9 text-xl font-black">{index < 3 ? ['1', '2', '3'][index] : `#${index + 1}`}</div>
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-semibold text-white">{entry.nickname || entry.user_id}</div>
                      <div className="text-xs text-violet-100/60">
                        收集 {entry.collected_count}/{entry.total_count} · {entry.progress_percent.toFixed(1)}%
                      </div>
                    </div>
                  </article>
                ))}
                {leaderboardQuery.data?.length === 0 ? (
                  <EmptyState icon="△" title="暂无排行数据" description="完成抽盒收集后会出现在这里。" />
                ) : null}
              </section>
            ) : null}

            {tab === 'member' ? (
              <section className="space-y-4">
                <div className="overflow-hidden rounded-3xl border border-white/10 bg-white/[0.06] p-6 text-center">
                  <Trophy className="mx-auto text-amber-300" size={48} />
                  <div className="mt-2 text-xl font-black text-white">{memberQuery.data?.level ?? 'normal'} 会员</div>
                  <div className="mt-1 text-sm text-violet-100/65">{memberQuery.data?.points ?? 0} 积分</div>
                  <div className="mt-3 text-xs text-violet-100/55">
                    累计抽盒 {memberQuery.data?.total_draws ?? 0} 次 · 累计消费 {memberQuery.data?.total_spent ?? 0} 分
                  </div>
                </div>
                <button
                  className="flex w-full items-center justify-center gap-2 rounded-2xl border border-white/15 bg-white/10 px-4 py-3 text-sm font-bold disabled:opacity-50"
                  disabled={shareMutation.isPending}
                  onClick={() => shareMutation.mutate()}
                  type="button"
                >
                  <Share2 size={16} />
                  分享领奖励
                </button>
                <div className="rounded-3xl border border-white/10 bg-white/[0.06] p-4">
                  <h3 className="font-bold text-white">月卡 / 战令</h3>
                  <p className="mt-1 text-sm text-violet-100/65">
                    {monthCardQuery.data?.has_card ? `${monthCardQuery.data.card_type} · 剩余 ${monthCardQuery.data.days_left} 天` : '未开通月卡'}
                  </p>
                  <p className="mt-1 text-sm text-violet-100/65">
                    {battlePassQuery.data?.season?.name ?? '暂无赛季'} · 等级 {battlePassQuery.data?.user_pass?.level ?? 1}
                  </p>
                  <button
                    className="mt-3 rounded-2xl bg-amber-400 px-3 py-2 text-xs font-bold text-slate-950 disabled:opacity-50"
                    disabled={buyMonthCardMutation.isPending}
                    onClick={() => buyMonthCardMutation.mutate('monthly')}
                    type="button"
                  >
                    购买月卡
                  </button>
                </div>
                <div>
                  <h3 className="mb-2 text-base font-black text-white">积分变动</h3>
                  <div className="space-y-2">
                    {(pointsLogQuery.data ?? []).slice(0, 20).map((log) => (
                      <div className="flex gap-3 border-b border-white/5 py-2 text-xs" key={log.id}>
                        <span className="w-16 shrink-0 text-violet-100/45">{formatDateTime(log.created_at)}</span>
                        <span className="min-w-0 flex-1 truncate text-violet-100/75">{log.remark || log.reason}</span>
                        <span className={log.points >= 0 ? 'font-bold text-emerald-300' : 'font-bold text-pink-300'}>
                          {log.points >= 0 ? '+' : ''}
                          {log.points}
                        </span>
                      </div>
                    ))}
                    {pointsLogQuery.data?.length === 0 ? (
                      <EmptyState icon="◇" title="暂无积分记录" description="签到、抽盒和分享会生成积分流水。" />
                    ) : null}
                  </div>
                </div>
              </section>
            ) : null}

            {tab === 'shop' ? (
              <section className="space-y-4">
                <h2 className="text-lg font-black text-white">商店</h2>
                <div className="rounded-3xl border border-amber-300/30 bg-amber-400/10 p-4">
                  <h3 className="font-bold text-amber-100">首充礼包</h3>
                  <div className="mt-3 grid gap-2">
                    {(firstRechargePacksQuery.data ?? []).map((pack) => {
                      const claimed = firstRechargeStatusQuery.data?.claimed.includes(pack.id);
                      return (
                        <button
                          className="rounded-2xl border border-white/10 bg-white/[0.06] p-3 text-left disabled:opacity-50"
                          disabled={claimed || firstRechargeMutation.isPending}
                          key={pack.id}
                          onClick={() => firstRechargeMutation.mutate(pack.id)}
                          type="button"
                        >
                          <div className="font-bold text-white">{pack.name}</div>
                          <div className="mt-1 text-xs text-violet-100/60">{pack.description}</div>
                          <div className="mt-1 text-xs text-amber-200">{claimed ? '已领取' : '领取礼包'}</div>
                        </button>
                      );
                    })}
                  </div>
                </div>
                <div className="rounded-3xl border border-white/10 bg-white/[0.06] p-4">
                  <h3 className="font-bold text-white">我的道具</h3>
                  <div className="mt-2 flex flex-wrap gap-2 text-xs">
                    {(userItemsQuery.data ?? []).map((item) => (
                      <span className="rounded-full bg-white/10 px-3 py-1 text-violet-100/75" key={item.item_type}>
                        {item.item_type}: {item.quantity}
                      </span>
                    ))}
                  </div>
                </div>
                {(shopQuery.data ?? []).map((item) => (
                  <article className="rounded-3xl border border-white/10 bg-white/[0.06] p-4" key={item.id}>
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <h3 className="font-bold text-white">{item.name}</h3>
                        <p className="mt-1 text-sm text-violet-100/65">{item.description}</p>
                        <div className="mt-2 text-xs text-violet-100/55">{item.price_points} 积分 · {item.item_qty} 个</div>
                      </div>
                      <button
                        className="rounded-xl bg-amber-400 px-3 py-2 text-xs font-bold text-slate-950 disabled:opacity-50"
                        disabled={buyShopMutation.isPending}
                        onClick={() => buyShopMutation.mutate(item.id)}
                        type="button"
                      >
                        购买
                      </button>
                    </div>
                  </article>
                ))}
              </section>
            ) : null}

            {tab === 'social' ? (
              <section className="space-y-4">
                <h2 className="text-lg font-black text-white">社交中心</h2>
                <div className="rounded-3xl border border-white/10 bg-white/[0.06] p-4">
                  <h3 className="font-bold text-white">邀请助力</h3>
                  <p className="mt-1 text-sm text-violet-100/65">
                    已邀请 {inviteStatsQuery.data?.total_invites ?? 0} 人 · 助力 {inviteStatsQuery.data?.total_assists ?? 0} 次
                  </p>
                  <div className="mt-3 grid grid-cols-2 gap-2">
                    <button className="rounded-2xl bg-violet-400 px-3 py-2 text-sm font-bold text-white" onClick={() => inviteMutation.mutate()} type="button">
                      生成邀请
                    </button>
                    <button className="rounded-2xl border border-white/15 bg-white/10 px-3 py-2 text-sm font-bold" onClick={() => assistMutation.mutate()} type="button">
                      模拟助力
                    </button>
                  </div>
                  <div className="mt-3 space-y-2">
                    {Object.values(assistProgressQuery.data ?? {}).map((item) => (
                      <div className="text-xs text-violet-100/70" key={item.assist_type}>
                        {item.assist_type}: {item.current}/{item.target_count} {item.claimed ? '· 已领取' : ''}
                      </div>
                    ))}
                  </div>
                </div>
                <div className="rounded-3xl border border-white/10 bg-white/[0.06] p-4">
                  <h3 className="font-bold text-white">组队开盒</h3>
                  {teamQuery.data?.team ? (
                    <p className="mt-1 text-sm text-violet-100/65">
                      {teamQuery.data.team.name} · {teamQuery.data.members.length}人 · {teamQuery.data.team.current_draws}/{teamQuery.data.team.goal_draws} 次
                    </p>
                  ) : (
                    <button className="mt-3 rounded-2xl bg-pink-400 px-3 py-2 text-sm font-bold text-white" onClick={() => createTeamMutation.mutate()} type="button">
                      创建队伍
                    </button>
                  )}
                </div>
                <div className="rounded-3xl border border-white/10 bg-white/[0.06] p-4">
                  <h3 className="font-bold text-white">礼物</h3>
                  <p className="mt-1 text-xs text-violet-100/55">收到 {giftsQuery.data?.length ?? 0} 份礼物</p>
                  {inventoryQuery.data?.[0] ? (
                    <button className="mt-3 rounded-2xl border border-white/15 bg-white/10 px-3 py-2 text-sm font-bold" onClick={() => sendGiftMutation.mutate(inventoryQuery.data[0].prize_id)} type="button">
                      赠送一个收藏给示例好友
                    </button>
                  ) : null}
                </div>
              </section>
            ) : null}

            {tab === 'puzzle' ? (
              <section className="space-y-4">
                <h2 className="text-lg font-black text-white">碎片拼图</h2>
                {(puzzleQuery.data ?? []).map((item) => (
                  <article className="rounded-3xl border border-white/10 bg-white/[0.06] p-4" key={item.template.id}>
                    <h3 className="font-bold text-white">{item.template.name}</h3>
                    <p className="mt-1 text-sm text-violet-100/65">
                      已收集 {item.progress.collected.length}/{item.template.total_pieces} · 奖励 {item.template.reward_name}
                    </p>
                    <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-white/10">
                      <div className="h-full rounded-full bg-emerald-300" style={{ width: `${item.progress_percent}%` }} />
                    </div>
                    <button className="mt-3 rounded-xl bg-emerald-400 px-3 py-2 text-xs font-bold text-slate-950 disabled:opacity-50" disabled={composePuzzleMutation.isPending} onClick={() => composePuzzleMutation.mutate(item.template.id)} type="button">
                      拼合领奖
                    </button>
                  </article>
                ))}
                <h2 className="text-lg font-black text-white">限时抢购</h2>
                {(flashQuery.data ?? []).map((item) => (
                  <article className="rounded-3xl border border-white/10 bg-white/[0.06] p-4" key={item.flash.id}>
                    <h3 className="font-bold text-white">{item.flash.name}</h3>
                    <p className="mt-1 text-sm text-violet-100/65">{item.flash.description}</p>
                    <div className="mt-2 text-xs text-violet-100/55">库存 {item.flash.remaining_stock} · {item.flash.price_points} 积分</div>
                    <button className="mt-3 rounded-xl bg-amber-400 px-3 py-2 text-xs font-bold text-slate-950 disabled:opacity-50" disabled={!item.purchasable || flashPurchaseMutation.isPending} onClick={() => flashPurchaseMutation.mutate(item.flash.id)} type="button">
                      抢购
                    </button>
                  </article>
                ))}
              </section>
            ) : null}
          </>
        )}
      </div>

      <nav className="fixed inset-x-0 bottom-0 z-40 border-t border-white/10 bg-[#0d0f1a]/95 px-1 pb-[calc(6px+env(safe-area-inset-bottom))] pt-2 backdrop-blur-xl">
        <div className="mx-auto grid max-w-[520px] grid-cols-8 gap-0.5">
          {tabs.map((item) => {
            const Icon = item.icon;
            const active = tab === item.key && !selectedCampaignId;
            return (
              <button
                className={`flex flex-col items-center gap-1 rounded-xl px-1 py-1.5 text-[10px] transition ${
                  active ? 'bg-violet-400/20 text-violet-300' : 'text-violet-100/50'
                }`}
                key={item.key}
                onClick={() => {
                  setSelectedCampaignId(null);
                  setTab(item.key);
                }}
                type="button"
              >
                <Icon size={17} />
                {item.label}
              </button>
            );
          })}
        </div>
      </nav>

      {showBoxModal ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-5 backdrop-blur-md">
          <div className="relative w-full max-w-[340px] rounded-[28px] border border-white/15 bg-[linear-gradient(160deg,#1a1040,#0f2027)] p-6 text-center shadow-[0_24px_80px_rgba(0,0,0,0.55)]">
            <button
              className="absolute right-4 top-4 rounded-full bg-white/10 p-2 text-white"
              onClick={() => setShowBoxModal(false)}
              type="button"
            >
              <X size={16} />
            </button>
            {boxAnimating ? (
              <div className="py-10">
                <div className="mx-auto mb-2 flex size-28 animate-bounce items-center justify-center rounded-[32px] bg-[radial-gradient(circle,rgba(167,139,250,0.45),rgba(244,114,182,0.2),transparent_70%)] text-7xl">
                  <Gift size={76} />
                </div>
                <p className="mt-6 text-sm text-violet-100/65">正在打开盲盒...</p>
              </div>
            ) : lastDraw ? (
              <DrawResultView draw={lastDraw} onClose={() => setShowBoxModal(false)} onShare={() => shareMutation.mutate()} />
            ) : drawMutation.error ? (
              <div className="py-8">
                <p className="text-sm text-red-100">{drawMutation.error.message}</p>
              </div>
            ) : null}
          </div>
        </div>
      ) : null}

      {showExchangeModal ? (
        <div className="fixed inset-0 z-50 flex items-end bg-black/70 p-3 backdrop-blur-md sm:items-center sm:justify-center">
          <form
            className="w-full rounded-[28px] border border-white/15 bg-[linear-gradient(160deg,#1a1040,#0f2027)] p-5 shadow-[0_24px_80px_rgba(0,0,0,0.55)] sm:max-w-[360px]"
            onSubmit={exchangeForm.handleSubmit((values) => publishExchangeMutation.mutate(values))}
          >
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-lg font-black text-white">发布交换</h3>
              <button className="rounded-full bg-white/10 p-2" onClick={() => setShowExchangeModal(false)} type="button">
                <X size={16} />
              </button>
            </div>
            <label className="mb-3 block text-sm text-violet-100/70">
              我有（重复款）
              <select
                className="mt-2 w-full rounded-2xl border border-white/10 bg-white px-3 py-3 text-slate-950"
                {...exchangeForm.register('have_prize_id')}
              >
                <option value="">选择重复款</option>
                {duplicateInventory.map((item) => (
                  <option key={item.prize_id} value={item.prize_id}>
                    {item.prize_name}
                  </option>
                ))}
              </select>
            </label>
            <label className="mb-4 block text-sm text-violet-100/70">
              我想要
              <select
                className="mt-2 w-full rounded-2xl border border-white/10 bg-white px-3 py-3 text-slate-950"
                {...exchangeForm.register('want_prize_id')}
              >
                <option value="">选择目标款</option>
                {allPrizes.map((prize) => (
                  <option key={prize.id} value={prize.id}>
                    [{prize.campaign_name}] {prize.name}
                  </option>
                ))}
              </select>
            </label>
            <button
              className="w-full rounded-2xl bg-[linear-gradient(135deg,#f472b6,#a78bfa)] px-4 py-3 font-bold text-white disabled:opacity-50"
              disabled={publishExchangeMutation.isPending}
              type="submit"
            >
              发布
            </button>
            {publishExchangeMutation.error ? (
              <p className="mt-3 rounded-xl border border-red-300/20 bg-red-500/15 px-4 py-3 text-sm text-red-100">
                {publishExchangeMutation.error.message}
              </p>
            ) : null}
          </form>
        </div>
      ) : null}
    </main>
  );
}

function DrawResultView({
  draw,
  onClose,
  onShare,
}: {
  readonly draw: BlindBoxDrawResult;
  readonly onClose: () => void;
  readonly onShare: () => void;
}): React.ReactNode {
  const strongest = [...draw.draws].sort((left, right) => levelScore(right.prize_level) - levelScore(left.prize_level))[0];
  const meta = levelMeta(strongest?.prize_level);

  return (
    <div className="py-2">
      <span className={`inline-flex rounded-full border border-white/10 bg-white/10 px-3 py-1 text-xs font-bold ${meta.className}`}>
        {meta.label}
      </span>
      <div className={`mt-5 text-7xl ${meta.className}`}>{meta.icon}</div>
      <div className={`mt-4 text-2xl font-black ${meta.className}`}>{strongest?.prize_name ?? '谢谢参与'}</div>
      {strongest?.is_new ? <div className="mt-2 text-sm font-bold text-amber-300">NEW!</div> : null}
      {draw.draws.length > 1 ? (
        <div className="mt-5 max-h-36 space-y-2 overflow-y-auto rounded-2xl bg-white/[0.06] p-3 text-left text-xs">
          {draw.draws.map((item) => (
            <div className="flex items-center justify-between gap-2" key={item.record_id}>
              <span className="truncate text-violet-100/75">{item.prize_name}</span>
              <span className={levelMeta(item.prize_level).className}>{item.is_win ? item.prize_level : '未中'}</span>
            </div>
          ))}
        </div>
      ) : null}
      <div className="mt-5 grid grid-cols-2 gap-3">
        <button className="rounded-2xl border border-white/15 bg-white/10 px-4 py-3 text-sm font-bold" onClick={onClose} type="button">
          再来一盒
        </button>
        <button
          className="rounded-2xl bg-[linear-gradient(135deg,#f472b6,#a78bfa)] px-4 py-3 text-sm font-bold text-white"
          onClick={onShare}
          type="button"
        >
          炫耀
        </button>
      </div>
    </div>
  );
}

function levelScore(level: string): number {
  return { limited: 5, S: 5, secret: 4, A: 3, rare: 3, B: 2, common: 1 }[level] ?? 0;
}
