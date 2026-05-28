'use client';

import { useMemo, useState } from 'react';
import type { BlendResult, RedeemResult, UserInventory } from '@/types/api';
import { levelMeta, PrizeMedia } from '../rarity';
import { EmptyState, SkeletonCards } from './ui/empty-state';
import { Modal } from './ui/modal';

export function InventoryTabPanel({
  items,
  prizeImageUrlById,
  isLoading,
  viewMode,
  onViewModeChange,
  onBlend,
  onRedeem,
  blendPending,
  redeemPending,
}: {
  readonly items: readonly UserInventory[] | undefined;
  readonly prizeImageUrlById: ReadonlyMap<string, string>;
  readonly isLoading: boolean;
  readonly viewMode: 'list' | 'grouped';
  onViewModeChange: (mode: 'list' | 'grouped') => void;
  onBlend: (prizeId: string, campaignId: string) => Promise<BlendResult>;
  onRedeem: (prizeId: string) => Promise<RedeemResult>;
  readonly blendPending: boolean;
  readonly redeemPending: boolean;
}): React.ReactNode {
  const [blendTarget, setBlendTarget] = useState<UserInventory | null>(null);
  const [blendResult, setBlendResult] = useState<BlendResult | null>(null);

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
        <h2 className="text-lg font-black text-white">我的收藏</h2>
        <div className="flex gap-2">
          <button
            className={`rounded-full px-3 py-1.5 text-xs ${viewMode === 'list' ? 'bg-violet-500 text-white' : 'border border-white/15 bg-white/10'}`}
            onClick={() => onViewModeChange('list')}
            type="button"
          >
            列表
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
              <div className="mb-2 text-xs font-bold text-violet-200/80">系列 {group.campaign_id.slice(0, 8)}</div>
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
                      onRedeem={() => void onRedeem(item.prize_id).then(() => window.alert('兑换成功'))}
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
                onRedeem={() => void onRedeem(item.prize_id).then(() => window.alert('兑换成功'))}
                prizeImageUrl={prizeImageUrlById.get(item.prize_id)}
                redeemPending={redeemPending}
                showBlend={count >= 3}
              />
            );
          })}
        </div>
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
          兑换
        </button>
      </div>
    </div>
  );
}
