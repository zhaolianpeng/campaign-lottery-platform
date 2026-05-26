'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  BarChart3,
  Boxes,
  ClipboardList,
  Gift,
  Loader2,
  LockKeyhole,
  Package,
  Settings,
  ShoppingBag,
  Truck,
  UserRound,
} from 'lucide-react';
import { useState, type ComponentType } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { apiAssetUrl, apiRequest } from '@/client/api';
import type {
  AdminOverview,
  AdminUserDetail,
  AdminUserListResult,
  BattlePassInfo,
  Campaign,
  CampaignPublishValidation,
  DrawRecord,
  FirstRechargePack,
  FirstRechargePackMutation,
  FulfillmentTask,
  PityConfig,
  Prize,
  ShopItem,
  ShopItemMutation,
  UserStatus,
} from '@/types/api';

type PrizeMutationPayload = {
  readonly name: string;
  readonly level: Prize['level'];
  readonly stock: number;
  readonly probability_weight: number;
  readonly status: Prize['status'];
  readonly image_url?: string;
};

const loginSchema = z.object({
  username: z.string().min(1),
  password: z.string().min(1),
});

type LoginValues = z.infer<typeof loginSchema>;
type AdminTab = 'overview' | 'users' | 'campaigns' | 'prizes' | 'pity' | 'delivery' | 'records' | 'monthcard' | 'shop';

interface AdminLoginPayload {
  readonly token: string;
}

interface AdminTabItem {
  readonly key: AdminTab;
  readonly label: string;
  readonly icon: ComponentType<{ readonly size?: number; readonly className?: string }>;
}

const adminTabs: readonly AdminTabItem[] = [
  { key: 'overview', label: '总览', icon: BarChart3 },
  { key: 'users', label: '用户', icon: UserRound },
  { key: 'campaigns', label: '盲盒', icon: Package },
  { key: 'prizes', label: '奖品', icon: Gift },
  { key: 'pity', label: '概率', icon: Settings },
  { key: 'delivery', label: '发奖', icon: Truck },
  { key: 'records', label: '记录', icon: ClipboardList },
  { key: 'monthcard', label: '月卡', icon: Boxes },
  { key: 'shop', label: '商店', icon: ShoppingBag },
];

function statusClass(status: string): string {
  if (status === 'online' || status === 'win' || status === 'fulfilled' || status === 'active') {
    return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300';
  }
  if (status === 'draft' || status === 'pending' || status === 'pending_phone' || status === 'frozen') {
    return 'border-amber-500/30 bg-amber-500/10 text-amber-300';
  }
  return 'border-red-500/30 bg-red-500/10 text-red-300';
}

function shortId(value: string | undefined): string {
  return value ? value.slice(0, 12) : '-';
}

const prizeLevels: readonly Prize['level'][] = ['common', 'rare', 'secret', 'limited', 'S', 'A', 'B'];
const prizeStatuses: readonly Prize['status'][] = ['active', 'inactive'];
const shopItemTypes: readonly ShopItem['item_type'][] = ['hint_card', 'see_through', 'pity_inherit', 'specify_voucher', 'ten_draw_ticket', 'free_draw'];

type PrizeEditorValues = {
  readonly name: string;
  readonly level: Prize['level'];
  readonly stock: string;
  readonly probability_weight: string;
  readonly status: Prize['status'];
  readonly image_url: string;
};

type PityEditorValues = {
  readonly enabled: boolean;
  readonly soft_pity_n: string;
  readonly pity_factor: string;
  readonly hard_pity_n: string;
  readonly target_prize: string;
  readonly up_pool_enabled: boolean;
  readonly up_prize_id: string;
  readonly up_multiplier: string;
  readonly up_level: Prize['level'];
  readonly up_start_at: string;
  readonly up_end_at: string;
};

type ShopItemEditorValues = {
  readonly name: string;
  readonly description: string;
  readonly image_url: string;
  readonly price_points: string;
  readonly price_cash: string;
  readonly item_type: ShopItem['item_type'];
  readonly item_qty: string;
  readonly stock: string;
  readonly daily_limit: string;
  readonly category: string;
  readonly is_active: boolean;
  readonly expires_at: string;
  readonly sort_order: string;
};

type PackItemEditorValues = {
  readonly type: string;
  readonly name: string;
  readonly qty: string;
};

type FirstRechargeEditorValues = {
  readonly name: string;
  readonly price_points: string;
  readonly cash_price: string;
  readonly description: string;
  readonly image_url: string;
  readonly sort_order: string;
  readonly items: readonly PackItemEditorValues[];
};

function toDatetimeLocalValue(value?: string): string {
  if (!value) {
    return '';
  }
  return value.slice(0, 16);
}

function toIsoValue(value: string): string | undefined {
  return value ? new Date(value).toISOString() : undefined;
}

function createPrizeEditorValues(initial?: Prize): PrizeEditorValues {
  return {
    name: initial?.name ?? '新礼品',
    level: initial?.level ?? 'common',
    stock: String(initial?.stock ?? 100),
    probability_weight: String(initial?.probability_weight ?? 10),
    status: initial?.status ?? 'active',
    image_url: initial?.image_url ?? '',
  };
}

function toPrizePayload(values: PrizeEditorValues): PrizeMutationPayload {
  return {
    name: values.name.trim(),
    level: values.level,
    stock: Number(values.stock),
    probability_weight: Number(values.probability_weight),
    status: values.status,
    image_url: values.image_url.trim() || undefined,
  };
}

function createPityEditorValues(initial?: PityConfig): PityEditorValues {
  return {
    enabled: initial?.enabled ?? true,
    soft_pity_n: String(initial?.soft_pity_n ?? 20),
    pity_factor: String(initial?.pity_factor ?? 0.08),
    hard_pity_n: String(initial?.hard_pity_n ?? 60),
    target_prize: initial?.target_prize ?? '',
    up_pool_enabled: initial?.up_pool_enabled ?? false,
    up_prize_id: initial?.up_prize_id ?? '',
    up_multiplier: String(initial?.up_multiplier ?? 3),
    up_level: initial?.up_level ?? 'secret',
    up_start_at: toDatetimeLocalValue(initial?.up_start_at),
    up_end_at: toDatetimeLocalValue(initial?.up_end_at),
  };
}

function toPityPayload(values: PityEditorValues): PityConfig {
  return {
    enabled: values.enabled,
    soft_pity_n: Number(values.soft_pity_n),
    pity_factor: Number(values.pity_factor),
    hard_pity_n: Number(values.hard_pity_n),
    target_prize: values.target_prize.trim(),
    up_pool_enabled: values.up_pool_enabled,
    up_prize_id: values.up_prize_id.trim() || undefined,
    up_multiplier: Number(values.up_multiplier),
    up_level: values.up_level,
    up_start_at: toIsoValue(values.up_start_at),
    up_end_at: toIsoValue(values.up_end_at),
  };
}

function createShopItemEditorValues(initial?: ShopItem): ShopItemEditorValues {
  return {
    name: initial?.name ?? '新商品',
    description: initial?.description ?? '请填写商品描述',
    image_url: initial?.image_url ?? '',
    price_points: String(initial?.price_points ?? 100),
    price_cash: String(initial?.price_cash ?? 0),
    item_type: initial?.item_type ?? 'hint_card',
    item_qty: String(initial?.item_qty ?? 1),
    stock: String(initial?.stock ?? -1),
    daily_limit: String(initial?.daily_limit ?? 0),
    category: initial?.category ?? 'daily',
    is_active: initial?.is_active ?? true,
    expires_at: toDatetimeLocalValue(initial?.expires_at),
    sort_order: String(initial?.sort_order ?? 1),
  };
}

function toShopItemPayload(values: ShopItemEditorValues): ShopItemMutation {
  return {
    name: values.name.trim(),
    description: values.description.trim(),
    image_url: values.image_url.trim() || undefined,
    price_points: Number(values.price_points),
    price_cash: Number(values.price_cash),
    item_type: values.item_type,
    item_qty: Number(values.item_qty),
    stock: Number(values.stock),
    daily_limit: Number(values.daily_limit),
    category: values.category.trim(),
    is_active: values.is_active,
    expires_at: toIsoValue(values.expires_at),
    sort_order: Number(values.sort_order),
  };
}

function createPackItemEditorValues(item?: FirstRechargePack['items'][number]): PackItemEditorValues {
  return {
    type: item?.type ?? 'points',
    name: item?.name ?? '积分',
    qty: String(item?.qty ?? 60),
  };
}

