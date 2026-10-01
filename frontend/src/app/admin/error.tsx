"use client";

import Link from "next/link";
import { useEffect } from "react";

export default function AdminError({ error, retry }: { error: Error & { digest?: string }; retry: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main className="flex min-h-screen items-center justify-center bg-[var(--bg-primary)] px-4">
      <section className="w-full max-w-md rounded-2xl border border-red-500/20 bg-zinc-950 p-7 text-center">
        <h1 className="text-xl font-bold text-zinc-100">Bảng quản trị gặp sự cố</h1>
        <p className="mt-2 text-sm text-zinc-400">Không thể hiển thị nội dung quản trị. Bạn có thể thử tải lại phần này.</p>
        <div className="mt-6 flex justify-center gap-3">
          <Link href="/" className="rounded-lg border border-zinc-700 px-4 py-2 text-sm text-zinc-200 hover:bg-zinc-800">Quay lại game</Link>
          <button type="button" onClick={retry} className="rounded-lg bg-amber-500 px-4 py-2 text-sm font-semibold text-zinc-950 hover:bg-amber-400">Thử lại</button>
        </div>
      </section>
    </main>
  );
}
