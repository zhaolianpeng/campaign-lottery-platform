import { apiAssetUrl } from '@/client/api';

export const rarityStyles: Record<string, { readonly icon: string; readonly label: string; readonly className: string }> = {
  common: { icon: '□', label: '普通', className: 'text-slate-300' },
  rare: { icon: '◆', label: '稀有', className: 'text-sky-300' },
  secret: { icon: '✦', label: '隐藏', className: 'text-violet-300' },
  limited: { icon: '★', label: '限定', className: 'text-amber-300' },
  S: { icon: '★', label: 'S', className: 'text-amber-300' },
  A: { icon: '◆', label: 'A', className: 'text-sky-300' },
  B: { icon: '□', label: 'B', className: 'text-slate-300' },
};

export function levelMeta(level?: string): { readonly icon: string; readonly label: string; readonly className: string } {
  return rarityStyles[level ?? ''] ?? { icon: '□', label: level ?? '未知', className: 'text-slate-300' };
}

export function PrizeMedia({
  imageUrl,
  name,
  meta,
  imageClassName,
  fallbackClassName,
}: {
  readonly imageUrl?: string;
  readonly name: string;
  readonly meta: { readonly icon: string; readonly className: string };
  readonly imageClassName: string;
  readonly fallbackClassName: string;
}): React.ReactNode {
  if (imageUrl) {
    return <img alt={name} className={imageClassName} src={apiAssetUrl(imageUrl)} />;
  }
  return <div className={`${fallbackClassName} ${meta.className}`}>{meta.icon}</div>;
}
