"use client";

import React, { useCallback, useEffect, useState } from "react";
import { Gem, Coins, Sparkles, RefreshCw } from "lucide-react";
import { authFetch } from "@/lib/auth";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

export interface WalletBalance {
  primevalStone: number;
  immortalEssenceStone: number;
  immortalJade: number;
}

interface PremiumWalletProps {
  compact?: boolean;
  onLoaded?: (wallet: WalletBalance) => void;
  refreshKey?: number;
}

export const PremiumWallet: React.FC<PremiumWalletProps> = ({ compact = false, onLoaded, refreshKey = 0 }) => {
  const [wallet, setWallet] = useState<WalletBalance | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchWallet = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await authFetch(`${BACKEND_URL}/api/wallet`);
      if (response.ok) {
        const data = await response.json();
        setWallet(data);
        onLoaded?.(data);
      } else {
        setError("Không thể tải ví");
      }
    } catch {
      setError("Lỗi kết nối máy chủ");
    } finally {
      setLoading(false);
    }
  }, [onLoaded]);

  useEffect(() => {
    const timer = window.setTimeout(() => { void fetchWallet(); }, 0);
    return () => window.clearTimeout(timer);
  }, [fetchWallet, refreshKey]);

  if (!wallet) {
    return (
      <div className="flex items-center justify-between rounded-2xl border border-zinc-800 bg-zinc-950/60 px-4 py-3 text-xs text-zinc-500">
        <span>{error ? error : "Đang tải ví..."}</span>
        {error && (
          <button
            onClick={() => { void fetchWallet(); }}
            disabled={loading}
            className="flex items-center gap-1.5 rounded-lg border border-amber-500/40 bg-amber-950/30 px-2.5 py-1 text-xs text-amber-300 hover:bg-amber-900/40 disabled:opacity-50"
          >
            <RefreshCw className={`h-3 w-3 ${loading ? "animate-spin" : ""}`} /> Thử lại
          </button>
        )}
      </div>
    );
  }

  const itemClass = compact
    ? "flex items-center gap-1.5 rounded-xl border border-zinc-800 bg-zinc-950/50 px-3 py-1.5 text-xs font-mono"
    : "flex items-center justify-between rounded-2xl border border-zinc-800 bg-zinc-950/50 px-4 py-3";

  return (
    <div className={compact ? "flex flex-wrap items-center gap-2" : "grid grid-cols-1 gap-3 md:grid-cols-3"}>
      <div className={`${itemClass} text-amber-300`}>
        <span className="flex items-center gap-1.5"><Coins className="h-4 w-4" /> Nguyên Thạch</span>
        <b>{wallet.primevalStone.toLocaleString()}</b>
      </div>
      <div className={`${itemClass} text-violet-300`}>
        <span className="flex items-center gap-1.5"><Sparkles className="h-4 w-4" /> Tiên Nguyên Thạch</span>
        <b>{wallet.immortalEssenceStone.toLocaleString()}</b>
      </div>
      <div className={`${itemClass} text-emerald-300`}>
        <span className="flex items-center gap-1.5"><Gem className="h-4 w-4" /> Tiên Ngọc</span>
        <b>{wallet.immortalJade.toLocaleString()}</b>
      </div>
      {!compact && (
        <button onClick={fetchWallet} disabled={loading} className="md:col-span-3 flex w-fit items-center gap-2 rounded-xl border border-emerald-700/40 bg-emerald-950/30 px-3 py-2 text-xs text-emerald-300 hover:bg-emerald-900/40 disabled:opacity-60">
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} /> Làm mới ví
        </button>
      )}
    </div>
  );
};