function createFirstRechargeEditorValues(initial?: FirstRechargePack): FirstRechargeEditorValues {
  return {
    name: initial?.name ?? '新首充礼包',
    price_points: String(initial?.price_points ?? 600),
    cash_price: String(initial?.cash_price ?? 600),
    description: initial?.description ?? '请填写礼包描述',
    image_url: initial?.image_url ?? '',
    sort_order: String(initial?.sort_order ?? 1),
    items: (initial?.items ?? [{ type: 'points', name: '积分', qty: 60 }]).map((item) => createPackItemEditorValues(item)),
  };
}

function toFirstRechargePayload(values: FirstRechargeEditorValues): FirstRechargePackMutation {
  return {
    name: values.name.trim(),
    price_points: Number(values.price_points),
    cash_price: Number(values.cash_price),
    description: values.description.trim(),
    image_url: values.image_url.trim() || undefined,
    sort_order: Number(values.sort_order),
    items: values.items.map((item) => ({
      type: item.type.trim(),
      name: item.name.trim(),
      qty: Number(item.qty),
    })),
  };
}

export function AdminApp(): React.ReactNode {
  const [token, setToken] = useState('');
  const [tab, setTab] = useState<AdminTab>('overview');
  const [selectedCampaignId, setSelectedCampaignId] = useState('');
  const [userKeyword, setUserKeyword] = useState('');
  const [selectedUserId, setSelectedUserId] = useState('');
  const [prizeEditor, setPrizeEditor] = useState<{ readonly prizeId?: string; readonly values: PrizeEditorValues } | null>(null);
  const [pityEditor, setPityEditor] = useState<PityEditorValues | null>(null);
  const [shopItemEditor, setShopItemEditor] = useState<{ readonly itemId?: string; readonly values: ShopItemEditorValues } | null>(null);
  const [firstRechargeEditor, setFirstRechargeEditor] = useState<{ readonly packId?: string; readonly values: FirstRechargeEditorValues } | null>(null);
  const queryClient = useQueryClient();

  const loginForm = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      username: process.env.NEXT_PUBLIC_ADMIN_USER_HINT ?? 'admin',
      password: '',
    },
  });

  const loginMutation = useMutation({
    mutationFn: (values: LoginValues) =>
      apiRequest<AdminLoginPayload>('/api/v1/admin/login', '', {
        method: 'POST',
        body: JSON.stringify(values),
      }),
    onSuccess: (payload) => setToken(payload.token),
  });

  const overviewQuery = useQuery({
    queryKey: ['admin-overview', token],
    queryFn: () => apiRequest<AdminOverview>('/api/v1/admin/overview', token),
    enabled: Boolean(token) && tab === 'overview',
  });

  const campaignsQuery = useQuery({
    queryKey: ['admin-campaigns', token],
    queryFn: () => apiRequest<Campaign[]>('/api/v1/admin/campaigns', token),
    enabled: Boolean(token),
  });

  const usersQuery = useQuery({
    queryKey: ['admin-users', token, userKeyword],
    queryFn: () =>
      apiRequest<AdminUserListResult>(
        `/api/v1/admin/users?page=1&page_size=50${userKeyword ? `&keyword=${encodeURIComponent(userKeyword)}` : ''}`,
        token,
      ),
    enabled: Boolean(token) && tab === 'users',
  });

  const userDetailQuery = useQuery({
    queryKey: ['admin-user-detail', token, selectedUserId],
    queryFn: () => apiRequest<AdminUserDetail>(`/api/v1/admin/users/${selectedUserId}`, token),
    enabled: Boolean(token && selectedUserId) && tab === 'users',
  });

  const prizesQuery = useQuery({
    queryKey: ['admin-prizes', token, selectedCampaignId],
    queryFn: () => apiRequest<Prize[]>(`/api/v1/admin/campaigns/${selectedCampaignId}/prizes`, token),
    enabled: Boolean(token && selectedCampaignId) && tab === 'prizes',
  });

  const deliveryQuery = useQuery({
    queryKey: ['admin-delivery', token],
    queryFn: () => apiRequest<FulfillmentTask[]>('/api/v1/admin/fulfillment-tasks', token),
    enabled: Boolean(token) && tab === 'delivery',
  });

  const recordsQuery = useQuery({
    queryKey: ['admin-records', token],
    queryFn: () => apiRequest<DrawRecord[]>('/api/v1/admin/draw-records', token),
    enabled: Boolean(token) && tab === 'records',
  });

  const battlePassQuery = useQuery({
    queryKey: ['admin-battle-pass', token],
    queryFn: () => apiRequest<BattlePassInfo>('/api/v1/battle-pass/info', token),
    enabled: Boolean(token) && tab === 'monthcard',
  });

  const shopQuery = useQuery({
    queryKey: ['admin-shop-items', token],
    queryFn: () => apiRequest<ShopItem[]>('/api/v1/admin/shop-items', token),
    enabled: Boolean(token) && tab === 'shop',
  });

  const firstRechargeQuery = useQuery({
    queryKey: ['admin-first-recharge', token],
    queryFn: () => apiRequest<FirstRechargePack[]>('/api/v1/admin/first-recharge/packs', token),
    enabled: Boolean(token) && tab === 'shop',
  });

  const createCampaignMutation = useMutation({
    mutationFn: () =>
      apiRequest('/api/v1/admin/campaigns', token, {
        method: 'POST',
        body: JSON.stringify({
          name: `运营盲盒 ${Date.now().toString().slice(-4)}`,
          slug: `box-${Date.now()}`,
          status: 'draft',
          starts_at: new Date().toISOString(),
          ends_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
          daily_draw_limit: 5,
          miss_weight: 70,
          campaign_summary: '管理端快速创建的盲盒草稿，请补充奖品后再上线。',
        }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-campaigns', token] });
      void queryClient.invalidateQueries({ queryKey: ['admin-overview', token] });
    },
  });

  const createPrizeMutation = useMutation({
    mutationFn: (payload: PrizeMutationPayload) =>
      apiRequest(`/api/v1/admin/campaigns/${selectedCampaignId}/prizes`, token, {
        method: 'POST',
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-prizes', token, selectedCampaignId] });
      void queryClient.invalidateQueries({ queryKey: ['admin-campaigns', token] });
    },
  });

  const updatePrizeMutation = useMutation({
    mutationFn: ({ prizeId, payload }: { readonly prizeId: string; readonly payload: PrizeMutationPayload }) =>
      apiRequest(`/api/v1/admin/prizes/${prizeId}`, token, {
        method: 'PUT',
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-prizes', token, selectedCampaignId] });
      void queryClient.invalidateQueries({ queryKey: ['admin-campaigns', token] });
    },
  });

  const deletePrizeMutation = useMutation({
    mutationFn: (prizeId: string) =>
      apiRequest(`/api/v1/admin/prizes/${prizeId}`, token, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-prizes', token, selectedCampaignId] });
      void queryClient.invalidateQueries({ queryKey: ['admin-campaigns', token] });
    },
  });

  const uploadPrizeImageMutation = useMutation({
    mutationFn: (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      return apiRequest<{ readonly url: string }>('/api/v1/admin/uploads/prizes', token, {
        method: 'POST',
        body: formData,
      });
    },
    onSuccess: (payload) => {
      setPrizeEditor((current) => (current ? { ...current, values: { ...current.values, image_url: payload.url } } : current));
    },
  });

  const uploadShopImageMutation = useMutation({
    mutationFn: (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      return apiRequest<{ readonly url: string }>('/api/v1/admin/uploads/prizes', token, {
        method: 'POST',
        body: formData,
      });
    },
    onSuccess: (payload) => {
      setShopItemEditor((current) => (current ? { ...current, values: { ...current.values, image_url: payload.url } } : current));
    },
  });

  const uploadFirstRechargeImageMutation = useMutation({
    mutationFn: (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      return apiRequest<{ readonly url: string }>('/api/v1/admin/uploads/prizes', token, {
        method: 'POST',
        body: formData,
      });
    },
    onSuccess: (payload) => {
      setFirstRechargeEditor((current) => (current ? { ...current, values: { ...current.values, image_url: payload.url } } : current));
    },
  });

  const validateCampaignMutation = useMutation({
    mutationFn: (campaignId: string) =>
      apiRequest<CampaignPublishValidation>(`/api/v1/admin/campaigns/${campaignId}/validate`, token, {
        method: 'POST',
        body: JSON.stringify({}),
      }),
  });

  const savePityMutation = useMutation({
    mutationFn: (payload: PityConfig) =>
      apiRequest(`/api/v1/admin/campaigns/${selectedCampaignId}/pity-config`, token, {
        method: 'PUT',
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-campaigns', token] });
      void queryClient.invalidateQueries({ queryKey: ['admin-overview', token] });
    },
  });

  const createShopItemMutation = useMutation({
    mutationFn: (payload: ShopItemMutation) =>
      apiRequest('/api/v1/admin/shop-items', token, {
        method: 'POST',
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-shop-items', token] });
    },
  });

  const updateShopItemMutation = useMutation({
    mutationFn: ({ itemId, payload }: { readonly itemId: string; readonly payload: ShopItemMutation }) =>
      apiRequest(`/api/v1/admin/shop-items/${itemId}`, token, {
        method: 'PUT',
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-shop-items', token] });
    },
  });

  const deleteShopItemMutation = useMutation({
    mutationFn: (itemId: string) =>
      apiRequest(`/api/v1/admin/shop-items/${itemId}`, token, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-shop-items', token] });
    },
  });

  const createFirstRechargeMutation = useMutation({
    mutationFn: (payload: FirstRechargePackMutation) =>
      apiRequest('/api/v1/admin/first-recharge/packs', token, {
        method: 'POST',
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-first-recharge', token] });
    },
  });

  const updateFirstRechargeMutation = useMutation({
    mutationFn: ({ packId, payload }: { readonly packId: string; readonly payload: FirstRechargePackMutation }) =>
      apiRequest(`/api/v1/admin/first-recharge/packs/${packId}`, token, {
        method: 'PUT',
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-first-recharge', token] });
    },
  });

  const deleteFirstRechargeMutation = useMutation({
    mutationFn: (packId: string) =>
      apiRequest(`/api/v1/admin/first-recharge/packs/${packId}`, token, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-first-recharge', token] });
    },
  });

  const fulfillMutation = useMutation({
    mutationFn: (taskId: number) =>
      apiRequest(`/api/v1/admin/fulfillment-tasks/${taskId}`, token, {
        method: 'PATCH',
        body: JSON.stringify({ status: 'fulfilled', operator_note: '管理端确认发奖' }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-delivery', token] });
      void queryClient.invalidateQueries({ queryKey: ['admin-overview', token] });
    },
  });

  const updateUserStatusMutation = useMutation({
    mutationFn: ({ userId, status }: { readonly userId: string; readonly status: UserStatus }) =>
      apiRequest(`/api/v1/admin/users/${userId}/status`, token, {
        method: 'PATCH',
        body: JSON.stringify({ status, reason: `管理端设置为 ${status}` }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-users', token] });
      void queryClient.invalidateQueries({ queryKey: ['admin-user-detail', token, selectedUserId] });
      void queryClient.invalidateQueries({ queryKey: ['admin-overview', token] });
    },
  });

  const adjustPointsMutation = useMutation({
    mutationFn: ({ userId, points }: { readonly userId: string; readonly points: number }) =>
      apiRequest(`/api/v1/admin/users/${userId}/points-adjust`, token, {
        method: 'POST',
        body: JSON.stringify({ points, reason: '管理端人工调整', remark: 'MVP 后台操作' }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-users', token] });
      void queryClient.invalidateQueries({ queryKey: ['admin-user-detail', token, selectedUserId] });
    },
  });

  const campaigns = campaignsQuery.data ?? [];
  const selectedCampaign = campaigns.find((campaign) => campaign.id === selectedCampaignId);
  const selectedPrizes = prizesQuery.data ?? [];
  const activePrizeCount = selectedPrizes.filter((prize) => prize.status === 'active').length;
  const selectedPrizeStock = selectedPrizes.reduce((sum, prize) => sum + prize.stock, 0);
  const selectedPrizeWeight = selectedPrizes.reduce((sum, prize) => sum + prize.probability_weight, 0);

  function submitPrizeEditor(): void {
    if (!prizeEditor) {
      return;
    }
    if (uploadPrizeImageMutation.isPending) {
      window.alert('图片上传中，请稍后再保存。');
      return;
    }
    const payload = toPrizePayload(prizeEditor.values);
    if (!payload.name || !Number.isInteger(payload.stock) || !Number.isInteger(payload.probability_weight)) {
      window.alert('请完整填写礼品名称、库存和概率权重。');
      return;
    }
    if (prizeEditor.prizeId) {
      updatePrizeMutation.mutate(
        { prizeId: prizeEditor.prizeId, payload },
        { onSuccess: () => setPrizeEditor(null) },
      );
      return;
    }
    createPrizeMutation.mutate(payload, { onSuccess: () => setPrizeEditor(null) });
  }

  function submitPityEditor(): void {
    if (!pityEditor) {
      return;
    }
    const payload = toPityPayload(pityEditor);
    if (!selectedCampaignId || !Number.isInteger(payload.soft_pity_n) || !Number.isInteger(payload.hard_pity_n) || !Number.isFinite(payload.pity_factor)) {
      window.alert('请完整填写保底参数。');
      return;
    }
    savePityMutation.mutate(payload, { onSuccess: () => setPityEditor(null) });
  }

  function submitShopItemEditor(): void {
    if (!shopItemEditor) {
      return;
    }
    if (uploadShopImageMutation.isPending) {
      window.alert('商品图片上传中，请稍后再保存。');
      return;
    }
    const payload = toShopItemPayload(shopItemEditor.values);
    if (!payload.name || !payload.description || !Number.isInteger(payload.price_points) || !Number.isInteger(payload.item_qty)) {
      window.alert('请完整填写商品名称、描述、价格和数量。');
      return;
    }
    if (shopItemEditor.itemId) {
      updateShopItemMutation.mutate(
        { itemId: shopItemEditor.itemId, payload },
        { onSuccess: () => setShopItemEditor(null) },
      );
      return;
    }
    createShopItemMutation.mutate(payload, { onSuccess: () => setShopItemEditor(null) });
  }

  function submitFirstRechargeEditor(): void {
    if (!firstRechargeEditor) {
      return;
    }
    if (uploadFirstRechargeImageMutation.isPending) {
      window.alert('礼包图片上传中，请稍后再保存。');
      return;
    }
    const payload = toFirstRechargePayload(firstRechargeEditor.values);
    if (!payload.name || payload.items.length === 0 || payload.items.some((item) => !item.type || !item.name || !Number.isInteger(item.qty))) {
      window.alert('请完整填写礼包名称与礼包内容。');
      return;
    }
    if (firstRechargeEditor.packId) {
      updateFirstRechargeMutation.mutate(
        { packId: firstRechargeEditor.packId, payload },
        { onSuccess: () => setFirstRechargeEditor(null) },
      );
      return;
    }
    createFirstRechargeMutation.mutate(payload, { onSuccess: () => setFirstRechargeEditor(null) });
  }

  if (!token) {
    return (
      <main className="min-h-screen bg-[#0f0f13] px-4 py-8 text-zinc-100">
        <section className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-[480px] items-center">
          <div className="w-full rounded-2xl border border-zinc-800 bg-[#1a1a24] p-8 shadow-[0_24px_80px_rgba(0,0,0,0.45)]">
            <div className="mb-8 text-center">
              <div className="mx-auto mb-4 flex size-16 items-center justify-center rounded-2xl bg-orange-500/15 text-orange-400">
                <LockKeyhole size={32} />
              </div>
              <h1 className="text-2xl font-black text-white">BOX·MAGIC 管理后台</h1>
              <p className="mt-2 text-sm text-zinc-500">参考原生 Admin 控制台风格重构</p>
            </div>
            <form
              className="space-y-3"
              onSubmit={loginForm.handleSubmit((values) => loginMutation.mutate(values))}
            >
              <input
                className="w-full rounded-lg border border-zinc-700 bg-[#222] px-4 py-3 text-sm text-zinc-100 outline-none focus:border-orange-500"
                placeholder="用户名"
                {...loginForm.register('username')}
              />
              <input
                className="w-full rounded-lg border border-zinc-700 bg-[#222] px-4 py-3 text-sm text-zinc-100 outline-none focus:border-orange-500"
                placeholder="密码"
                type="password"
                {...loginForm.register('password')}
              />
              <button
                className="flex w-full items-center justify-center gap-2 rounded-lg bg-orange-500 px-4 py-3 text-sm font-bold text-white disabled:opacity-60"
                disabled={loginMutation.isPending}
                type="submit"
              >
                {loginMutation.isPending ? <Loader2 className="animate-spin" size={18} /> : <LockKeyhole size={18} />}
                登录
              </button>
              {loginMutation.error ? (
                <p className="rounded-lg border border-red-900/50 bg-red-950/40 px-4 py-3 text-sm text-red-300">
                  登录失败：{loginMutation.error.message}
                </p>
              ) : null}
            </form>
          </div>
        </section>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-[#0f0f13] text-zinc-100">
      <header className="sticky top-0 z-40 border-b border-zinc-800 bg-[#1a1a24]/95 px-4 py-3 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center gap-3">
          <div className="min-w-0 flex-1">
            <div className="truncate font-black text-orange-400">BOX·MAGIC 管理后台</div>
            <div className="text-xs text-zinc-500">已登录 · 后端 API 控制台</div>
          </div>
          <button
            className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300"
            onClick={() => setToken('')}
            type="button"
          >
            退出登录
          </button>
        </div>
      </header>

      <div className="mx-auto max-w-6xl px-4 py-4">
        <nav className="mb-5 flex gap-1 overflow-x-auto border-b border-zinc-800 pb-0 [scrollbar-width:none]">
          {adminTabs.map((item) => {
            const Icon = item.icon;
            return (
              <button
                className={`flex shrink-0 items-center gap-1.5 rounded-t-lg border px-3 py-2 text-sm ${
                  tab === item.key
                    ? 'border-zinc-700 border-b-[#1a1a24] bg-[#1a1a24] text-orange-400'
                    : 'border-transparent text-zinc-500'
                }`}
                key={item.key}
                onClick={() => setTab(item.key)}
                type="button"
              >
                <Icon size={15} />
                {item.label}
              </button>
            );
          })}
        </nav>

        {tab === 'overview' ? (
          <section className="space-y-5">
            <h2 className="text-xl font-black text-white">系统总览</h2>
            <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
              {[
                ['用户数', overviewQuery.data?.total_users ?? 0],
                ['抽奖次数', overviewQuery.data?.total_draws ?? 0],
                ['中奖次数', overviewQuery.data?.total_wins ?? 0],
                ['盲盒数', overviewQuery.data?.campaigns.length ?? 0],
              ].map(([label, value]) => (
                <div className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-4 text-center" key={label}>
                  <div className="text-3xl font-black text-orange-400">{value}</div>
                  <div className="mt-1 text-xs text-zinc-500">{label}</div>
                </div>
              ))}
            </div>
            <AdminList title="盲盒列表">
              {(overviewQuery.data?.campaigns ?? []).map((campaign) => (
                <DataCard
                  badge={campaign.status}
                  badgeClass={statusClass(campaign.status)}
                  key={campaign.id}
                  subtitle={`${campaign.id} · 每日上限 ${campaign.daily_draw_limit} · 未中权重 ${campaign.miss_weight}`}
                  title={campaign.name}
                />
              ))}
            </AdminList>
            <AdminList title="最近抽奖">
              {(overviewQuery.data?.recent_draws ?? []).slice(0, 10).map((record) => (
                <DataCard
                  badge={record.result}
                  badgeClass={statusClass(record.result)}
                  key={record.id}
                  subtitle={`${shortId(record.user_id)} · ${record.drawn_at?.slice(0, 16) ?? ''}`}
                  title={record.prize_name}
                />
              ))}
            </AdminList>
          </section>
        ) : null}

        {tab === 'users' ? (
          <section className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-xl font-black text-white">用户管理</h2>
              <input
                className="w-full rounded-lg border border-zinc-700 bg-[#222] px-3 py-2 text-sm text-zinc-100 outline-none focus:border-orange-500 md:w-72"
                onChange={(event) => setUserKeyword(event.target.value)}
                placeholder="搜索手机号 / 昵称 / 用户ID"
                value={userKeyword}
              />
            </div>

            <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
              <div className="space-y-2">
                {(usersQuery.data?.items ?? []).map((user) => (
                  <DataCard
                    action={
                      <button
                        className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300"
                        onClick={() => setSelectedUserId(user.id)}
                        type="button"
                      >
                        详情
                      </button>
                    }
                    badge={user.status}
                    badgeClass={statusClass(user.status)}
                    key={user.id}
                    subtitle={`${user.mobile?.replace(/^(\d{3})\d{4}(\d{4})$/, '$1****$2') ?? '未绑定手机号'} · ${user.register_source} · ${user.points_balance}积分 · 抽奖${user.total_draws}次`}
                    title={user.nickname}
                  />
                ))}
                {usersQuery.data?.items.length === 0 ? <EmptyAdmin text="暂无匹配用户" /> : null}
              </div>

              <aside className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-4">
                {userDetailQuery.data ? (
                  <div className="space-y-4">
                    <div>
                      <h3 className="text-lg font-black text-white">{userDetailQuery.data.user.nickname}</h3>
                      <p className="mt-1 text-xs text-zinc-500">{userDetailQuery.data.user.id}</p>
                      <p className="mt-2 text-sm text-zinc-300">
                        手机：{userDetailQuery.data.user.mobile?.replace(/^(\d{3})\d{4}(\d{4})$/, '$1****$2') ?? '未绑定'}
                      </p>
                      <p className="text-sm text-zinc-300">状态：{userDetailQuery.data.user.status ?? 'active'}</p>
                    </div>

                    <div className="grid grid-cols-2 gap-2 text-center text-sm">
                      <div className="rounded-lg bg-white/[0.04] p-3">
                        <div className="text-xl font-black text-orange-400">{userDetailQuery.data.member.points}</div>
                        <div className="text-xs text-zinc-500">积分</div>
                      </div>
                      <div className="rounded-lg bg-white/[0.04] p-3">
                        <div className="text-xl font-black text-orange-400">{userDetailQuery.data.member.total_draws}</div>
                        <div className="text-xs text-zinc-500">抽奖次数</div>
                      </div>
                    </div>

                    <div className="flex flex-wrap gap-2">
                      <button
                        className="rounded-md bg-emerald-600 px-3 py-1.5 text-xs font-bold text-white disabled:opacity-50"
                        disabled={updateUserStatusMutation.isPending}
                        onClick={() => updateUserStatusMutation.mutate({ userId: userDetailQuery.data.user.id, status: 'active' })}
                        type="button"
                      >
                        解冻/启用
                      </button>
                      <button
                        className="rounded-md bg-amber-600 px-3 py-1.5 text-xs font-bold text-white disabled:opacity-50"
                        disabled={updateUserStatusMutation.isPending}
                        onClick={() => updateUserStatusMutation.mutate({ userId: userDetailQuery.data.user.id, status: 'frozen' })}
                        type="button"
                      >
                        冻结
                      </button>
                      <button
                        className="rounded-md bg-red-700 px-3 py-1.5 text-xs font-bold text-white disabled:opacity-50"
                        disabled={updateUserStatusMutation.isPending}
                        onClick={() => updateUserStatusMutation.mutate({ userId: userDetailQuery.data.user.id, status: 'disabled' })}
                        type="button"
                      >
                        禁用
                      </button>
                      <button
                        className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-200 disabled:opacity-50"
                        disabled={adjustPointsMutation.isPending}
                        onClick={() => {
                          const value = Number(window.prompt('输入积分调整数，正数增加，负数扣减', '100') ?? 0);
                          if (value) {
                            adjustPointsMutation.mutate({ userId: userDetailQuery.data.user.id, points: value });
                          }
                        }}
                        type="button"
                      >
                        调整积分
                      </button>
                    </div>

                    <AdminList title="最近积分流水">
                      {userDetailQuery.data.points_logs.slice(0, 5).map((log) => (
                        <DataCard
                          badge={log.points > 0 ? `+${log.points}` : `${log.points}`}
                          badgeClass={log.points > 0 ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300' : 'border-red-500/30 bg-red-500/10 text-red-300'}
                          key={log.id}
                          subtitle={`${log.reason} · 余额 ${log.balance}`}
                          title={log.remark}
                        />
                      ))}
                    </AdminList>

                    <AdminList title="状态记录">
                      {userDetailQuery.data.status_logs.slice(0, 5).map((log) => (
                        <DataCard key={log.id} subtitle={log.reason} title={`${log.from_status} -> ${log.to_status}`} />
                      ))}
                    </AdminList>
                  </div>
                ) : (
                  <EmptyAdmin text="选择左侧用户查看详情" />
                )}
              </aside>
            </div>
          </section>
        ) : null}

        {tab === 'campaigns' ? (
          <section className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-xl font-black text-white">盲盒管理</h2>
              <button
                className="rounded-md bg-orange-500 px-3 py-2 text-xs font-bold text-white disabled:opacity-50"
                disabled={createCampaignMutation.isPending}
                onClick={() => createCampaignMutation.mutate()}
                type="button"
              >
                + 新建盲盒
              </button>
            </div>
            <p className="text-sm text-zinc-500">一个盲盒可以包含多个奖品，进入奖品管理后维护该盲盒下的奖品池。</p>
            {campaigns.map((campaign) => (
              <DataCard
                action={
                  <div className="flex flex-wrap gap-2">
                    <button
                      className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300"
                      onClick={() => {
                        setSelectedCampaignId(campaign.id);
                        setTab('prizes');
                      }}
                      type="button"
                    >
                      奖品管理
                    </button>
                    <button
                      className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300 disabled:opacity-50"
                      disabled={validateCampaignMutation.isPending}
                      onClick={() => validateCampaignMutation.mutate(campaign.id)}
                      type="button"
                    >
                      发布校验
                    </button>
                  </div>
                }
                badge={campaign.status}
                badgeClass={statusClass(campaign.status)}
                key={campaign.id}
                subtitle={`${campaign.id} · 每日上限 ${campaign.daily_draw_limit} · 未中权重 ${campaign.miss_weight}`}
                title={campaign.name}
              />
            ))}
            {validateCampaignMutation.data ? (
              <div className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-4">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <h3 className="font-bold text-white">发布校验：{validateCampaignMutation.data.campaign_name}</h3>
                  <span className={`rounded px-2 py-1 text-xs ${validateCampaignMutation.data.can_publish ? statusClass('active') : statusClass('disabled')}`}>
                    {validateCampaignMutation.data.can_publish ? '可发布' : '需修正'}
                  </span>
                </div>
                <p className="mt-2 text-sm text-zinc-400">
                  奖品 {validateCampaignMutation.data.prize_count} 个 · 上架 {validateCampaignMutation.data.active_prize_count} 个 · 总库存{' '}
                  {validateCampaignMutation.data.total_stock} · 权重合计 {validateCampaignMutation.data.total_weight}
                </p>
                {validateCampaignMutation.data.errors.length > 0 ? (
                  <div className="mt-3 text-sm text-red-300">{validateCampaignMutation.data.errors.join('；')}</div>
                ) : null}
                {validateCampaignMutation.data.warnings.length > 0 ? (
                  <div className="mt-2 text-sm text-amber-300">{validateCampaignMutation.data.warnings.join('；')}</div>
                ) : null}
              </div>
            ) : null}
          </section>
        ) : null}

        {tab === 'prizes' ? (
          <section className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="text-xs font-bold text-orange-400">盲盒管理 / 奖品管理</div>
                <h2 className="mt-1 text-xl font-black text-white">奖品管理</h2>
              </div>
              <button
                className="rounded-md bg-orange-500 px-3 py-2 text-xs font-bold text-white disabled:opacity-50"
                disabled={!selectedCampaignId || createPrizeMutation.isPending}
                onClick={() => setPrizeEditor({ values: createPrizeEditorValues() })}
                type="button"
              >
                + 新建奖品
              </button>
            </div>
            <select
              className="w-full rounded-lg border border-zinc-700 bg-[#222] px-3 py-3 text-sm text-zinc-100 outline-none focus:border-orange-500 md:max-w-md"
              onChange={(event) => setSelectedCampaignId(event.target.value)}
              value={selectedCampaignId}
            >
              <option value="">-- 选择盲盒 --</option>
              {campaigns.map((campaign) => (
                <option key={campaign.id} value={campaign.id}>
                  {campaign.name}
                </option>
              ))}
            </select>
            {selectedCampaignId ? (
              <div className="space-y-3">
                {selectedCampaign ? (
                  <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
                    {[
                      ['所属盲盒', selectedCampaign.name],
                      ['奖品数量', selectedPrizes.length],
                      ['上架奖品', activePrizeCount],
                      ['总库存', selectedPrizeStock],
                      ['权重合计', selectedPrizeWeight],
                    ].map(([label, value]) => (
                      <div className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-3" key={label}>
                        <div className="text-lg font-black text-orange-400">{value}</div>
                        <div className="mt-1 text-xs text-zinc-500">{label}</div>
                      </div>
                    ))}
                  </div>
                ) : null}
                {selectedPrizes.map((prize) => (
                  <DataCard
                    action={
                      <div className="flex gap-2">
                        <button
                          className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-200 disabled:opacity-50"
                          disabled={updatePrizeMutation.isPending}
                          onClick={() => setPrizeEditor({ prizeId: prize.id, values: createPrizeEditorValues(prize) })}
                          type="button"
                        >
                          编辑
                        </button>
                        <button
                          className="rounded-md border border-red-800 px-3 py-1.5 text-xs text-red-300 disabled:opacity-50"
                          disabled={deletePrizeMutation.isPending}
                          onClick={() => {
                            if (window.confirm(`确认删除礼品「${prize.name}」吗？`)) {
                              deletePrizeMutation.mutate(prize.id);
                            }
                          }}
                          type="button"
                        >
                          删除
                        </button>
                      </div>
                    }
                    badge={prize.level}
                    badgeClass="border-blue-500/30 bg-blue-500/10 text-blue-300"
                    coverUrl={prize.image_url}
                    key={prize.id}
                    subtitle={`ID: ${prize.id} · 库存 ${prize.stock} · 权重 ${prize.probability_weight} · ${prize.status}${prize.sort_order ? ` · 排序 ${prize.sort_order}` : ''}`}
                    title={prize.name}
                  />
                ))}
                {selectedPrizes.length === 0 ? <EmptyAdmin text="当前盲盒暂无奖品" /> : null}
              </div>
            ) : (
              <EmptyAdmin text="请先选择一个盲盒" />
            )}
          </section>
        ) : null}

        {tab === 'pity' ? (
          <section className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-xl font-black text-white">概率 / UP 池配置</h2>
              <button
                className="rounded-md bg-orange-500 px-3 py-2 text-xs font-bold text-white disabled:opacity-50"
                disabled={!selectedCampaignId || savePityMutation.isPending}
                onClick={() => setPityEditor(createPityEditorValues(selectedCampaign?.pity_config))}
                type="button"
              >
                编辑并保存配置
              </button>
            </div>
            <select
              className="w-full rounded-lg border border-zinc-700 bg-[#222] px-3 py-3 text-sm text-zinc-100 outline-none focus:border-orange-500 md:max-w-md"
              onChange={(event) => setSelectedCampaignId(event.target.value)}
              value={selectedCampaignId}
            >
              <option value="">-- 选择盲盒 --</option>
              {campaigns.map((campaign) => (
                <option key={campaign.id} value={campaign.id}>
                  {campaign.name}
                </option>
              ))}
            </select>
            <div className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-4">
              <h3 className="mb-2 font-bold text-orange-400">配置说明</h3>
              {selectedCampaignId ? (
                <div className="space-y-2 text-sm leading-6 text-zinc-400">
                  <p>当前活动：{selectedCampaign?.name ?? '-'}</p>
                  <p>软保底：{selectedCampaign?.pity_config?.soft_pity_n ?? 0}</p>
                  <p>硬保底：{selectedCampaign?.pity_config?.hard_pity_n ?? 0}</p>
                  <p>概率递增：{selectedCampaign?.pity_config?.pity_factor ?? 0}</p>
                  <p>目标奖品：{selectedCampaign?.pity_config?.target_prize ?? '未配置'}</p>
                </div>
              ) : (
                <p className="text-sm leading-6 text-zinc-400">先选择活动，再录入软保底、硬保底、UP 奖品和时间窗口。</p>
              )}
            </div>
          </section>
        ) : null}

        {tab === 'delivery' ? (
          <section className="space-y-4">
            <h2 className="text-xl font-black text-white">发奖管理</h2>
            {(deliveryQuery.data ?? []).map((task) => (
              <DataCard
                action={
                  <button
                    className="rounded-md bg-orange-500 px-3 py-1.5 text-xs font-bold text-white disabled:opacity-50"
                    disabled={task.status === 'fulfilled' || fulfillMutation.isPending}
                    onClick={() => fulfillMutation.mutate(task.id)}
                    type="button"
                  >
                    审核通过
                  </button>
                }
                badge={task.status}
                badgeClass={statusClass(task.status)}
                key={task.id}
                subtitle={`用户 ${shortId(task.user_id)} · 奖品 ${task.prize_id}`}
                title={`任务 #${task.id}`}
              />
            ))}
            {deliveryQuery.data?.length === 0 ? <EmptyAdmin text="暂无待发奖记录" /> : null}
          </section>
        ) : null}

        {tab === 'records' ? (
          <section className="space-y-4">
            <h2 className="text-xl font-black text-white">抽奖记录</h2>
            <div className="overflow-x-auto rounded-xl border border-zinc-800 bg-[#1a1a24]">
              <table className="w-full min-w-[640px] border-collapse text-sm">
                <thead className="bg-white/[0.04] text-left text-xs text-zinc-500">
                  <tr>
                    <th className="px-3 py-3">ID</th>
                    <th className="px-3 py-3">用户</th>
                    <th className="px-3 py-3">奖品</th>
                    <th className="px-3 py-3">结果</th>
                    <th className="px-3 py-3">时间</th>
                  </tr>
                </thead>
                <tbody>
                  {(recordsQuery.data ?? []).slice(0, 50).map((record) => (
                    <tr className="border-t border-zinc-800" key={record.id}>
                      <td className="px-3 py-3 text-zinc-400">{shortId(record.id)}</td>
                      <td className="px-3 py-3 text-zinc-400">{shortId(record.user_id)}</td>
                      <td className="px-3 py-3 text-zinc-100">{record.prize_name}</td>
                      <td className="px-3 py-3">
                        <span className={`rounded px-2 py-1 text-xs ${statusClass(record.result)}`}>{record.result}</span>
                      </td>
                      <td className="px-3 py-3 text-zinc-500">{record.drawn_at?.slice(0, 16) ?? ''}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        ) : null}

        {tab === 'monthcard' ? (
          <section className="space-y-4">
            <h2 className="text-xl font-black text-white">月卡 / 战令</h2>
            <div className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-4">
              <h3 className="font-bold text-orange-400">当前赛季</h3>
              <p className="mt-2 text-sm text-zinc-400">
                {battlePassQuery.data?.season?.name ?? '暂无赛季'} · 满级 {battlePassQuery.data?.season?.max_level ?? 0}
              </p>
            </div>
            <AdminList title="战令任务">
              {(battlePassQuery.data?.tasks ?? []).map((task) => (
                <DataCard key={task.id} subtitle={`${task.description} · ${task.xp_reward} XP`} title={task.name} />
              ))}
            </AdminList>
          </section>
        ) : null}
        {tab === 'shop' ? (
          <section className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-xl font-black text-white">商店管理</h2>
              <div className="flex gap-2">
                <button
                  className="rounded-md bg-orange-500 px-3 py-2 text-xs font-bold text-white disabled:opacity-50"
                  disabled={createShopItemMutation.isPending}
                  onClick={() => setShopItemEditor({ values: createShopItemEditorValues() })}
                  type="button"
                >
                  + 新建商品
                </button>
                <button
                  className="rounded-md bg-zinc-700 px-3 py-2 text-xs font-bold text-white disabled:opacity-50"
                  disabled={createFirstRechargeMutation.isPending}
                  onClick={() => setFirstRechargeEditor({ values: createFirstRechargeEditorValues() })}
                  type="button"
                >
                  + 新建首充礼包
                </button>
              </div>
            </div>
            <AdminList title="商品列表">
              {(shopQuery.data ?? []).map((item) => (
                <DataCard
                  action={
                    <div className="flex gap-2">
                      <button
                        className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-200 disabled:opacity-50"
                        disabled={updateShopItemMutation.isPending}
                        onClick={() => setShopItemEditor({ itemId: item.id, values: createShopItemEditorValues(item) })}
                        type="button"
                      >
                        编辑
                      </button>
                      <button
                        className="rounded-md border border-red-800 px-3 py-1.5 text-xs text-red-300 disabled:opacity-50"
                        disabled={deleteShopItemMutation.isPending}
                        onClick={() => {
                          if (window.confirm(`确认删除商品「${item.name}」吗？`)) {
                            deleteShopItemMutation.mutate(item.id);
                          }
                        }}
                        type="button"
                      >
                        删除
                      </button>
                    </div>
                  }
                  badge={`${item.price_points}积分`}
                  badgeClass="border-amber-500/30 bg-amber-500/10 text-amber-300"
                  coverUrl={item.image_url}
                  key={item.id}
                  subtitle={`${item.description} · 库存 ${item.stock < 0 ? '不限' : item.stock}`}
                  title={item.name}
                />
              ))}
            </AdminList>
            <AdminList title="首充礼包">
              {(firstRechargeQuery.data ?? []).map((pack) => (
                <DataCard
                  action={
                    <div className="flex gap-2">
                      <button
                        className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-200 disabled:opacity-50"
                        disabled={updateFirstRechargeMutation.isPending}
                        onClick={() => setFirstRechargeEditor({ packId: pack.id, values: createFirstRechargeEditorValues(pack) })}
                        type="button"
                      >
                        编辑
                      </button>
                      <button
                        className="rounded-md border border-red-800 px-3 py-1.5 text-xs text-red-300 disabled:opacity-50"
                        disabled={deleteFirstRechargeMutation.isPending}
                        onClick={() => {
                          if (window.confirm(`确认删除礼包「${pack.name}」吗？`)) {
                            deleteFirstRechargeMutation.mutate(pack.id);
                          }
                        }}
                        type="button"
                      >
                        删除
                      </button>
                    </div>
                  }
                  coverUrl={pack.image_url}
                  key={pack.id}
                  subtitle={`${pack.description} · ${pack.price_points}积分 / ¥${(pack.cash_price / 100).toFixed(2)} · ${pack.items.map((item) => `${item.name}x${item.qty}`).join('、')}`}
                  title={pack.name}
                />
              ))}
            </AdminList>
          </section>
        ) : null}

        <AdminModal
          isOpen={Boolean(prizeEditor)}
          isSubmitting={createPrizeMutation.isPending || updatePrizeMutation.isPending || uploadPrizeImageMutation.isPending}
          onClose={() => setPrizeEditor(null)}
          onSubmit={submitPrizeEditor}
          submitText={prizeEditor?.prizeId ? '保存礼品' : '创建礼品'}
          title={prizeEditor?.prizeId ? '编辑礼品' : '新建礼品'}
        >
          {prizeEditor ? (
            <div className="grid gap-3 md:grid-cols-2">
              <Field label="礼品名称">
                <input className="admin-input" value={prizeEditor.values.name} onChange={(event) => setPrizeEditor((current) => current ? { ...current, values: { ...current.values, name: event.target.value } } : current)} />
              </Field>
              <Field label="礼品等级">
                <select className="admin-input" value={prizeEditor.values.level} onChange={(event) => setPrizeEditor((current) => current ? { ...current, values: { ...current.values, level: event.target.value as Prize['level'] } } : current)}>
                  {prizeLevels.map((level) => <option key={level} value={level}>{level}</option>)}
                </select>
              </Field>
              <Field label="库存">
                <input className="admin-input" min={0} type="number" value={prizeEditor.values.stock} onChange={(event) => setPrizeEditor((current) => current ? { ...current, values: { ...current.values, stock: event.target.value } } : current)} />
              </Field>
              <Field label="概率权重">
                <input className="admin-input" min={0} type="number" value={prizeEditor.values.probability_weight} onChange={(event) => setPrizeEditor((current) => current ? { ...current, values: { ...current.values, probability_weight: event.target.value } } : current)} />
              </Field>
              <Field label="状态">
                <select className="admin-input" value={prizeEditor.values.status} onChange={(event) => setPrizeEditor((current) => current ? { ...current, values: { ...current.values, status: event.target.value as Prize['status'] } } : current)}>
                  {prizeStatuses.map((status) => <option key={status} value={status}>{status}</option>)}
                </select>
              </Field>
              <Field label="礼品图片 URL">
                <input className="admin-input" placeholder="/api/v1/uploads/prizes/..." value={prizeEditor.values.image_url} onChange={(event) => setPrizeEditor((current) => current ? { ...current, values: { ...current.values, image_url: event.target.value } } : current)} />
              </Field>
              <Field label="上传礼品图片">
                <div className="space-y-3">
                  <label className="inline-flex cursor-pointer items-center rounded-md border border-zinc-700 px-3 py-2 text-xs font-semibold text-zinc-200">
                    {uploadPrizeImageMutation.isPending ? '上传中...' : '选择图片'}
                    <input
                      accept="image/png,image/jpeg,image/webp,image/gif"
                      className="hidden"
                      onChange={(event) => {
                        const file = event.target.files?.[0];
                        if (file) {
                          uploadPrizeImageMutation.mutate(file);
                        }
                        event.target.value = '';
                      }}
                      type="file"
                    />
                  </label>
                  {prizeEditor.values.image_url ? (
                    <img alt={prizeEditor.values.name} className="h-36 w-full rounded-xl border border-zinc-800 object-cover" src={apiAssetUrl(prizeEditor.values.image_url)} />
                  ) : (
                    <div className="flex h-36 items-center justify-center rounded-xl border border-dashed border-zinc-800 bg-black/10 text-xs text-zinc-500">
                      暂无礼品图片
                    </div>
                  )}
                  {uploadPrizeImageMutation.error ? (
                    <p className="text-xs text-red-300">上传失败：{uploadPrizeImageMutation.error.message}</p>
                  ) : null}
                </div>
              </Field>
            </div>
          ) : null}
        </AdminModal>

        <AdminModal
          isOpen={Boolean(pityEditor)}
          isSubmitting={savePityMutation.isPending}
          onClose={() => setPityEditor(null)}
          onSubmit={submitPityEditor}
          submitText="保存配置"
          title="编辑保底 / UP 池"
        >
          {pityEditor ? (
            <div className="grid gap-3 md:grid-cols-2">
              <ToggleField checked={pityEditor.enabled} label="启用保底" onChange={(checked) => setPityEditor((current) => current ? { ...current, enabled: checked } : current)} />
              <ToggleField checked={pityEditor.up_pool_enabled} label="启用 UP 池" onChange={(checked) => setPityEditor((current) => current ? { ...current, up_pool_enabled: checked } : current)} />
              <Field label="软保底次数"><input className="admin-input" min={0} type="number" value={pityEditor.soft_pity_n} onChange={(event) => setPityEditor((current) => current ? { ...current, soft_pity_n: event.target.value } : current)} /></Field>
              <Field label="硬保底次数"><input className="admin-input" min={0} type="number" value={pityEditor.hard_pity_n} onChange={(event) => setPityEditor((current) => current ? { ...current, hard_pity_n: event.target.value } : current)} /></Field>
              <Field label="概率递增系数"><input className="admin-input" min={0} step="0.01" type="number" value={pityEditor.pity_factor} onChange={(event) => setPityEditor((current) => current ? { ...current, pity_factor: event.target.value } : current)} /></Field>
              <Field label="目标奖品 ID"><input className="admin-input" value={pityEditor.target_prize} onChange={(event) => setPityEditor((current) => current ? { ...current, target_prize: event.target.value } : current)} /></Field>
              <Field label="UP 奖品 ID"><input className="admin-input" value={pityEditor.up_prize_id} onChange={(event) => setPityEditor((current) => current ? { ...current, up_prize_id: event.target.value } : current)} /></Field>
              <Field label="UP 倍率"><input className="admin-input" min={0} step="0.1" type="number" value={pityEditor.up_multiplier} onChange={(event) => setPityEditor((current) => current ? { ...current, up_multiplier: event.target.value } : current)} /></Field>
              <Field label="UP 等级">
                <select className="admin-input" value={pityEditor.up_level} onChange={(event) => setPityEditor((current) => current ? { ...current, up_level: event.target.value as Prize['level'] } : current)}>
                  {prizeLevels.map((level) => <option key={level} value={level}>{level}</option>)}
                </select>
              </Field>
              <Field label="UP 开始时间"><input className="admin-input" type="datetime-local" value={pityEditor.up_start_at} onChange={(event) => setPityEditor((current) => current ? { ...current, up_start_at: event.target.value } : current)} /></Field>
              <Field label="UP 结束时间"><input className="admin-input" type="datetime-local" value={pityEditor.up_end_at} onChange={(event) => setPityEditor((current) => current ? { ...current, up_end_at: event.target.value } : current)} /></Field>
            </div>
          ) : null}
        </AdminModal>

        <AdminModal
          isOpen={Boolean(shopItemEditor)}
          isSubmitting={createShopItemMutation.isPending || updateShopItemMutation.isPending || uploadShopImageMutation.isPending}
          onClose={() => setShopItemEditor(null)}
          onSubmit={submitShopItemEditor}
          submitText={shopItemEditor?.itemId ? '保存商品' : '创建商品'}
          title={shopItemEditor?.itemId ? '编辑商品' : '新建商品'}
        >
          {shopItemEditor ? (
            <div className="grid gap-3 md:grid-cols-2">
              <Field label="商品名称"><input className="admin-input" value={shopItemEditor.values.name} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, name: event.target.value } } : current)} /></Field>
              <Field label="分类"><input className="admin-input" value={shopItemEditor.values.category} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, category: event.target.value } } : current)} /></Field>
              <Field label="商品描述"><textarea className="admin-input min-h-24" value={shopItemEditor.values.description} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, description: event.target.value } } : current)} /></Field>
              <Field label="商品图片 URL"><input className="admin-input" placeholder="/api/v1/uploads/prizes/..." value={shopItemEditor.values.image_url} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, image_url: event.target.value } } : current)} /></Field>
              <Field label="道具类型"><select className="admin-input" value={shopItemEditor.values.item_type} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, item_type: event.target.value as ShopItem['item_type'] } } : current)}>{shopItemTypes.map((itemType) => <option key={itemType} value={itemType}>{itemType}</option>)}</select></Field>
              <Field label="积分价格"><input className="admin-input" min={0} type="number" value={shopItemEditor.values.price_points} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, price_points: event.target.value } } : current)} /></Field>
              <Field label="现金价格（分）"><input className="admin-input" min={0} type="number" value={shopItemEditor.values.price_cash} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, price_cash: event.target.value } } : current)} /></Field>
              <Field label="发放数量"><input className="admin-input" min={1} type="number" value={shopItemEditor.values.item_qty} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, item_qty: event.target.value } } : current)} /></Field>
              <Field label="库存"><input className="admin-input" type="number" value={shopItemEditor.values.stock} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, stock: event.target.value } } : current)} /></Field>
              <Field label="每日限购"><input className="admin-input" min={0} type="number" value={shopItemEditor.values.daily_limit} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, daily_limit: event.target.value } } : current)} /></Field>
              <Field label="排序值"><input className="admin-input" min={0} type="number" value={shopItemEditor.values.sort_order} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, sort_order: event.target.value } } : current)} /></Field>
              <Field label="失效时间"><input className="admin-input" type="datetime-local" value={shopItemEditor.values.expires_at} onChange={(event) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, expires_at: event.target.value } } : current)} /></Field>
              <Field label="上传商品图片">
                <div className="space-y-3">
                  <label className="inline-flex cursor-pointer items-center rounded-md border border-zinc-700 px-3 py-2 text-xs font-semibold text-zinc-200">
                    {uploadShopImageMutation.isPending ? '上传中...' : '选择图片'}
                    <input accept="image/png,image/jpeg,image/webp,image/gif" className="hidden" onChange={(event) => {
                      const file = event.target.files?.[0];
                      if (file) {
                        uploadShopImageMutation.mutate(file);
                      }
                      event.target.value = '';
                    }} type="file" />
                  </label>
                  {shopItemEditor.values.image_url ? <img alt={shopItemEditor.values.name} className="h-36 w-full rounded-xl border border-zinc-800 object-cover" src={apiAssetUrl(shopItemEditor.values.image_url)} /> : <div className="flex h-36 items-center justify-center rounded-xl border border-dashed border-zinc-800 bg-black/10 text-xs text-zinc-500">暂无商品图片</div>}
                  {uploadShopImageMutation.error ? <p className="text-xs text-red-300">上传失败：{uploadShopImageMutation.error.message}</p> : null}
                </div>
              </Field>
              <ToggleField checked={shopItemEditor.values.is_active} label="上架状态" onChange={(checked) => setShopItemEditor((current) => current ? { ...current, values: { ...current.values, is_active: checked } } : current)} />
            </div>
          ) : null}
        </AdminModal>

        <AdminModal
          isOpen={Boolean(firstRechargeEditor)}
          isSubmitting={createFirstRechargeMutation.isPending || updateFirstRechargeMutation.isPending || uploadFirstRechargeImageMutation.isPending}
          onClose={() => setFirstRechargeEditor(null)}
          onSubmit={submitFirstRechargeEditor}
          submitText={firstRechargeEditor?.packId ? '保存礼包' : '创建礼包'}
          title={firstRechargeEditor?.packId ? '编辑首充礼包' : '新建首充礼包'}
        >
          {firstRechargeEditor ? (
            <div className="space-y-4">
              <div className="grid gap-3 md:grid-cols-2">
                <Field label="礼包名称"><input className="admin-input" value={firstRechargeEditor.values.name} onChange={(event) => setFirstRechargeEditor((current) => current ? { ...current, values: { ...current.values, name: event.target.value } } : current)} /></Field>
                <Field label="排序值"><input className="admin-input" min={0} type="number" value={firstRechargeEditor.values.sort_order} onChange={(event) => setFirstRechargeEditor((current) => current ? { ...current, values: { ...current.values, sort_order: event.target.value } } : current)} /></Field>
                <Field label="积分价格"><input className="admin-input" min={0} type="number" value={firstRechargeEditor.values.price_points} onChange={(event) => setFirstRechargeEditor((current) => current ? { ...current, values: { ...current.values, price_points: event.target.value } } : current)} /></Field>
                <Field label="现金价格（分）"><input className="admin-input" min={0} type="number" value={firstRechargeEditor.values.cash_price} onChange={(event) => setFirstRechargeEditor((current) => current ? { ...current, values: { ...current.values, cash_price: event.target.value } } : current)} /></Field>
              </div>
              <Field label="礼包描述"><textarea className="admin-input min-h-24" value={firstRechargeEditor.values.description} onChange={(event) => setFirstRechargeEditor((current) => current ? { ...current, values: { ...current.values, description: event.target.value } } : current)} /></Field>
              <Field label="礼包图片 URL"><input className="admin-input" placeholder="/api/v1/uploads/prizes/..." value={firstRechargeEditor.values.image_url} onChange={(event) => setFirstRechargeEditor((current) => current ? { ...current, values: { ...current.values, image_url: event.target.value } } : current)} /></Field>
              <Field label="上传礼包图片">
                <div className="space-y-3">
                  <label className="inline-flex cursor-pointer items-center rounded-md border border-zinc-700 px-3 py-2 text-xs font-semibold text-zinc-200">
                    {uploadFirstRechargeImageMutation.isPending ? '上传中...' : '选择图片'}
                    <input accept="image/png,image/jpeg,image/webp,image/gif" className="hidden" onChange={(event) => {
                      const file = event.target.files?.[0];
                      if (file) {
                        uploadFirstRechargeImageMutation.mutate(file);
                      }
                      event.target.value = '';
                    }} type="file" />
                  </label>
                  {firstRechargeEditor.values.image_url ? <img alt={firstRechargeEditor.values.name} className="h-36 w-full rounded-xl border border-zinc-800 object-cover" src={apiAssetUrl(firstRechargeEditor.values.image_url)} /> : <div className="flex h-36 items-center justify-center rounded-xl border border-dashed border-zinc-800 bg-black/10 text-xs text-zinc-500">暂无礼包图片</div>}
                  {uploadFirstRechargeImageMutation.error ? <p className="text-xs text-red-300">上传失败：{uploadFirstRechargeImageMutation.error.message}</p> : null}
                </div>
              </Field>
              <div className="space-y-3 rounded-xl border border-zinc-800 bg-black/10 p-3">
                <div className="flex items-center justify-between gap-3">
                  <h4 className="font-semibold text-white">礼包内容</h4>
                  <button className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-200" onClick={() => setFirstRechargeEditor((current) => current ? { ...current, values: { ...current.values, items: [...current.values.items, createPackItemEditorValues()] } } : current)} type="button">+ 添加条目</button>
                </div>
                {firstRechargeEditor.values.items.map((item, index) => (
                  <div className="grid gap-3 rounded-lg border border-zinc-800 bg-[#16161d] p-3 md:grid-cols-[1fr_1fr_120px_auto]" key={`${item.type}-${index}`}>
                    <input className="admin-input" placeholder="类型" value={item.type} onChange={(event) => setFirstRechargeEditor((current) => current ? { ...current, values: { ...current.values, items: current.values.items.map((currentItem, currentIndex) => currentIndex === index ? { ...currentItem, type: event.target.value } : currentItem) } } : current)} />
                    <input className="admin-input" placeholder="名称" value={item.name} onChange={(event) => setFirstRechargeEditor((current) => current ? { ...current, values: { ...current.values, items: current.values.items.map((currentItem, currentIndex) => currentIndex === index ? { ...currentItem, name: event.target.value } : currentItem) } } : current)} />
                    <input className="admin-input" min={1} placeholder="数量" type="number" value={item.qty} onChange={(event) => setFirstRechargeEditor((current) => current ? { ...current, values: { ...current.values, items: current.values.items.map((currentItem, currentIndex) => currentIndex === index ? { ...currentItem, qty: event.target.value } : currentItem) } } : current)} />
                    <button className="rounded-md border border-red-800 px-3 py-2 text-xs text-red-300" disabled={firstRechargeEditor.values.items.length === 1} onClick={() => setFirstRechargeEditor((current) => current ? { ...current, values: { ...current.values, items: current.values.items.filter((_, currentIndex) => currentIndex !== index) } } : current)} type="button">删除</button>
                  </div>
                ))}
              </div>
            </div>
          ) : null}
        </AdminModal>
      </div>
      <AdminEditorStyles />
    </main>
  );
}

