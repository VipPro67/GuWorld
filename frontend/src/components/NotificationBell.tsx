"use client";

import { useEffect, useRef, useState } from "react";
import { Bell, CheckCheck, Package, Skull, ShoppingBag, Gavel, Coins, Info, ChevronDown, ChevronUp } from "lucide-react";
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

export const NotificationBell: React.FC = () => {
  const { notifications, unreadNotifications, fetchNotifications, fetchUnreadNotificationsCount, markAllNotificationsRead } = useGameStore();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (open) {
      void fetchNotifications();
      void fetchUnreadNotificationsCount();
    }
  }, [open, fetchNotifications, fetchUnreadNotificationsCount]);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen((o) => !o)}
        className="relative w-9 h-9 rounded-xl bg-[var(--bg-surface)] hover:bg-[var(--bg-elevated)] text-zinc-400 hover:text-zinc-200 transition-colors border border-[var(--border-default)] flex items-center justify-center cursor-pointer"
        aria-label="Thông báo"
      >
        <Bell className="w-4 h-4" />
        {unreadNotifications > 0 && (
          <span className="absolute -top-1 -right-1 min-w-[18px] h-[18px] px-1 rounded-full bg-red-500 text-white text-[10px] font-bold flex items-center justify-center">
            {unreadNotifications > 99 ? "99+" : unreadNotifications}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 top-full z-[70] mt-2 flex max-h-[70dvh] w-[calc(100vw-2rem)] max-w-[380px] flex-col rounded-2xl border border-[var(--border-default)] bg-[var(--bg-surface)] shadow-2xl">
          <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border-default)]">
            <span className="text-sm font-semibold text-[var(--text-primary)]">Thông báo</span>
            <button
              onClick={markAllNotificationsRead}
              className="inline-flex items-center gap-1 text-xs text-emerald-500 hover:text-emerald-400 transition-colors cursor-pointer"
            >
              <CheckCheck className="w-3.5 h-3.5" />
              Đánh dấu đã đọc
            </button>
          </div>

          <div className="flex-1 overflow-y-auto">
            {notifications.length === 0 ? (
              <div className="px-4 py-8 text-center text-xs text-zinc-500">Chưa có thông báo nào.</div>
            ) : (
              notifications.map((n: PlayerNotification) => (
                <NotificationItem key={n.id} notification={n} />
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
};
