import type { ReactNode } from "react";
import { LockKeyhole } from "lucide-react";

export interface AdminPermissionGateProps {
  allowed: boolean;
  children: ReactNode;
  fallback?: ReactNode;
  hidden?: boolean;
}

export function AdminPermissionGate({ allowed, children, fallback, hidden = false }: AdminPermissionGateProps) {
  if (allowed) return <>{children}</>;
  if (hidden) return null;
  return <>{fallback ?? <div role="note" className="flex items-center gap-2 rounded-xl border border-zinc-800 bg-zinc-900/40 px-4 py-3 text-xs text-zinc-500"><LockKeyhole className="h-4 w-4" />Bạn không có quyền thực hiện thao tác này.</div>}</>;
}