function AdminList({ title, children }: { readonly title: string; readonly children: React.ReactNode }): React.ReactNode {
  return (
    <section>
      <h3 className="mb-3 font-bold text-white">{title}</h3>
      <div className="space-y-2">{children}</div>
    </section>
  );
}

function DataCard({
  title,
  subtitle,
  badge,
  badgeClass,
  action,
  coverUrl,
}: {
  readonly title: string;
  readonly subtitle: string;
  readonly badge?: string;
  readonly badgeClass?: string;
  readonly action?: React.ReactNode;
  readonly coverUrl?: string;
}): React.ReactNode {
  return (
    <article className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 flex-1 items-center gap-3">
          {coverUrl ? (
            <img alt={title} className="h-14 w-14 shrink-0 rounded-xl border border-zinc-800 object-cover" src={apiAssetUrl(coverUrl)} />
          ) : null}
          <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="truncate font-semibold text-zinc-100">{title}</h3>
            {badge ? <span className={`rounded border px-2 py-0.5 text-[11px] font-bold ${badgeClass ?? ''}`}>{badge}</span> : null}
          </div>
          <p className="mt-1 text-xs text-zinc-500">{subtitle}</p>
          </div>
        </div>
        {action}
      </div>
    </article>
  );
}

function EmptyAdmin({ text }: { readonly text: string }): React.ReactNode {
  return <div className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-8 text-center text-sm text-zinc-500">{text}</div>;
}

