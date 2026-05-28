import type { PityStatus } from '@/types/api';
import { Modal } from './ui/modal';

export function ProbabilitySheet({
  campaignName,
  prizes,
  pityConfig,
  disclosureUpdatedAt,
  filingNumber,
  rulesText,
  pityStatus,
  onClose,
}: {
  readonly campaignName: string;
  readonly prizes: readonly { readonly name: string; readonly level: string; readonly base_prob?: string }[];
  readonly pityConfig?: { readonly enabled: boolean; readonly soft_pity_n: number; readonly hard_pity_n: number } | null;
  readonly disclosureUpdatedAt?: string;
  readonly filingNumber?: string;
  readonly rulesText?: string;
  readonly pityStatus?: PityStatus;
  readonly onClose: () => void;
}): React.ReactNode {
  return (
    <Modal onClose={onClose} title={`${campaignName} · 概率公示`} wide>
      <p className="text-xs text-violet-100/60">
        概率信息最后更新：{disclosureUpdatedAt ? new Date(disclosureUpdatedAt).toLocaleString('zh-CN') : '—'}
        {filingNumber ? ` · 备案号：${filingNumber}` : ''}
      </p>
      <div className="mt-3 max-h-48 space-y-2 overflow-y-auto text-sm">
        {prizes.map((prize) => (
          <div className="flex justify-between gap-2 border-b border-white/5 py-1" key={prize.name + prize.level}>
            <span className="text-violet-100/80">{prize.name}</span>
            <span className="shrink-0 text-violet-100/55">{prize.base_prob ?? prize.level}</span>
          </div>
        ))}
      </div>
      {pityConfig?.enabled ? (
        <p className="mt-3 rounded-xl bg-white/[0.06] p-3 text-xs text-violet-100/70">
          软保底 {pityConfig.soft_pity_n} 次起概率提升 · 硬保底 {pityConfig.hard_pity_n} 次必出稀有及以上
        </p>
      ) : null}
      {pityStatus ? (
        <p className="mt-2 text-xs text-amber-200/90">
          当前连续未中稀有：{pityStatus.consecutive_misses} 次 · 距硬保底 {pityStatus.misses_to_hard_pity} 次
        </p>
      ) : null}
      <p className="mt-3 text-xs leading-relaxed text-violet-100/55">{rulesText ?? '抽奖结果由系统随机生成，请理性消费。'}</p>
    </Modal>
  );
}
