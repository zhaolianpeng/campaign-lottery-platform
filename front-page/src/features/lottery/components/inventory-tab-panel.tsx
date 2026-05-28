'use client';

import { useEffect, useMemo, useState } from 'react';
import type { BlendResult, RedeemResult, UserInventory } from '@/types/api';
import { levelMeta, PrizeMedia } from '../rarity';
import { EmptyState, SkeletonCards } from './ui/empty-state';
import { Modal } from './ui/modal';

const FREE_SHIPPING_THRESHOLD_YUAN = 98;
const SHIPPING_FEE_CENTS = 990;

export function InventoryTabPanel({
  items,
  prizeImageUrlById,
  campaignNameById,
  isLoading,
  viewMode,
  onViewModeChange,
  onBlend,
  onRedeem,
  onSubmitDelivery,
  blendPending,
  deliveryPending,
  paymentEnabled,
  redeemPending,
}: {
  readonly items: readonly UserInventory[] | undefined;
  readonly prizeImageUrlById: ReadonlyMap<string, string>;
  readonly campaignNameById: ReadonlyMap<string, string>;
  readonly isLoading: boolean;
  readonly viewMode: 'list' | 'grouped';
  onViewModeChange: (mode: 'list' | 'grouped') => void;
  onBlend: (prizeId: string, campaignId: string) => Promise<BlendResult>;
  onRedeem: (prizeId: string) => Promise<RedeemResult>;
  onSubmitDelivery: (itemIds: readonly string[]) => Promise<void>;
  readonly blendPending: boolean;
  readonly deliveryPending: boolean;
  readonly paymentEnabled: boolean;
  readonly redeemPending: boolean;
}): React.ReactNode {
  const [blendTarget, setBlendTarget] = useState<UserInventory | null>(null);
  const [blendResult, setBlendResult] = useState<BlendResult | null>(null);
  const [selectedItemIds, setSelectedItemIds] = useState<string[]>([]);

  const grouped = useMemo(() => {
    const map = new Map<string, { readonly campaign_id: string; readonly items: Map<string, { readonly item: UserInventory; count: number }> }>();
    for (const item of items ?? []) {
      const bucket = map.get(item.campaign_id) ?? { campaign_id: item.campaign_id, items: new Map() };
      const row = bucket.items.get(item.prize_id) ?? { item, count: 0 };
      bucket.items.set(item.prize_id, { item: row.item, count: row.count + 1 });
      map.set(item.campaign_id, bucket);
    }
    return [...map.values()];
  }, [items]);

  const countsByPrize = useMemo(() => {
    const m = new Map<string, number>();
    for (const item of items ?? []) {
      m.set(item.prize_id, (m.get(item.prize_id) ?? 0) + 1);
    }
    return m;
  }, [items]);

  const detailItems = useMemo(() => [...(items ?? [])].sort((left, right) => right.created_at.localeCompare(left.created_at)), [items]);

  const deliverableIds = useMemo(
    () => new Set((items ?? []).filter((item) => item.delivery_status === 'not_requested' && !item.exchange_offer_id).map((item) => item.id)),
    [items],
  );

  useEffect(() => {
    setSelectedItemIds((current) => current.filter((itemId) => deliverableIds.has(itemId)));
  }, [deliverableIds]);

  const selectedItems = useMemo(
    () => detailItems.filter((item) => selectedItemIds.includes(item.id) && item.delivery_status === 'not_requested'),
    [detailItems, selectedItemIds],
  );

  const deliverySubtotalYuan = useMemo(
    () => Number(selectedItems.reduce((sum, item) => sum + item.shipping_value_yuan, 0).toFixed(2)),
    [selectedItems],
  );

  const shippingFeeCents = selectedItems.length === 0 ? 0 : deliverySubtotalYuan >= FREE_SHIPPING_THRESHOLD_YUAN ? 0 : SHIPPING_FEE_CENTS;

  function toggleItemSelection(itemId: string): void {
    setSelectedItemIds((current) => (current.includes(itemId) ? current.filter((id) => id !== itemId) : [...current, itemId]));
  }

  async function handleSubmitDelivery(): Promise<void> {
    if (!selectedItems.length) {
      return;
    }
    if (shippingFeeCents > 0 && !paymentEnabled) {
      window.alert('当前支付功能未启用，暂时无法支付运费');
      return;
    }
    const confirmed = window.confirm(
      shippingFeeCents > 0
        ? `本次发货奖品总额 ${deliverySubtotalYuan.toFixed(2)} 元，需支付 9.9 元运费。确认继续？`
        : `本次发货奖品总额 ${deliverySubtotalYuan.toFixed(2)} 元，已满足包邮条件，确认提交发货申请？`,
    );
    if (!confirmed) {
      return;
    }
    await onSubmitDelivery(selectedItems.map((item) => item.id));
    setSelectedItemIds([]);
  }

  async function handleBlend(): Promise<void> {
    if (!blendTarget) {
      return;
    }
    const result = await onBlend(blendTarget.prize_id, blendTarget.campaign_id);
    setBlendResult(result);
  }

  return (
    <section className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h2 className="text-lg font-black text-white">我的图鉴</h2>
          <p className="mt-1 text-xs text-violet-100/60">按系列查看已收集款式，也可以切换到按奖品浏览。</p>
        </div>
        <div className="flex gap-2">
          <button
            className={`rounded-full px-3 py-1.5 text-xs ${viewMode === 'list' ? 'bg-violet-500 text-white' : 'border border-white/15 bg-white/10'}`}
            onClick={() => onViewModeChange('list')}
            type="button"
          >
            按奖品
          </button>
          <button
            className={`rounded-full px-3 py-1.5 text-xs ${viewMode === 'grouped' ? 'bg-violet-500 text-white' : 'border border-white/15 bg-white/10'}`}
            onClick={() => onViewModeChange('grouped')}
            type="button"
          >
            按系列
          </button>
          <span className="rounded-full border border-white/15 bg-white/10 px-3 py-1.5 text-xs">共 {items?.length ?? 0} 件</span>
        </div>
      </div>
      {isLoading ? <SkeletonCards /> : null}
      {viewMode === 'grouped' && grouped.length > 0 ? (
        <div className="space-y-4">
          {grouped.map((group) => (
            <div className="rounded-2xl border border-white/10 bg-white/[0.04] p-3" key={group.campaign_id}>
              <div className="mb-2 text-xs font-bold text-violet-200/80">
                {campaignNameById.get(group.campaign_id) ?? `系列 ${group.campaign_id.slice(0, 8)}`}
              </div>
              <div className="grid grid-cols-3 gap-2">
                {[...group.items.values()].map(({ item, count }) => {
                  const meta = levelMeta(item.prize_level);
                  return (
                    <InventoryCard
                      count={count}
                      item={item}
                      key={item.id}
                      meta={meta}
                      onBlend={() => setBlendTarget(item)}
                      onRedeem={() => void onRedeem(item.prize_id).then(() => window.alert('积分兑换成功'))}
                      prizeImageUrl={prizeImageUrlById.get(item.prize_id)}
                      redeemPending={redeemPending}
                      showBlend={count >= 3}
                    />
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      ) : null}
      {viewMode === 'list' && items?.length ? (
        <div className="grid grid-cols-3 gap-2">
          {items.map((item) => {
            const meta = levelMeta(item.prize_level);
            const count = countsByPrize.get(item.prize_id) ?? 1;
            return (
              <InventoryCard
                count={count}
                item={item}
                key={item.id}
                meta={meta}
                onBlend={() => setBlendTarget(item)}
                onRedeem={() => void onRedeem(item.prize_id).then(() => window.alert('积分兑换成功'))}
                prizeImageUrl={prizeImageUrlById.get(item.prize_id)}
                redeemPending={redeemPending}
                showBlend={count >= 3}
              />
            );
          })}
        </div>
      ) : null}

      {items?.length ? (
        <section className="space-y-3 rounded-3xl border border-white/10 bg-white/[0.04] p-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h3 className="text-base font-black text-white">我抽到的奖品明细</h3>
              <p className="mt-1 text-xs text-violet-100/60">奖品默认状态为不发货，勾选后才会进入发货流程。</p>
            </div>
            <div className="flex gap-2 text-xs">
              <span className="rounded-full border border-white/15 bg-white/8 px-3 py-1.5 text-violet-100/80">可发货 {deliverableIds.size} 件</span>
              <span className="rounded-full border border-white/15 bg-white/8 px-3 py-1.5 text-violet-100/80">已选 {selectedItems.length} 件</span>
            </div>
          </div>

          <div className="space-y-2">
            {detailItems.map((item) => {
              const meta = levelMeta(item.prize_level);
              const selectable = item.delivery_status === 'not_requested' && !item.exchange_offer_id;
              return (
                <label
                  className={`flex items-center gap-3 rounded-2xl border px-3 py-3 ${
                    selectable ? 'border-white/10 bg-white/[0.03]' : 'border-white/8 bg-white/[0.02] opacity-90'
                  }`}
                  key={item.id}
                >
                  <input
                    checked={selectedItemIds.includes(item.id)}
                    className="h-4 w-4 accent-violet-400"
                    disabled={!selectable || deliveryPending}
                    onChange={() => toggleItemSelection(item.id)}
                    type="checkbox"
                  />
                  <PrizeMedia
                    fallbackClassName="flex h-14 w-14 items-center justify-center text-2xl"
                    imageClassName="h-14 w-14 rounded-xl border border-white/10 object-cover"
                    imageUrl={prizeImageUrlById.get(item.prize_id)}
                    meta={meta}
                    name={item.prize_name}
                  />
                  <div className="min-w-0 flex-1">
                    <div className={`line-clamp-1 text-sm font-bold ${meta.className}`}>{item.prize_name}</div>
                    <div className="mt-1 flex flex-wrap gap-2 text-[11px] text-violet-100/55">
                      <span>{meta.label}</span>
                      <span>{sourceLabel(item.source)}</span>
                      <span>{formatInventoryTime(item.created_at)}</span>
                    </div>
                    <div className="mt-2 flex flex-wrap gap-2 text-[11px]">
                      <span className="rounded-full border border-amber-300/20 bg-amber-400/10 px-2 py-1 text-amber-100">
                        计价 {item.shipping_value_yuan.toFixed(2)} 元
                      </span>
                      <span className={`rounded-full border px-2 py-1 ${deliveryStatusClass(item.delivery_status)}`}>{deliveryStatusLabel(item.delivery_status)}</span>
                      {item.exchange_offer_id ? <span className="rounded-full border border-fuchsia-300/25 bg-fuchsia-400/10 px-2 py-1 text-fuchsia-100">交换中</span> : null}
                    </div>
                  </div>
                </label>
              );
            })}
          </div>

          <div className="rounded-2xl border border-white/10 bg-[linear-gradient(135deg,rgba(167,139,250,0.14),rgba(15,23,42,0.65))] p-4">
            <div className="flex flex-wrap items-center justify-between gap-2 text-sm text-violet-100/80">
              <span>奖品总额 {deliverySubtotalYuan.toFixed(2)} 元</span>
              <span>{shippingFeeCents === 0 ? '已满足满 98 包邮' : '未满 98 元，需支付 9.9 元运费'}</span>
            </div>
            {!paymentEnabled && shippingFeeCents > 0 ? <p className="mt-2 text-xs text-amber-200">当前支付未开启，暂时只能提交满足包邮条件的奖品。</p> : null}
            <div className="mt-3 flex flex-wrap gap-2">
              <button
                className="rounded-full border border-white/15 px-4 py-2 text-xs text-violet-100 disabled:opacity-50"
                disabled={deliveryPending || selectedItemIds.length === 0}
                onClick={() => setSelectedItemIds([])}
                type="button"
              >
                清空勾选
              </button>
              <button
                className="rounded-full bg-[linear-gradient(135deg,#f97316,#fb7185)] px-4 py-2 text-xs font-bold text-white disabled:opacity-50"
                disabled={deliveryPending || selectedItems.length === 0 || (shippingFeeCents > 0 && !paymentEnabled)}
                onClick={() => void handleSubmitDelivery()}
                type="button"
              >
                {deliveryPending ? '提交中...' : shippingFeeCents === 0 ? '提交发货申请' : '支付运费并提交发货'}
              </button>
            </div>
          </div>
        </section>
      ) : null}

      {!isLoading && !items?.length ? <EmptyState description="还没有收集到任何盲盒款式，去抽盒吧。" icon="□" title="暂无收藏" /> : null}

      {blendTarget ? (
        <Modal
          onClose={() => {
            setBlendTarget(null);
            setBlendResult(null);
          }}
          title="合成升级"
        >
          {!blendResult ? (
            <>
              <p className="text-sm text-violet-100/70">将多个「{blendTarget.prize_name}」合成为更高等级款式（需满足数量）。</p>
              <button
                className="mt-4 w-full rounded-2xl bg-emerald-500 py-3 font-bold text-white disabled:opacity-50"
                disabled={blendPending}
                onClick={() => void handleBlend()}
                type="button"
              >
                确认合成
              </button>
            </>
          ) : (
            <p className="text-sm text-emerald-200">
              合成成功：{blendResult.result_prize_name}（{blendResult.result_level}），剩余原料 {blendResult.remaining_src} 个
            </p>
          )}
        </Modal>
      ) : null}
    </section>
  );
}

function deliveryStatusLabel(status: UserInventory['delivery_status']): string {
  if (status === 'pending_payment') {
    return '待支付运费';
  }
  if (status === 'pending_fulfillment') {
    return '待发货';
  }
  if (status === 'fulfilled') {
    return '已发货';
  }
  return '不发货';
}

function deliveryStatusClass(status: UserInventory['delivery_status']): string {
  if (status === 'pending_payment') {
    return 'border-amber-300/25 bg-amber-400/10 text-amber-100';
  }
  if (status === 'pending_fulfillment') {
    return 'border-sky-300/25 bg-sky-400/10 text-sky-100';
  }
  if (status === 'fulfilled') {
    return 'border-emerald-300/25 bg-emerald-400/10 text-emerald-100';
  }
  return 'border-white/10 bg-white/[0.04] text-violet-100/70';
}

function sourceLabel(source: UserInventory['source']): string {
  if (source === 'draw') {
    return '抽盒获得';
  }
  if (source === 'exchange') {
    return '交换/赠送';
  }
  if (source === 'redeem') {
    return '积分兑换';
  }
  return '收藏奖励';
}

function formatInventoryTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function InventoryCard({
  item,
  meta,
  prizeImageUrl,
  count,
  showBlend,
  onBlend,
  onRedeem,
  redeemPending,
}: {
  readonly item: UserInventory;
  readonly meta: ReturnType<typeof levelMeta>;
  readonly prizeImageUrl?: string;
  readonly count: number;
  readonly showBlend: boolean;
  readonly onBlend: () => void;
  readonly onRedeem: () => void;
  readonly redeemPending: boolean;
}): React.ReactNode {
  return (
    <div className="relative rounded-2xl border border-violet-300/40 bg-white/[0.06] p-2 text-center">
      {count > 1 ? (
        <span className="absolute right-1 top-1 rounded-full bg-pink-500 px-1.5 text-[10px] font-bold text-white">×{count}</span>
      ) : null}
      <PrizeMedia
        fallbackClassName="mx-auto flex h-14 w-14 items-center justify-center text-2xl"
        imageClassName="mx-auto h-14 w-14 rounded-xl border border-white/10 object-cover"
        imageUrl={prizeImageUrl}
        meta={meta}
        name={item.prize_name}
      />
      <div className={`mt-1 line-clamp-1 text-[11px] font-semibold ${meta.className}`}>{item.prize_name}</div>
      <div className="mt-2 flex flex-col gap-1">
        {showBlend ? (
          <button className="rounded-lg bg-emerald-500/90 py-1 text-[10px] font-bold text-white" onClick={onBlend} type="button">
            合成
          </button>
        ) : null}
        <button className="rounded-lg border border-white/15 py-1 text-[10px] text-violet-100" disabled={redeemPending} onClick={onRedeem} type="button">
          积分兑换
        </button>
      </div>
    </div>
  );
}
