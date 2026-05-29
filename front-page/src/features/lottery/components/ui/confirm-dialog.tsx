import { Modal } from './modal';

export function ConfirmDialog({
  title,
  message,
  confirmLabel = '确认',
  cancelLabel = '取消',
  onConfirm,
  onCancel,
}: {
  readonly title: string;
  readonly message: string;
  readonly confirmLabel?: string;
  readonly cancelLabel?: string;
  readonly onConfirm: () => void;
  readonly onCancel: () => void;
}): React.ReactNode {
  return (
    <Modal onClose={onCancel} title={title}>
      <p className="text-sm text-violet-100/75">{message}</p>
      <div className="mt-4 grid grid-cols-2 gap-2">
        <button className="rounded-2xl border border-white/15 py-2.5 text-sm font-semibold text-white" onClick={onCancel} type="button">
          {cancelLabel}
        </button>
        <button className="rounded-2xl bg-violet-500 py-2.5 text-sm font-bold text-white" onClick={onConfirm} type="button">
          {confirmLabel}
        </button>
      </div>
    </Modal>
  );
}