function AdminModal({
  isOpen,
  title,
  children,
  submitText,
  isSubmitting,
  onClose,
  onSubmit,
}: {
  readonly isOpen: boolean;
  readonly title: string;
  readonly children: React.ReactNode;
  readonly submitText: string;
  readonly isSubmitting: boolean;
  readonly onClose: () => void;
  readonly onSubmit: () => void;
}): React.ReactNode {
  if (!isOpen) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4 py-8 backdrop-blur-sm">
      <div className="max-h-[90vh] w-full max-w-3xl overflow-y-auto rounded-2xl border border-zinc-800 bg-[#15151c] shadow-[0_32px_120px_rgba(0,0,0,0.55)]">
        <div className="flex items-center justify-between gap-3 border-b border-zinc-800 px-5 py-4">
          <h3 className="text-lg font-black text-white">{title}</h3>
          <button className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300" onClick={onClose} type="button">
            关闭
          </button>
        </div>
        <div className="space-y-4 px-5 py-5">{children}</div>
        <div className="flex justify-end gap-3 border-t border-zinc-800 px-5 py-4">
          <button className="rounded-md border border-zinc-700 px-4 py-2 text-sm text-zinc-300" onClick={onClose} type="button">
            取消
          </button>
          <button className="rounded-md bg-orange-500 px-4 py-2 text-sm font-bold text-white disabled:opacity-50" disabled={isSubmitting} onClick={onSubmit} type="button">
            {isSubmitting ? '提交中...' : submitText}
          </button>
        </div>
      </div>
    </div>
  );
}

