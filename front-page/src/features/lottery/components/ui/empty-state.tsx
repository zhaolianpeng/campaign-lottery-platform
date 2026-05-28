export function EmptyState({
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

export function SkeletonCards(): React.ReactNode {
  return (
    <div className="grid grid-cols-2 gap-3">
      {Array.from({ length: 4 }).map((_, index) => (
        <div className="h-28 animate-pulse rounded-2xl bg-white/[0.06]" key={index} />
      ))}
    </div>
  );
}
