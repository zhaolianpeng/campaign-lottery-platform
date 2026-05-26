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
import { apiRequest } from '@/client/api';
import type {
  AdminOverview,
  AdminUserDetail,
  AdminUserListResult,
  BattlePassInfo,
  Campaign,
  DrawRecord,
  FirstRechargePack,
  FulfillmentTask,
  Prize,
  ShopItem,
  UserStatus,
} from '@/types/api';

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
  { key: 'campaigns', label: '活动', icon: Package },
  { key: 'prizes', label: '礼品', icon: Gift },
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

export function AdminApp(): React.ReactNode {
  const [token, setToken] = useState('');
  const [tab, setTab] = useState<AdminTab>('overview');
  const [selectedCampaignId, setSelectedCampaignId] = useState('');
  const [userKeyword, setUserKeyword] = useState('');
  const [selectedUserId, setSelectedUserId] = useState('');
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
    queryFn: () => apiRequest<ShopItem[]>('/api/v1/shop/items', token),
    enabled: Boolean(token) && tab === 'shop',
  });

  const firstRechargeQuery = useQuery({
    queryKey: ['admin-first-recharge', token],
    queryFn: () => apiRequest<FirstRechargePack[]>('/api/v1/first-recharge/packs', token),
    enabled: Boolean(token) && tab === 'shop',
  });

  const createCampaignMutation = useMutation({
    mutationFn: () =>
      apiRequest('/api/v1/admin/campaigns', token, {
        method: 'POST',
        body: JSON.stringify({
          name: `运营活动 ${Date.now().toString().slice(-4)}`,
          slug: `ops-${Date.now()}`,
          status: 'online',
          starts_at: new Date().toISOString(),
          ends_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
          daily_draw_limit: 5,
          miss_weight: 70,
          campaign_summary: '管理端快速创建的对齐活动',
        }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-campaigns', token] });
      void queryClient.invalidateQueries({ queryKey: ['admin-overview', token] });
    },
  });

  const createPrizeMutation = useMutation({
    mutationFn: () =>
      apiRequest(`/api/v1/admin/campaigns/${selectedCampaignId}/prizes`, token, {
        method: 'POST',
        body: JSON.stringify({ name: '新礼品', level: 'common', stock: 100, probability_weight: 10, status: 'active' }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-prizes', token, selectedCampaignId] });
    },
  });

  const savePityMutation = useMutation({
    mutationFn: () =>
      apiRequest(`/api/v1/admin/campaigns/${selectedCampaignId}/pity-config`, token, {
        method: 'PUT',
        body: JSON.stringify({
          enabled: true,
          soft_pity_n: 20,
          pity_factor: 0.08,
          hard_pity_n: 60,
          target_prize: '',
          up_pool_enabled: true,
          up_multiplier: 3,
          up_level: 'secret',
        }),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin-campaigns', token] });
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
                ['活动数', overviewQuery.data?.campaigns.length ?? 0],
              ].map(([label, value]) => (
                <div className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-4 text-center" key={label}>
                  <div className="text-3xl font-black text-orange-400">{value}</div>
                  <div className="mt-1 text-xs text-zinc-500">{label}</div>
                </div>
              ))}
            </div>
            <AdminList title="活动列表">
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
              <h2 className="text-xl font-black text-white">活动管理</h2>
              <button
                className="rounded-md bg-orange-500 px-3 py-2 text-xs font-bold text-white disabled:opacity-50"
                disabled={createCampaignMutation.isPending}
                onClick={() => createCampaignMutation.mutate()}
                type="button"
              >
                + 新建活动
              </button>
            </div>
            {(campaignsQuery.data ?? []).map((campaign) => (
              <DataCard
                action={
                  <button
                    className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs text-zinc-300"
                    onClick={() => {
                      setSelectedCampaignId(campaign.id);
                      setTab('prizes');
                    }}
                    type="button"
                  >
                    礼品
                  </button>
                }
                badge={campaign.status}
                badgeClass={statusClass(campaign.status)}
                key={campaign.id}
                subtitle={`${campaign.id} · 每日上限 ${campaign.daily_draw_limit} · 未中权重 ${campaign.miss_weight}`}
                title={campaign.name}
              />
            ))}
          </section>
        ) : null}

        {tab === 'prizes' ? (
          <section className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-xl font-black text-white">礼品管理</h2>
              <button
                className="rounded-md bg-orange-500 px-3 py-2 text-xs font-bold text-white disabled:opacity-50"
                disabled={!selectedCampaignId || createPrizeMutation.isPending}
                onClick={() => createPrizeMutation.mutate()}
                type="button"
              >
                + 新建礼品
              </button>
            </div>
            <select
              className="w-full rounded-lg border border-zinc-700 bg-[#222] px-3 py-3 text-sm text-zinc-100 outline-none focus:border-orange-500 md:max-w-md"
              onChange={(event) => setSelectedCampaignId(event.target.value)}
              value={selectedCampaignId}
            >
              <option value="">-- 选择活动 --</option>
              {(campaignsQuery.data ?? []).map((campaign) => (
                <option key={campaign.id} value={campaign.id}>
                  {campaign.name}
                </option>
              ))}
            </select>
            {selectedCampaignId ? (
              <div className="space-y-3">
                {(prizesQuery.data ?? []).map((prize) => (
                  <DataCard
                    badge={prize.level}
                    badgeClass="border-blue-500/30 bg-blue-500/10 text-blue-300"
                    key={prize.id}
                    subtitle={`ID: ${prize.id} · 库存 ${prize.stock} · 权重 ${prize.probability_weight} · ${prize.status}`}
                    title={prize.name}
                  />
                ))}
                {prizesQuery.data?.length === 0 ? <EmptyAdmin text="当前活动暂无礼品" /> : null}
              </div>
            ) : (
              <EmptyAdmin text="请先选择一个活动" />
            )}
          </section>
        ) : null}

        {tab === 'pity' ? (
          <section className="space-y-4">
            <h2 className="text-xl font-black text-white">概率 / UP 池配置</h2>
            <select
              className="w-full rounded-lg border border-zinc-700 bg-[#222] px-3 py-3 text-sm text-zinc-100 outline-none focus:border-orange-500 md:max-w-md"
              onChange={(event) => setSelectedCampaignId(event.target.value)}
              value={selectedCampaignId}
            >
              <option value="">-- 选择活动 --</option>
              {(campaignsQuery.data ?? []).map((campaign) => (
                <option key={campaign.id} value={campaign.id}>
                  {campaign.name}
                </option>
              ))}
            </select>
            <div className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-4">
              <h3 className="mb-2 font-bold text-orange-400">配置说明</h3>
              <p className="text-sm leading-6 text-zinc-400">
                当前提供一键保存默认软/硬保底和 UP 池配置，保持与参考后台的概率配置入口一致。
              </p>
              <button
                className="mt-3 rounded-md bg-orange-500 px-3 py-2 text-xs font-bold text-white disabled:opacity-50"
                disabled={!selectedCampaignId || savePityMutation.isPending}
                onClick={() => savePityMutation.mutate()}
                type="button"
              >
                保存默认保底配置
              </button>
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
            <h2 className="text-xl font-black text-white">商店管理</h2>
            <AdminList title="商品列表">
              {(shopQuery.data ?? []).map((item) => (
                <DataCard
                  badge={`${item.price_points}积分`}
                  badgeClass="border-amber-500/30 bg-amber-500/10 text-amber-300"
                  key={item.id}
                  subtitle={`${item.description} · 库存 ${item.stock < 0 ? '不限' : item.stock}`}
                  title={item.name}
                />
              ))}
            </AdminList>
            <AdminList title="首充礼包">
              {(firstRechargeQuery.data ?? []).map((pack) => (
                <DataCard key={pack.id} subtitle={pack.description} title={pack.name} />
              ))}
            </AdminList>
          </section>
        ) : null}
      </div>
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
}: {
  readonly title: string;
  readonly subtitle: string;
  readonly badge?: string;
  readonly badgeClass?: string;
  readonly action?: React.ReactNode;
}): React.ReactNode {
  return (
    <article className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="truncate font-semibold text-zinc-100">{title}</h3>
            {badge ? <span className={`rounded border px-2 py-0.5 text-[11px] font-bold ${badgeClass ?? ''}`}>{badge}</span> : null}
          </div>
          <p className="mt-1 text-xs text-zinc-500">{subtitle}</p>
        </div>
        {action}
      </div>
    </article>
  );
}

function EmptyAdmin({ text }: { readonly text: string }): React.ReactNode {
  return <div className="rounded-xl border border-zinc-800 bg-[#1a1a24] p-8 text-center text-sm text-zinc-500">{text}</div>;
}