function Field({ label, children }: { readonly label: string; readonly children: React.ReactNode }): React.ReactNode {
  return (
    <label className="block space-y-2 text-sm text-zinc-300">
      <span className="text-xs font-semibold uppercase tracking-[0.12em] text-zinc-500">{label}</span>
      {children}
    </label>
  );
}

function ToggleField({
  label,
  checked,
  onChange,
}: {
  readonly label: string;
  readonly checked: boolean;
  readonly onChange: (checked: boolean) => void;
}): React.ReactNode {
  return (
    <label className="flex min-h-[44px] items-center justify-between rounded-lg border border-zinc-800 bg-black/10 px-3 py-2 text-sm text-zinc-300">
      <span>{label}</span>
      <input checked={checked} className="size-4 accent-orange-500" onChange={(event) => onChange(event.target.checked)} type="checkbox" />
    </label>
  );
}

function AdminEditorStyles(): React.ReactNode {
  return (
    <style jsx global>{`
      .admin-input {
        width: 100%;
        border-radius: 0.75rem;
        border: 1px solid rgb(63 63 70 / 1);
        background: #222;
        color: rgb(244 244 245 / 1);
        padding: 0.7rem 0.85rem;
        outline: none;
      }

      .admin-input:focus {
        border-color: rgb(249 115 22 / 1);
      }

      .admin-input::placeholder {
        color: rgb(113 113 122 / 1);
      }
    `}</style>
  );
}

