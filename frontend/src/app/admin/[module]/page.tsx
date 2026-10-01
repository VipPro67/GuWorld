"use client";

import Link from "next/link";
import { ShieldX } from "lucide-react";
import { useParams } from "next/navigation";
import { AdminView } from "@/components/AdminView";
import { canAccessAdmin, useAdminAccess } from "@/components/admin/AdminAccessContext";
import { findAdminModule } from "@/components/admin/navigation";

export default function AdminModulePage() {
  const params = useParams<{ module: string }>();
  const access = useAdminAccess();
  const navigationItem = findAdminModule(params.module);

  if (!navigationItem || !canAccessAdmin(access, navigationItem.permission)) {
    return (
      <section className="m-4 rounded-2xl border border-red-500/20 bg-red-500/5 p-8 text-center sm:m-8">
        <ShieldX className="mx-auto h-10 w-10 text-red-400" />
        <h1 className="mt-4 text-xl font-bold text-zinc-100">403 — Không có quyền truy cập</h1>
        <p className="mt-2 text-sm text-zinc-400">Bạn không có permission cần thiết để mở module này.</p>
        <Link href="/admin" className="mt-6 inline-flex rounded-lg border border-zinc-700 px-4 py-2 text-sm text-zinc-200 hover:bg-zinc-800">Về trang quản trị</Link>
      </section>
    );
  }

  return <AdminView key={navigationItem.key} authorized embedded initialSection={navigationItem.key} />;
}
