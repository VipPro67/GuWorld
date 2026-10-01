import { LoaderCircle } from "lucide-react";

export function AdminLoadingState({ label = "Đang tải dữ liệu…", compact = false }: { label?: string; compact?: boolean }) {
  return <div role="status" aria-live="polite" aria-busy="true" className={`flex items-center justify-center gap-2 text-sm text-zinc-500 ${compact ? "py-5" : "min-h-48 py-10"}`}><LoaderCircle aria-hidden="true" className="h-5 w-5 animate-spin text-amber-400" /><span>{label}</span></div>;
}
