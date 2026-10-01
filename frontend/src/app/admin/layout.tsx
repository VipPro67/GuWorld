"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { ArrowLeft, LayoutDashboard, RefreshCw, ShieldCheck, UserRound } from "lucide-react";
import {
  AdminAccessProvider,
  type AdminAccess,
} from "@/components/admin/AdminAccessContext";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

type AccessState =
  | { status: "loading" }
  | { status: "ready"; access: AdminAccess }
  | { status: "unauthorized" }
  | { status: "forbidden" }
  | { status: "error"; message: string };

function AccessMessage({
  title,
  description,
  retry,
}: {
  title: string;
  description: string;
  retry?: () => void;
}) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-[var(--bg-primary)] px-4">
      <section className="w-full max-w-md rounded-2xl border border-zinc-800 bg-zinc-950/80 p-7 text-center shadow-2xl">
        <ShieldCheck className="mx-auto mb-4 h-10 w-10 text-amber-400" />
        <h1 className="text-xl font-bold text-zinc-100">{title}</h1>
        <p className="mt-2 text-sm leading-6 text-zinc-400">{description}</p>
        <div className="mt-6 flex justify-center gap-3">
          <Link href="/" className="inline-flex items-center gap-2 rounded-lg border border-zinc-700 px-4 py-2 text-sm text-zinc-200 transition hover:bg-zinc-800">
            <ArrowLeft className="h-4 w-4" /> Quay lại game
          </Link>
          {retry && (
            <button type="button" onClick={retry} className="inline-flex items-center gap-2 rounded-lg bg-amber-500 px-4 py-2 text-sm font-semibold text-zinc-950 transition hover:bg-amber-400">
              <RefreshCw className="h-4 w-4" /> Thử lại
            </button>
          )}
        </div>
      </section>
    </main>
  );
}

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AccessState>({ status: "loading" });
  const activeRequest = useRef<AbortController | null>(null);

  const resolveAccess = useCallback(async () => {
    activeRequest.current?.abort();
    const controller = new AbortController();
    activeRequest.current = controller;
    setState({ status: "loading" });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/admin/me`, { signal: controller.signal });
      if (controller.signal.aborted) return;
      if (response.status === 401) {
        setState({ status: "unauthorized" });
        return;
      }
      if (response.status === 403) {
        setState({ status: "forbidden" });
        return;
      }
      if (!response.ok) {
        setState({ status: "error", message: `Máy chủ trả về mã ${response.status}.` });
        return;
      }
      const access = (await response.json()) as Partial<AdminAccess>;
      if (
        typeof access.actor_id !== "string" ||
        typeof access.username !== "string" ||
        !Array.isArray(access.roles) ||
        !access.roles.every((role) => typeof role === "string") ||
        !Array.isArray(access.effective_permissions) ||
        !access.effective_permissions.every((permission) => typeof permission === "string") ||
        typeof access.legacy_fallback !== "boolean" ||
        typeof access.permission_version !== "string"
      ) {
        setState({ status: "error", message: "Máy chủ trả về thông tin quyền không hợp lệ." });
        return;
      }
      setState({ status: "ready", access: access as AdminAccess });
    } catch {
      if (controller.signal.aborted) return;
      setState({ status: "error", message: "Không thể kết nối tới máy chủ quản trị." });
    }
  }, []);

  useEffect(() => {
    void resolveAccess();
    return () => activeRequest.current?.abort();
  }, [resolveAccess]);

  if (state.status === "loading") {
    return (
      <main className="flex min-h-screen items-center justify-center bg-[var(--bg-primary)]" aria-busy="true">
        <div className="flex items-center gap-3 text-sm font-mono text-zinc-400">
          <RefreshCw className="h-5 w-5 animate-spin text-amber-400" /> Đang xác minh quyền quản trị…
        </div>
      </main>
    );
  }
  if (state.status === "unauthorized") {
    return <AccessMessage title="Chưa đăng nhập" description="Hãy đăng nhập trong game trước khi mở bảng quản trị." />;
  }
  if (state.status === "forbidden") {
    return <AccessMessage title="Không có quyền truy cập" description="Tài khoản hiện tại không được cấp quyền quản trị." />;
  }
  if (state.status === "error") {
    return <AccessMessage title="Không thể tải quyền quản trị" description={state.message} retry={() => void resolveAccess()} />;
  }

  const { access } = state;
  return (
    <AdminAccessProvider value={access}>
      <div className="min-h-screen bg-[var(--bg-primary)] text-zinc-100">
        <header className="sticky top-0 z-30 border-b border-zinc-800 bg-zinc-950/90 backdrop-blur">
          <div className="mx-auto flex max-w-[1600px] items-center justify-between gap-4 px-4 py-3 sm:px-6">
            <div className="flex min-w-0 items-center gap-3">
              <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 p-2"><ShieldCheck className="h-5 w-5 text-amber-400" /></div>
              <div className="min-w-0">
                <p className="truncate font-serif font-bold">Quản trị Cổ Giới</p>
                <p className="truncate text-xs text-zinc-500">Không gian vận hành độc lập</p>
              </div>
            </div>
            <Link href="/" className="inline-flex shrink-0 items-center gap-2 rounded-lg border border-zinc-700 px-3 py-2 text-xs font-semibold text-zinc-300 transition hover:border-zinc-600 hover:bg-zinc-800 hover:text-white">
              <ArrowLeft className="h-4 w-4" /> <span className="hidden sm:inline">Quay lại game</span>
            </Link>
          </div>
        </header>

        <div className="mx-auto grid max-w-[1600px] lg:grid-cols-[240px_minmax(0,1fr)]">
          <aside className="border-b border-zinc-800 bg-zinc-950/40 p-4 lg:min-h-[calc(100vh-65px)] lg:border-b-0 lg:border-r">
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
              <div className="flex items-center gap-2 text-sm font-semibold"><UserRound className="h-4 w-4 text-amber-400" /> {access.username}</div>
              <p className="mt-1 break-all text-[11px] font-mono text-zinc-500">{access.actor_id}</p>
              <div className="mt-3 flex flex-wrap gap-1.5">
                {access.roles.map((role) => <span key={role} className="rounded bg-amber-500/10 px-2 py-1 text-[10px] font-semibold uppercase text-amber-400">{role}</span>)}
              </div>
              <p className="mt-3 text-xs text-zinc-500">{access.effective_permissions.length} quyền hiệu lực</p>
            </div>
            <nav aria-label="Điều hướng quản trị" className="mt-4">
              <Link href="/admin" aria-current="page" className="flex items-center gap-2 rounded-lg bg-amber-500/10 px-3 py-2.5 text-sm font-semibold text-amber-400">
                <LayoutDashboard className="h-4 w-4" /> Bảng điều khiển
              </Link>
            </nav>
          </aside>
          <main className="min-w-0">{children}</main>
        </div>
      </div>
    </AdminAccessProvider>
  );
}
