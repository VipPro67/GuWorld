import type { ReactNode } from "react";

export interface AdminPageHeaderProps {
  title: string;
  description?: ReactNode;
  eyebrow?: ReactNode;
  icon?: ReactNode;
  actions?: ReactNode;
}

export function AdminPageHeader({ title, description, eyebrow, icon, actions }: AdminPageHeaderProps) {
  return (
    <header className="flex flex-col gap-4 border-b border-zinc-800 pb-5 sm:flex-row sm:items-start sm:justify-between">
      <div className="flex min-w-0 gap-3">
        {icon && <div className="mt-0.5 rounded-xl border border-amber-500/20 bg-amber-500/10 p-2 text-amber-400">{icon}</div>}
        <div className="min-w-0">
          {eyebrow && <div className="mb-1 text-[10px] font-bold uppercase tracking-[0.18em] text-amber-500/80">{eyebrow}</div>}
          <h1 className="font-serif text-xl font-bold text-zinc-100 sm:text-2xl">{title}</h1>
          {description && <div className="mt-1 max-w-3xl text-sm leading-6 text-zinc-400">{description}</div>}
        </div>
      </div>
      {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
    </header>
  );
}
