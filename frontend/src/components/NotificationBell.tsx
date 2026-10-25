"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Bell, CheckCheck, Package, Skull, ShoppingBag, Gavel, Coins, Info, ChevronDown, ChevronUp, X } from "lucide-react";
import { useGameStore, type PlayerNotification } from "@/store/useGameStore";

const TYPE_ICONS: Record<string, { icon: React.ReactNode; color: string }> = {
  gu_death: { icon: <Skull className="w-4 h-4" />, color: "text-red-400" },
  vital_gu_death: { icon: <Skull className="w-4 h-4" />, color: "text-red-400" },
  death_drop: { icon: <Skull className="w-4 h-4" />, color: "text-rose-400" },
  loot_gained: { icon: <Package className="w-4 h-4" />, color: "text-emerald-400" },
  combat_loot: { icon: <Package className="w-4 h-4" />, color: "text-emerald-400" },
  market_sold: { icon: <ShoppingBag className="w-4 h-4" />, color: "text-amber-400" },
  market_bought: { icon: <ShoppingBag className="w-4 h-4" />, color: "text-sky-400" },
  auction_bid: { icon: <Gavel className="w-4 h-4" />, color: "text-amber-400" },
  auction_won: { icon: <Gavel className="w-4 h-4" />, color: "text-emerald-400" },
  auction_sold: { icon: <Gavel className="w-4 h-4" />, color: "text-emerald-400" },
  auction_expired: { icon: <Gavel className="w-4 h-4" />, color: "text-zinc-400" },
  coins: { icon: <Coins className="w-4 h-4" />, color: "text-amber-400" },
};

function relativeTime(iso: string): string {
  if (!iso) return "";
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "";
  const diff = Math.max(0, Date.now() - then);
  const sec = Math.floor(diff / 1000);
  if (sec < 60) return "vừa xong";
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min} phút trước`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr} giờ trước`;
  const day = Math.floor(hr / 24);
  return `${day} ngày trước`;
}

const NotificationItem: React.FC<{ notification: PlayerNotification }> = ({ notification: n }) => {
  const { markNotificationRead } = useGameStore();
  const [expanded, setExpanded] = useState(false);
  const meta = TYPE_ICONS[n.type] || { icon: <Info className="w-4 h-4" />, color: "text-zinc-400" };
  const isLong = (n.content || "").length > 80 || (n.content || "").includes("\n");

  const handleToggle = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!n.is_read) {
      void markNotificationRead(n.id);
    }
    setExpanded((prev) => !prev);
  };

  const handleClick = () => {
    if (!n.is_read) {
      void markNotificationRead(n.id);
    }
  };

  return (
    <div
      onClick={handleClick}
      className={`w-full text-left px-4 py-3 flex items-start gap-3 hover:bg-[var(--bg-elevated)] transition-colors border-b border-[var(--border-default)] last:border-b-0 cursor-pointer ${
        n.is_read ? "opacity-60" : ""
      }`}
    >
      <span className={`mt-0.5 shrink-0 ${meta.color}`}>{meta.icon}</span>
      <div className="flex-1 min-w-0">
        <span className="block text-xs font-semibold text-[var(--text-primary)]">{n.title}</span>
        <div
          className={`text-[11px] text-zinc-300 mt-1 leading-relaxed ${
            expanded ? "whitespace-pre-wrap break-words" : "line-clamp-2"
          }`}
        >
          {n.content}
        </div>
        <div className="flex items-center justify-between mt-1.5">
          <span className="text-[10px] text-zinc-500">{relativeTime(n.created_at)}</span>
          {isLong && (
            <button
              type="button"
              onClick={handleToggle}
              className="inline-flex items-center gap-0.5 text-[10px] font-medium text-emerald-400 hover:text-emerald-300 transition-colors py-0.5 px-1.5 rounded bg-emerald-500/10 hover:bg-emerald-500/20 cursor-pointer"
            >
              {expanded ? (
                <>
                  Thu gọn <ChevronUp className="w-3 h-3" />
                </>
              ) : (
                <>
                  Xem thêm <ChevronDown className="w-3 h-3" />
                </>
              )}
            </button>
          )}
        </div>
      </div>
      {!n.is_read && <span className="mt-1 shrink-0 w-2 h-2 rounded-full bg-emerald-500" />}
    </div>
  );
};

