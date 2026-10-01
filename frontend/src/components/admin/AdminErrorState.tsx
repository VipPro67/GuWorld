import type { ReactNode } from "react";
import { CircleAlert } from "lucide-react";
import { GameButton } from "@/components/ui";

export function AdminErrorState({ title = "Không thể tải dữ liệu", description, actionLabel = "Thử lại", onRetry, action }: { title?: string; description?: ReactNode; actionLabel?: string; onRetry?: () => void; action?: ReactNode }) {
  return <div role="alert" className="flex min-h-48 flex-col items-center justify-center rounded-xl border border-red-500/20 bg-red-500/5 px-5 py-10 text-center"><CircleAlert className="h-8 w-8 text-red-400" /><h3 className="mt-3 font-serif font-bold text-zinc-100">{title}</h3>{description && <div className="mt-1 max-w-lg text-sm text-zinc-400">{description}</div>}<div className="mt-4">{action ?? (onRetry && <GameButton type="button" variant="secondary" onClick={onRetry}>{actionLabel}</GameButton>)}</div></div>;
}
