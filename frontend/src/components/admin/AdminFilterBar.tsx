import type { ReactNode } from "react";
import { Search, X } from "lucide-react";
import { GameButton, GameInput } from "@/components/ui";

export interface AdminFilterBarProps {
  search?: string;
  onSearchChange?: (value: string) => void;
  searchPlaceholder?: string;
  children?: ReactNode;
  actions?: ReactNode;
  active?: boolean;
  onReset?: () => void;
}

export function AdminFilterBar({ search, onSearchChange, searchPlaceholder = "Tìm kiếm…", children, actions, active, onReset }: AdminFilterBarProps) {
  return (
    <section aria-label="Bộ lọc" className="flex flex-col gap-2 rounded-xl border border-zinc-800 bg-zinc-900/40 p-3 lg:flex-row lg:items-center">
      {onSearchChange && <div className="min-w-0 flex-1 lg:max-w-md"><GameInput aria-label="Tìm kiếm" value={search ?? ""} onChange={(event) => onSearchChange(event.target.value)} placeholder={searchPlaceholder} leftIcon={<Search className="h-3.5 w-3.5" />} size="sm" /></div>}
      {children && <div className="flex flex-1 flex-wrap items-center gap-2">{children}</div>}
      {(onReset || actions) && <div className="flex flex-wrap items-center gap-2 lg:ml-auto">{onReset && <GameButton type="button" variant="ghost" size="sm" disabled={!active} onClick={onReset} leftIcon={<X className="h-3.5 w-3.5" />}>Xóa lọc</GameButton>}{actions}</div>}
    </section>
  );
}