export const NotificationBell: React.FC<{ size?: "sm" | "md" }> = ({ size = "md" }) => {
  const { notifications, unreadNotifications, fetchNotifications, fetchUnreadNotificationsCount, markAllNotificationsRead } = useGameStore();
  const [open, setOpen] = useState(false);
  const [mounted, setMounted] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const modalRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (open) {
      void fetchNotifications();
      void fetchUnreadNotificationsCount();
    }
  }, [open, fetchNotifications, fetchUnreadNotificationsCount]);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      const target = e.target as Node;
      if (
        ref.current &&
        !ref.current.contains(target) &&
        modalRef.current &&
        !modalRef.current.contains(target)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  const isSmall = size === "sm";

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen((o) => !o)}
        className={`relative rounded-xl bg-[#1e1712]/90 hover:bg-[#2c2016] text-[#e8d7b3] hover:text-[#fff0cd] transition-colors border border-[#a88952]/60 flex items-center justify-center cursor-pointer shadow-md backdrop-blur-md ${
          isSmall ? "w-7 h-7 sm:w-8 sm:h-8" : "w-9 h-9"
        }`}
        aria-label="Thông báo"
        title="Thông báo"
      >
        <Bell className={`${isSmall ? "w-3.5 h-3.5" : "w-4 h-4"} text-amber-300`} />
        {unreadNotifications > 0 && (
          <span
            className={`absolute -top-1 -right-1 rounded-full bg-red-600 text-white font-bold flex items-center justify-center shadow leading-none ${
              isSmall ? "min-w-[15px] h-[15px] px-0.5 text-[8.5px]" : "min-w-[18px] h-[18px] px-1 text-[10px]"
            }`}
          >
            {unreadNotifications > 99 ? "99+" : unreadNotifications}
          </span>
        )}
      </button>

      {open && mounted && createPortal(
        <div className="fixed inset-0 z-[9999] pointer-events-none">
          <div
            className="pointer-events-auto absolute inset-0 bg-black/40 sm:bg-transparent"
            onClick={() => setOpen(false)}
          />
          <div
            ref={modalRef}
            className="pointer-events-auto fixed top-14 left-4 right-4 sm:left-4 sm:right-auto sm:w-[380px] max-w-[calc(100vw-2rem)] max-h-[75dvh] flex flex-col rounded-2xl border border-[#a88952]/80 bg-[#16100bfd] text-[#f4e8cb] shadow-[0_20px_50px_rgba(0,0,0,.9)] backdrop-blur-md animate-in fade-in zoom-in-95 duration-150 overflow-hidden font-serif"
          >
            <div className="flex items-center justify-between px-4 py-3 border-b border-[#a88952]/30 bg-[#251a13]/90 rounded-t-2xl">
              <div className="flex items-center gap-2">
                <Bell className="w-4 h-4 text-amber-300" />
                <span className="text-sm font-serif font-bold text-[#ffeac0]">Thông báo ({notifications.length})</span>
              </div>
              <div className="flex items-center gap-3">
                <button
                  onClick={markAllNotificationsRead}
                  className="inline-flex items-center gap-1 text-xs font-serif text-emerald-400 hover:text-emerald-300 transition-colors cursor-pointer"
                >
                  <CheckCheck className="w-3.5 h-3.5" />
                  Đã đọc hết
                </button>
                <button
                  type="button"
                  onClick={() => setOpen(false)}
                  className="p-1 rounded-lg hover:bg-white/10 text-zinc-400 hover:text-zinc-200 transition cursor-pointer"
                  title="Đóng bảng thông báo"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            </div>

            <div className="flex-1 overflow-y-auto divide-y divide-[#a88952]/15 custom-scrollbar">
              {notifications.length === 0 ? (
                <div className="px-4 py-8 text-center text-xs text-zinc-400 font-serif italic">Chưa có thông báo nào.</div>
              ) : (
                notifications.map((n: PlayerNotification) => (
                  <NotificationItem key={n.id} notification={n} />
                ))
              )}
            </div>
          </div>
        </div>,
        document.body
      )}
    </div>
  );
};
