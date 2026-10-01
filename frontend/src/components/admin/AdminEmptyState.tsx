import type { ReactNode } from "react";
import { Inbox } from "lucide-react";

export function AdminEmptyState({ title = "Chưa có dữ liệu", description, icon, action, compact = false }: { title?: string; description?: ReactNode; icon?: ReactNode; action?: ReactNode; compact?: boolean }) {
  return <div className={`flex flex-col items-center justify-center px-5 text-center ${compact ? "py-6" : "min-h-48 py-10"}`}>{icon ?? <Inbox className="h-8 w-8 text-zinc-600" />}<h3 className="mt-3 font-serif font-bold text-zinc-300">{title}</h3>{description && <div className="mt-1 max-w-lg text-sm text-zinc-500">{description}</div>}{action && <div className="mt-4">{action}</div>}</div>;
}
