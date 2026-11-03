"use client";

import { useState } from "react";

const FEATURES = [
  ["character", "Nhân vật"], ["inventory", "Túi đồ"], ["cultivation", "Tu luyện"],
  ["refine", "Luyện Cổ"], ["shop", "Cửa hàng"], ["auction", "Chợ Tiên"],
  ["party", "Tổ đội"], ["sect", "Tông môn"], ["achievements", "Thành tựu"],
  ["leaderboard", "Xếp hạng"], ["premium-store", "Cát Bảo Các"],
  ["utilities", "Cài đặt & Tiện ích"], ["feedback", "Góp ý"],
] as const;

export function BasicFeatureNavigation({ worldConnected, onNavigate, onReconnect }: {
  worldConnected: boolean;
  onNavigate: (view: string) => void;
  onReconnect: () => void;
}) {
  const [open, setOpen] = useState(false);
  return (
    <nav aria-label="Chức năng cơ bản" className="relative z-50 flex shrink-0 items-center gap-2 border-b border-amber-200/20 bg-zinc-950 px-3 py-2 text-xs text-amber-100">
      <p role="status" className="min-w-0 flex-1">
        {worldConnected ? "Cập nhật trực tiếp đang gián đoạn." : "Thế giới chưa kết nối."}
        {" "}Bạn vẫn dùng được các chức năng cơ bản.
      </p>
      <button type="button" onClick={onReconnect} className="shrink-0 rounded border border-white/20 px-2 py-2">Kết nối lại</button>
      <button type="button" aria-expanded={open} aria-controls="basic-feature-menu" onClick={() => setOpen(!open)} className="shrink-0 rounded border border-amber-200/30 px-2 py-2">Chức năng</button>
      {open && (
        <div id="basic-feature-menu" className="absolute inset-x-0 top-full grid max-h-[65dvh] grid-cols-2 gap-2 overflow-y-auto border-b border-amber-200/20 bg-zinc-950 p-3 shadow-xl sm:grid-cols-4" onKeyDown={(event) => { if (event.key === "Escape") setOpen(false); }}>
          {FEATURES.map(([view, label]) => (
            <button key={view} type="button" onClick={() => { onNavigate(view); setOpen(false); }} className="rounded border border-white/10 px-3 py-3 text-left hover:bg-white/10 focus-visible:outline-amber-200">{label}</button>
          ))}
          <button type="button" onClick={() => { onNavigate("mmo-world"); setOpen(false); }} className="rounded border border-white/10 px-3 py-3 text-left hover:bg-white/10">Về thế giới</button>
        </div>
      )}
    </nav>
  );
}
