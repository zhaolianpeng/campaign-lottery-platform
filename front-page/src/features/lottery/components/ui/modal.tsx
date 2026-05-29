import { X } from 'lucide-react';
import type { ReactNode } from 'react';

export function Modal({
  title,
  children,
  onClose,
  wide,
}: {
  readonly title: string;
  readonly children: ReactNode;
  readonly onClose: () => void;
  readonly wide?: boolean;
}): React.ReactNode {
  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/70 p-3 sm:items-center" onClick={onClose} role="presentation">
      <div
        className={`max-h-[88vh] w-full overflow-y-auto rounded-3xl border border-white/10 bg-[#141828] p-4 shadow-2xl ${wide ? 'max-w-lg' : 'max-w-md'}`}
        onClick={(event) => event.stopPropagation()}
        role="dialog"
      >
        <div className="mb-3 flex items-center justify-between gap-2">
          <h2 className="text-lg font-black text-white">{title}</h2>
          <button aria-label="关闭" className="rounded-full border border-white/15 p-1.5 text-violet-100" onClick={onClose} type="button">
            <X size={18} />
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}
