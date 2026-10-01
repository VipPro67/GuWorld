"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAdminAccess, canAccessAdmin } from "@/components/admin/AdminAccessContext";
import { ADMIN_NAVIGATION } from "@/components/admin/navigation";

export default function AdminPage() {
  const router = useRouter();
  const access = useAdminAccess();
  const firstModule = ADMIN_NAVIGATION
    .filter((item) => canAccessAdmin(access, item.permission))
    .sort((left, right) => left.order - right.order)[0];

  useEffect(() => {
    if (firstModule) router.replace(firstModule.route);
  }, [firstModule, router]);

  return <div className="p-8 text-sm text-zinc-500">{firstModule ? "Đang mở module quản trị…" : "Không có module nào được cấp quyền."}</div>;
}
